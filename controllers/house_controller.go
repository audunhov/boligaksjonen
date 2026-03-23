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

// EditHandler serves the edit form for a house.
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

// UpdateHandler handles the POST request to update a house.
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lng, _ := strconv.ParseFloat(r.FormValue("lng"), 64)

	ownershipType := r.FormValue("ownership_type")
	if ownershipType == "freetext" {
		ownershipType = r.FormValue("ownership_freetext")
	}

	author := r.FormValue("author")
	if author == "" {
		author = "Anonymous"
	}

	updatedHouse := models.House{
		ID:            id,
		Address:       r.FormValue("address"),
		Latitude:      lat,
		Longitude:     lng,
		Description:   r.FormValue("description"),
		OwnershipType: ownershipType,
		LastUpdatedBy: author,
		UpdatedAt:     time.Now(),
	}

	if err := models.UpdateHouse(updatedHouse); err != nil {
		slog.Error("Failed to update house", "error", err)
		http.Error(w, "Failed to update house", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
