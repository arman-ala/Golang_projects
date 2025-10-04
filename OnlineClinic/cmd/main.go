// ./cmd/main.go

package main

import (
	// For formatted I/O operations
	// For logging errors and informational messages

	"net/http"  // For handling HTTP requests and responses
	"os"        // For interacting with the operating system
	"os/signal" // For receiving OS signals
	"syscall"   // For system call constants (e.g., SIGINT, SIGTERM)
	"time"      // For time-related operations (e.g., timeouts)

	"onlineClinic/config" // Custom package for loading configuration and database connection
	"onlineClinic/routes" // Custom package for setting up application routes

	"github.com/gorilla/mux" // Gorilla Mux router for handling HTTP routes
)

func main() {
	// Load the application configuration from the config package.
	// This typically involves reading environment variables or a configuration file.
	config.LoadConfig()

	// Establish a connection to the database using the config package.
	// The defer statement ensures the database connection is closed when the program exits.
	config.ConnectDB()
	defer config.DB.Close()

	// Create necessary directories for storing uploaded files.
	// These directories are used for profile pictures and chat-related files.
	os.MkdirAll("uploads/profile", 0755) // Create "uploads/profile" directory with permissions 0755
	os.MkdirAll("uploads/chat", 0755)    // Create "uploads/chat" directory with permissions 0755

	// Initialize a new Gorilla Mux router for handling HTTP requests.
	router := mux.NewRouter()

	// Set up all the application routes by calling the SetupRoutes function from the routes package.
	// This function returns an http.Handler that includes all the defined routes.
	handler := routes.SetupRoutes(router)

	// Serve static files from the "uploads" directory.
	// This allows clients to access files stored in the "uploads" directory via HTTP.
	fs := http.FileServer(http.Dir("uploads"))                                // Create a file server for the "uploads" directory
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", fs)) // Strip "/uploads/" prefix and serve files

	// Configure the HTTP server with longer timeouts for file uploads
	server := &http.Server{
		Addr:         ":8080",           // Listen on port 8080
		Handler:      handler,           // Use the handler returned by SetupRoutes
		ReadTimeout:  300 * time.Second, // 5 minutes for reading request (increased for file uploads)
		WriteTimeout: 300 * time.Second, // 5 minutes for writing response (increased for file uploads)
		IdleTimeout:  120 * time.Second, // 2 minutes for idle connections
		// Add these for better file upload handling
		ReadHeaderTimeout: 60 * time.Second, // 1 minute to read headers
	}

	// Start the HTTP server in a goroutine to allow it to run concurrently with other operations.
	go func() {
		// fmt.Printf("Server running on https://online-clinic.liara.run%s\n", server.Addr) // Print server address to the console
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed { // Start the server
			// log.Fatalf("Server failed to start: %v", err) // Log a fatal error if the server fails to start
		}
	}()

	// Set up a channel to listen for OS signals such as SIGINT (Ctrl+C) or SIGTERM.
	quit := make(chan os.Signal, 1)                      // Create a buffered channel to receive signals
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Notify the channel of the specified signals

	// Wait for a signal to be received, indicating that the server should shut down gracefully.
	<-quit
	// log.Println("Server is shutting down...") // Log a message indicating the server is shutting down

	// Close all WebSocket connections managed by the hub in the routes package.
	for client := range routes.Hub.Clients {
		client.Conn.Close() // Close each WebSocket connection
	}

	// Close the database connection to release resources.
	config.DB.Close()

	// Log a message indicating that the server has stopped.
	// log.Println("Server stopped")
}
