package dixhttp

import (
	"log"

	"github.com/pubgo/dix/v2"
)

// Example demonstrates how to use the HTTP server for dependency visualization
func Example(di *dix.Dix) {
	// Create HTTP server
	server := NewServer(di)

	// Start the server
	log.Println("Starting HTTP server on :8080")
	log.Println("Open http://localhost:8080 in your browser to view dependencies")
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatal("Server error:", err)
	}
}
