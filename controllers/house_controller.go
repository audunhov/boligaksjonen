package controllers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/audunhov/tombolig/models"
)

// Helper to get anonymous fingerprint
func getFingerprint(r *http.Request) string {
	ip := r.RemoteAddr
	ua := r.UserAgent()
	hash := sha256.Sum256([]byte(ip + ua))
	return fmt.Sprintf("%x", hash)[:8]
}

// Helper to get logged in user ID from session
func getLoggedInUserID(r *http.Request) *int {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return nil
	}
	id, err := strconv.Atoi(cookie.Value)
	if err != nil {
		return nil
	}
	return &id
}

// Helper to parse int from form value
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

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
	
	data := struct {
		UserID *int
	}{
		UserID: getLoggedInUserID(r),
	}
	tmpl.Execute(w, data)
}

// MapHandler serves the interactive map page.
func MapHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("views/index.html")
	if err != nil {
		slog.Error("Failed to parse map template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	var username string
	userID := getLoggedInUserID(r)
	if userID != nil {
		user, err := models.GetUserByID(*userID)
		if err == nil {
			username = user.Username
		}
	}

	data := struct {
		UserID   *int
		Username string
	}{
		UserID:   userID,
		Username: username,
	}
	tmpl.Execute(w, data)
}

// APIHousesHandler retrieves house data and returns it as JSON.
func APIHousesHandler(w http.ResponseWriter, r *http.Request) {
	houses := models.GetAllHouses()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	json.NewEncoder(w).Encode(houses)
}

// UpdateHandler handles the POST request to update or add a house.
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lng, _ := strconv.ParseFloat(r.FormValue("lng"), 64)

	ownershipType := r.FormValue("ownership_type")
	if ownershipType == "freetext" {
		ownershipType = r.FormValue("ownership_freetext")
	}

	var author string
	anonHash := getFingerprint(r)
	userID := getLoggedInUserID(r)
	
	if userID != nil {
		user, err := models.GetUserByID(*userID)
		if err == nil {
			author = user.Username
		} else {
			author = fmt.Sprintf("Anonym (%s)", anonHash)
		}
	} else {
		author = fmt.Sprintf("Anonym (%s)", anonHash)
	}

	house := models.House{
		ID:            id,
		Address:       r.FormValue("address"),
		Latitude:      lat,
		Longitude:     lng,
		Description:   r.FormValue("description"),
		OwnershipType: ownershipType,
		LastUpdatedBy: author,
		KommuneNr:     r.FormValue("knr"),
		GardsNr:       parseInt(r.FormValue("gnr")),
		BruksNr:       parseInt(r.FormValue("bnr")),
		FesteNr:       parseInt(r.FormValue("fnr")),
		SeksjonsNr:    parseInt(r.FormValue("snr")),
	}

	if id == 0 {
		models.AddHouse(house, userID, anonHash)
	} else {
		models.UpdateHouse(house, userID, anonHash)
	}

	http.Redirect(w, r, "/kart", http.StatusSeeOther)
}

// RemoveHandler handles the POST request to mark a house as deleted.
func RemoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "House ID required", http.StatusBadRequest)
		return
	}
	userID := getLoggedInUserID(r)
	anonHash := getFingerprint(r)
	err := models.RemoveHouse(id, userID, anonHash)
	if err != nil {
		slog.Error("Failed to remove house", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/kart", http.StatusSeeOther)
}

// RestoreHandler handles the POST request to restore a deleted house.
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "House ID required", http.StatusBadRequest)
		return
	}
	userID := getLoggedInUserID(r)
	anonHash := getFingerprint(r)
	err := models.RestoreHouse(id, userID, anonHash)
	if err != nil {
		slog.Error("Failed to restore house", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/kart/historikk", http.StatusSeeOther)
}

// SignupHandler handles user registration.
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}
	err := models.CreateUser(username, password)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		http.Error(w, "Username already taken or other error", http.StatusConflict)
		return
	}
	// Log in automatically after signup
	user, _ := models.GetUserByUsername(username)
	http.SetCookie(w, &http.Cookie{
		Name:  "user_id",
		Value: strconv.Itoa(user.ID),
		Path:  "/",
	})
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

// LoginHandler handles user login.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	user, err := models.AuthenticateUser(username, password)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "user_id",
		Value: strconv.Itoa(user.ID),
		Path:  "/",
	})
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

// LogoutHandler handles user logout.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

// APIHouseHistoryHandler returns the audit log for a specific house as JSON.
func APIHouseHistoryHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "House ID required", http.StatusBadRequest)
		return
	}
	logs := models.GetAuditLogsForHouse(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// HistoryHandler serves the audit log view.
func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	logs := models.GetAuditLogs()
	
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006 15:04")
		},
		"getAuthor": func(l models.AuditLog) string {
			if l.Username != "" {
				return l.Username
			}
			return fmt.Sprintf("Anonym (%s)", l.AnonHash)
		},
	}

	tmpl, err := template.New("history.html").Funcs(funcMap).ParseFiles("views/history.html")
	if err != nil {
		slog.Error("Failed to parse history template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, logs)
}

// EditHandler serves the edit form for a house.
func EditHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
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
	tmpl.Execute(w, house)
}
