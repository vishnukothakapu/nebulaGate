package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SaisrikarVollala/nebulagate/internal/balancer"
	"github.com/SaisrikarVollala/nebulagate/internal/config"
	"github.com/SaisrikarVollala/nebulagate/internal/health"
	"github.com/SaisrikarVollala/nebulagate/internal/metrics"
	"github.com/SaisrikarVollala/nebulagate/internal/middleware"
)

func main() {

	// Load backend servers
	servers, err := config.LoadServers("config/servers.json")
	if err != nil {
		log.Fatalf("failed to load server config: %v", err)
	}

	// Initial health check
	for _, s := range servers {
		s.Alive = health.CheckServer(s)

		status := "DOWN"
		if s.Alive {
			status = "UP"
		}

		log.Printf("→ %s (%s) [%s]", s.ID, s.URL, status)
	}

	log.Printf("Loaded %d backend servers", len(servers))

	// Start background health monitoring
	go health.StartHealthChecker(
		servers,
		10*time.Second,
	)

	// Create load balancer
	lb, err := balancer.NewLoadBalancer(servers)
	if err != nil {
		log.Fatalf("failed to create load balancer: %v", err)
	}

	// Create router
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metrics.NewHandler(servers))
	mux.Handle("/", lb)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: middleware.Recovery(mux),
	}

	// Start NebulaGate
	go func() {
		log.Printf("NebulaGate listening on http://localhost%s", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)

	signal.Notify(
		sigChan,
		os.Interrupt,
		syscall.SIGTERM,
	)

	sig := <-sigChan

	log.Printf("received signal: %v", sig)
	log.Println("starting graceful shutdown...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("NebulaGate stopped gracefully")
}
