package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/audunhov/tombolig/controllers"
	"github.com/audunhov/tombolig/models"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize database
	if err := models.InitDB("tombolig.db"); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Register routes
	http.HandleFunc("/", controllers.HomeHandler)
	http.HandleFunc("/kart", controllers.MapHandler)
	http.HandleFunc("/kart/historikk", controllers.HistoryHandler)
	http.HandleFunc("/api/houses", controllers.APIHousesHandler)
	http.HandleFunc("/api/houses/history", controllers.APIHouseHistoryHandler)
	http.HandleFunc("/api/houses/comment", controllers.AddCommentHandler)
	http.HandleFunc("/houses/edit", controllers.EditHandler)
	http.HandleFunc("/houses/update", controllers.UpdateHandler)
	http.HandleFunc("/houses/remove", controllers.RemoveHandler)
	http.HandleFunc("/houses/restore", controllers.RestoreHandler)
	http.HandleFunc("/signup", controllers.SignupHandler)
	http.HandleFunc("/login", controllers.LoginHandler)
	http.HandleFunc("/logout", controllers.LogoutHandler)

	// Start server
	port := ":8080"
	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
