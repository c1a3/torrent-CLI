package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("╔══════════════════════════════════════╗")
	log.Println("║         Starting Server...   	    ║")
	log.Println("╚══════════════════════════════════════╝")

	// Determine port (Render injects PORT env var)
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	// Initialize WebSocket hub
	hub := NewHub()
	go hub.Run()

	// Initialize torrent manager
	tm, err := NewTorrentManager(hub)
	if err != nil {
		log.Fatalf("Failed to initialize torrent manager: %v", err)
	}

	// Set up HTTP routes
	mux := http.NewServeMux()
	SetupRoutes(mux, tm, hub)

	// Configure HTTP server
	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute, // Allow long writes for file downloads
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on http://0.0.0.0:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown on SIGTERM/SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Give in-flight requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Close the torrent client
	tm.Close()

	log.Println("Server stopped.")
}

// loggingMiddleware logs incoming HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Skip logging for WebSocket upgrades and static assets
		isAPI := len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api"
		isWS := r.URL.Path == "/ws"

		next.ServeHTTP(w, r)

		if isAPI || isWS {
			log.Printf("[http] %s %s %s (%s)", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start).Round(time.Millisecond))
		}
	})
}
