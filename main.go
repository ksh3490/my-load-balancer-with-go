package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*Backend
	current uint64
}

u, _ := url.Parse("http://localhost:8080")
rp := httputil.NewSingleHostReverseProxy(u)

// Initialize your server and add this as handler
http.HandlerFunc(rp.ServeHTTP)

// Increase the current value by one atomically and Return the index by modding with the slice length
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1))) % uint64(len(s.backends))
}

// GetNextPeer will return next active peer to take a connection
func (s *ServerPool) GetNextPeer() *Backend {
	 // Loop entire backends to find out on Alive Backend
	next := s.NextIndex()

	// Start from next and move a full cycle
	l := len(s.backends) + next 
	for i := next; i < l; i++ {
		idx := i % len(s.backends) // Take an index by modding with slice length
		// if you have an alive backend, use it and store if it's not the original one
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx)) // Mark the current one
			}
			return s.backends[idx]
		}
	}
	return nil
}