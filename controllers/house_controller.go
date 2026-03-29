package controllers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strconv"

	"github.com/audunhov/tombolig/components"
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
	
	// Verify user exists in DB
	_, err = models.GetUserByID(id)
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

// Helper to calculate password entropy in Go
func calculateEntropy(password string) float64 {
	if password == "" {
		return 0
	}
	var pool float64
	if regexp.MustCompile(`[a-z]`).MatchString(password) {
		pool += 26
	}
	if regexp.MustCompile(`[A-Z]`).MatchString(password) {
		pool += 26
	}
	if regexp.MustCompile(`[0-9]`).MatchString(password) {
		pool += 10
	}
	if regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		pool += 33
	}
	if pool == 0 {
		return 0
	}
	return float64(len(password)) * math.Log2(pool)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
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

	stats := models.GetGlobalStats()

	err := components.Home(username, stats).Render(r.Context(), w)
	if err != nil {
		slog.Error("Failed to render home component", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func MapHandler(w http.ResponseWriter, r *http.Request) {
	var username string
	userID := getLoggedInUserID(r)
	if userID != nil {
		user, err := models.GetUserByID(*userID)
		if err == nil {
			username = user.Username
		}
	}

	err := components.Map(username).Render(r.Context(), w)
	if err != nil {
		slog.Error("Failed to render map component", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func APIHousesHandler(w http.ResponseWriter, r *http.Request) {
	houses := models.GetAllHousesFull()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	json.NewEncoder(w).Encode(houses)
}

func CheckUsernameHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	exists := models.UserExists(username)
	
	err := components.UsernameNotice(exists).Render(r.Context(), w)
	if err != nil {
		slog.Error("Failed to render username notice component", "error", err)
	}
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
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
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
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
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
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	isHTMX := r.Header.Get("HX-Request") == "true"

	if username == "" || password == "" {
		if isHTMX {
			components.AuthError("Brukernavn og passord er påkrevd").Render(r.Context(), w)
			return
		}
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	if len(username) < 4 {
		if isHTMX {
			components.AuthError("Brukernavn må være minst 4 tegn").Render(r.Context(), w)
			return
		}
		http.Error(w, "Username must be at least 4 characters", http.StatusBadRequest)
		return
	}

	if password != confirm {
		if isHTMX {
			components.AuthError("Passordene er ikke like").Render(r.Context(), w)
			return
		}
		http.Error(w, "Passwords do not match", http.StatusBadRequest)
		return
	}

	if calculateEntropy(password) < 40 {
		if isHTMX {
			components.AuthError("Passordet er for svakt").Render(r.Context(), w)
			return
		}
		http.Error(w, "Password too weak", http.StatusBadRequest)
		return
	}

	err := models.CreateUser(username, password)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		if isHTMX {
			components.AuthError("Brukernavnet er tatt").Render(r.Context(), w)
			return
		}
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

	if isHTMX {
		w.Header().Set("HX-Redirect", r.Referer())
		return
	}
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

// LoginHandler handles user login.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	
	isHTMX := r.Header.Get("HX-Request") == "true"

	user, err := models.AuthenticateUser(username, password)
	if err != nil {
		if isHTMX {
			components.AuthError("Feil brukernavn eller passord").Render(r.Context(), w)
			return
		}
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "user_id",
		Value: strconv.Itoa(user.ID),
		Path:  "/",
	})

	if isHTMX {
		w.Header().Set("HX-Redirect", r.Referer())
		return
	}
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

// AddCommentHandler handles POST requests to add a comment to a house.
func AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	houseID, _ := strconv.Atoi(r.FormValue("house_id"))
	content := r.FormValue("comment")
	if houseID == 0 || content == "" {
		http.Error(w, "House ID and comment required", http.StatusBadRequest)
		return
	}

	userID := getLoggedInUserID(r)
	anonHash := getFingerprint(r)

	err := models.AddComment(houseID, content, userID, anonHash)
	if err != nil {
		slog.Error("Failed to add comment", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Fetch updated logs
	logs := models.GetAuditLogsForHouse(houseID)
	err = components.AuditLogList(logs).Render(r.Context(), w)
	if err != nil {
		slog.Error("Failed to render audit log list component", "error", err)
	}
}

// APIHouseHistoryHandler returns the audit log for a specific house as HTML for htmx.
func APIHouseHistoryHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "House ID required", http.StatusBadRequest)
		return
	}
	logs := models.GetAuditLogsForHouse(id)
	
	err := components.AuditLogList(logs).Render(r.Context(), w)
	if err != nil {
		slog.Error("Failed to render audit log list", "error", err)
	}
}

// HistoryHandler serves the audit log view.
func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	logs := models.GetAuditLogs()
	
	var username string
	userID := getLoggedInUserID(r)
	if userID != nil {
		user, err := models.GetUserByID(*userID)
		if err == nil {
			username = user.Username
		}
	}

	err := components.History(username, logs).Render(r.Context(), w)
	if err != nil {
		slog.Error("Failed to render history component", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

