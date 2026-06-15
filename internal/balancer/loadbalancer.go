package balancer

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/SaisrikarVollala/nebulagate/internal/server"
)

// LoadBalancer distributes incoming HTTP requests across backend servers
// using the Round Robin algorithm.
type LoadBalancer struct {
	servers []*server.Server
	current uint64 // atomic counter for round-robin index
}

// NewLoadBalancer creates a new LoadBalancer with the given servers.
// It initializes the reverse proxy for each server.
func NewLoadBalancer(servers []*server.Server) (*LoadBalancer, error) {
	for _, s := range servers {
		if err := s.SetupProxy(); err != nil {
			return nil, fmt.Errorf("failed to setup proxy for server %s: %w", s.ID, err)
		}
	}

	return &LoadBalancer{
		servers: servers,
		current: 0,
	}, nil
}

// getNextServer returns the next alive server using Round Robin selection.
//
// How it works:
//  1. Atomically increment the counter to get a unique index per request.
//  2. Use modulo (%) to wrap the index within the server list bounds.
//  3. If the selected server is alive, return it.
//  4. If not, keep advancing (up to a full rotation) to find an alive server.
//  5. If all servers are down, return nil.
func (lb *LoadBalancer) getNextServer() *server.Server {
	total := uint64(len(lb.servers))

	// Try each server at most once (full rotation)
	for i := uint64(0); i < total; i++ {
		// Atomically increment and get the next counter value.
		// atomic.AddUint64 is safe for concurrent goroutines —
		// no two requests will ever get the same counter value.
		next := atomic.AddUint64(&lb.current, 1)

		// Modulo wraps the counter back to 0 when it exceeds the server count.
		// e.g., with 3 servers: 1%3=1, 2%3=2, 3%3=0, 4%3=1, ...
		idx := (next - 1) % total

		if lb.servers[idx].Alive {
			return lb.servers[idx]
		}
	}

	// All servers are down
	return nil
}

// ServeHTTP implements the http.Handler interface.
// This makes LoadBalancer usable directly as an HTTP server handler.
// For each incoming request, it picks the next server and forwards the request.
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv := lb.getNextServer()
	if srv == nil {
		http.Error(w, "Service Unavailable: all backend servers are down", http.StatusServiceUnavailable)
		return
	}

	log.Printf("Forwarding request %s %s → %s (%s)", r.Method, r.URL.Path, srv.URL, srv.ID)

	// Delegate to the server's reverse proxy, which handles:
	// - Forwarding the request (method, headers, body) to the backend
	// - Streaming the response back to the client
	srv.ReverseProxy.ServeHTTP(w, r)
}
