package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/audunhov/tombolig/controllers"
	"github.com/audunhov/tombolig/models"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize database
	if err := models.InitDB("tombolig.db"); err != nil {
		slog.Error("Database initialization failed", "error", err)
		return
	}

	// Register routes
	http.HandleFunc("/", controllers.HomeHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/kart", controllers.MapHandler)
	http.HandleFunc("/kart/historikk", controllers.HistoryHandler)
	http.HandleFunc("/api/houses", controllers.APIHousesHandler)
	http.HandleFunc("/api/houses/history", controllers.APIHouseHistoryHandler)
	http.HandleFunc("/api/address/lookup", controllers.APIAddressLookupHandler)
	http.HandleFunc("/houses/edit", controllers.HouseEditHandler)
	http.HandleFunc("/houses/details", controllers.HouseDetailsHandler)
	http.HandleFunc("/api/houses/comment", controllers.AddCommentHandler)
	http.HandleFunc("/api/auth/check-username", controllers.CheckUsernameHandler)
	http.HandleFunc("/houses/update", controllers.UpdateHandler)
	http.HandleFunc("/houses/remove", controllers.RemoveHandler)
	http.HandleFunc("/houses/restore", controllers.RestoreHandler)
	http.HandleFunc("/signup", controllers.SignupHandler)
	http.HandleFunc("/login", controllers.LoginHandler)
	http.HandleFunc("/logout", controllers.LogoutHandler)

	// Start background cleanup worker
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			count, err := models.PermanentlyDeleteOldHouses()
			if err != nil {
				slog.Error("Background cleanup failed", "error", err)
			} else if count > 0 {
				slog.Info("Background cleanup successful", "deleted_count", count)
			}
		}
	}()

	// Start server
	port := ":8080"
	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server failed", "error", err)
	}
}
