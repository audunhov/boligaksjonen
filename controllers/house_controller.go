package controllers

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/audunhov/tombolig/models"
)

// HomeHandler serves the landing page.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("views/home.html")
	if err != nil {
		slog.Error("Failed to parse home template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		slog.Error("Failed to execute home template", "error", err)
	}
}

// MapHandler serves the interactive map page.
func MapHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("views/index.html")
	if err != nil {
		slog.Error("Failed to parse map template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		slog.Error("Failed to execute map template", "error", err)
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
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	if err := json.NewEncoder(w).Encode(houses); err != nil {
		slog.Error("Failed to encode JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// EditHandler serves the edit form for a house (legacy or if we want a direct page).
func EditHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid house ID", http.StatusBadRequest)
		return
	}

	house, err := models.GetHouseByID(id)
	if err != nil {
		http.Error(w, "House not found", http.StatusNotFound)
		return
	}

	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006 15:04")
		},
	}

	tmpl, err := template.New("edit.html").Funcs(funcMap).ParseFiles("views/edit.html")
	if err != nil {
		slog.Error("Failed to parse template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, house)
	if err != nil {
		slog.Error("Failed to execute template", "error", err)
	}
}

// UpdateHandler handles the POST request to update or add a house.
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse form", "error", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, _ := strconv.Atoi(idStr) // id 0 means new house

	lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
	if err != nil {
		slog.Error("Invalid latitude", "val", r.FormValue("lat"), "error", err)
		http.Error(w, "Invalid latitude", http.StatusBadRequest)
		return
	}

	lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
	if err != nil {
		slog.Error("Invalid longitude", "val", r.FormValue("lng"), "error", err)
		http.Error(w, "Invalid longitude", http.StatusBadRequest)
		return
	}

	ownershipType := r.FormValue("ownership_type")
	if ownershipType == "freetext" {
		ownershipType = r.FormValue("ownership_freetext")
	}

	author := r.FormValue("author")
	if author == "" {
		author = "Anonymous"
	}

	house := models.House{
		ID:            id,
		Address:       r.FormValue("address"),
		Latitude:      lat,
		Longitude:     lng,
		Description:   r.FormValue("description"),
		OwnershipType: ownershipType,
		LastUpdatedBy: author,
		UpdatedAt:     time.Now(),
	}

	if id == 0 {
		slog.Info("Adding new house", "address", house.Address)
		models.AddHouse(house)
	} else {
		slog.Info("Updating house", "id", id, "address", house.Address)
		if err := models.UpdateHouse(house); err != nil {
			slog.Error("Failed to update house in model", "id", id, "error", err)
			http.Error(w, "Failed to update house: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Redirect back to the map
	http.Redirect(w, r, "/kart", http.StatusSeeOther)
}
