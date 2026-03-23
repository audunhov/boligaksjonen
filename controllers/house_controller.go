package controllers

import (
	"encoding/json"
	"net/http"
	"html/template"
	"log/slog"

	"github.com/audunhov/tombolig/models"
)

// IndexHandler serves the main HTML page containing the map.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("views/index.html")
	if err != nil {
		slog.Error("Failed to parse template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		slog.Error("Failed to execute template", "error", err)
	}
}

// APIHousesHandler retrieves house data and returns it as JSON.
func APIHousesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	houses := models.GetAllHouses()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(houses); err != nil {
		slog.Error("Failed to encode JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
