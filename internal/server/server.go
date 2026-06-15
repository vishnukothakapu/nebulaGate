package server

import (
	"net/http/httputil"
	"net/url"
)

// Server represents a backend server that can receive proxied requests.
type Server struct {
	ID           string                 `json:"id"`
	URL          string                 `json:"url"`
	Alive        bool                   `json:"-"`
	ReverseProxy *httputil.ReverseProxy `json:"-"`
}

// SetupProxy parses the server URL and creates a reverse proxy for it.
// This must be called after loading the server from config.
func (s *Server) SetupProxy() error {
	targetURL, err := url.Parse(s.URL)
	if err != nil {
		return err
	}

	s.ReverseProxy = httputil.NewSingleHostReverseProxy(targetURL)
	return nil
}
