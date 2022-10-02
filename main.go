package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

const (
	Attempts int = iota
	Retry
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

// SetAlive func for this backend
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

// IsAlive returns true when backend is alive
func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

// GetAttemptsFromContext returns the attempts for request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetRetryFromContext returns the attempts for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

// lb func load balances the incoming requests
func lb(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// IsBackendAlive checks whether a backend is alive by establishing a TCP connection
func IsBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	_ = conn.Close() // close it, we don't need to maintain this connection
	return true
}