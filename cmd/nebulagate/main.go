package main

import (
	"log"
	"net/http"

	"github.com/SaisrikarVollala/nebulagate/internal/balancer"
	"github.com/SaisrikarVollala/nebulagate/internal/config"
	"github.com/SaisrikarVollala/nebulagate/internal/middleware"
)

func main() {
	// Step 1: Load backend servers from the JSON config file
	servers, err := config.LoadServers("config/servers.json")
	if err != nil {
		log.Fatalf("Failed to load server config: %v", err)
	}

	log.Printf("Loaded %d backend servers", len(servers))
	for _, s := range servers {
		log.Printf("  → %s (%s)", s.ID, s.URL)
	}

	// Step 2: Create the Round Robin load balancer
	lb, err := balancer.NewLoadBalancer(servers)
	if err != nil {
		log.Fatalf("Failed to create load balancer: %v", err)
	}

	// Step 3: Start the HTTP server — all incoming requests go through the load balancer
	addr := ":8080"
	log.Printf("nebulaGate load balancer started on %s", addr)
	log.Fatal(http.ListenAndServe(addr, lb))
}
