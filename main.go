package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/audunhov/tombolig/controllers"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Register routes
	http.HandleFunc("/", controllers.IndexHandler)
	http.HandleFunc("/api/houses", controllers.APIHousesHandler)

	// Start server
	port := ":8080"
	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
