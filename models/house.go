package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

// User represents a registered user.
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Don't expose password hash in JSON
	CreatedAt    time.Time `json:"created_at"`
}

// AuditLog represents a single change to a house entry.
type AuditLog struct {
	ID        int       `json:"id"`
	HouseID   int       `json:"house_id"`
	Action    string    `json:"action"` // "add", "update"
	OldData   string    `json:"old_data"`
	NewData   string    `json:"new_data"`
	UserID    *int      `json:"user_id,omitempty"`
	AnonHash  string    `json:"anon_hash,omitempty"`
	Username  string    `json:"username,omitempty"` // For display in history
	Timestamp time.Time `json:"timestamp"`
}

// House represents a residential property that might be empty.
type House struct {
	ID            int       `json:"id"`
	Address       string    `json:"address"`
	Latitude      float64   `json:"lat"`
	Longitude     float64   `json:"lng"`
	Description   string    `json:"description"`
	OwnershipType string    `json:"ownership_type"`
	LastUpdatedBy string    `json:"last_updated_by"`
	UpdatedAt     time.Time `json:"updated_at"`
	UserID        *int      `json:"user_id,omitempty"`
	AnonHash      string    `json:"anon_hash,omitempty"`
	// Matrikkel fields for direct links
	KommuneNr  string `json:"knr"`
	GardsNr    int    `json:"gnr"`
	BruksNr    int    `json:"bnr"`
	FesteNr    int    `json:"fnr"`
	SeksjonsNr int    `json:"snr"`
}

var db *sql.DB

// InitDB initializes the SQLite database connection and creates the necessary tables.
func InitDB(filepath string) error {
	var err error
	db, err = sql.Open("sqlite", filepath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables if they don't exist
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS houses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			address TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			description TEXT,
			ownership_type TEXT,
			last_updated_by TEXT,
			updated_at DATETIME,
			user_id INTEGER,
			anon_hash TEXT,
			knr TEXT,
			gnr INTEGER,
			bnr INTEGER,
			fnr INTEGER,
			snr INTEGER,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			house_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			old_data TEXT,
			new_data TEXT,
			user_id INTEGER,
			anon_hash TEXT,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (house_id) REFERENCES houses(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
	}

	for _, q := range queries {
		_, err = db.Exec(q)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// Seed data if table is completely empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM houses").Scan(&count)
	if err == nil && count == 0 {
		slog.Info("Database is empty, seeding initial mock data")
		seedData := []House{
			{Address: "Oslogate 1", Latitude: 59.9079, Longitude: 10.7686, Description: "Tom bolig siden 2023", OwnershipType: "offentlig", LastUpdatedBy: "System", UpdatedAt: time.Now(), KommuneNr: "0301", GardsNr: 232, BruksNr: 1},
			{Address: "Trondheimsveien 5", Latitude: 59.9194, Longitude: 10.7645, Description: "Tomt lokale i 1. etasje", OwnershipType: "næring", LastUpdatedBy: "System", UpdatedAt: time.Now(), KommuneNr: "0301", GardsNr: 228, BruksNr: 1},
			{Address: "Thorvald Meyers gate 10", Latitude: 59.9234, Longitude: 10.7588, Description: "Oppusningsobjekt, ubebodd", OwnershipType: "privat-bolig", LastUpdatedBy: "System", UpdatedAt: time.Now(), KommuneNr: "0301", GardsNr: 226, BruksNr: 1},
		}
		for _, h := range seedData {
			AddHouse(h, nil, "SystemHash")
		}
	}

	return nil
}

// GetAllHouses returns all houses from the database.
func GetAllHouses() []House {
	houses := []House{}
	if db == nil {
		return houses
	}

	rows, err := db.Query(`
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, IFNULL(knr, ''), IFNULL(gnr, 0), IFNULL(bnr, 0), IFNULL(fnr, 0), IFNULL(snr, 0)
		FROM houses
	`)
	if err != nil {
		slog.Error("Failed to query houses", "error", err)
		return houses
	}
	defer rows.Close()

	for rows.Next() {
		var h House
		if err := rows.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt, &h.UserID, &h.AnonHash, &h.KommuneNr, &h.GardsNr, &h.BruksNr, &h.FesteNr, &h.SeksjonsNr); err != nil {
			slog.Error("Failed to scan house row", "error", err)
			continue
		}
		houses = append(houses, h)
	}
	return houses
}

// GetHouseByID finds a house by its ID in the database.
func GetHouseByID(id int) (House, error) {
	var h House
	if db == nil {
		return h, errors.New("database not initialized")
	}

	query := `
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, IFNULL(knr, ''), IFNULL(gnr, 0), IFNULL(bnr, 0), IFNULL(fnr, 0), IFNULL(snr, 0)
		FROM houses WHERE id = ?
	`
	row := db.QueryRow(query, id)
	err := row.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt, &h.UserID, &h.AnonHash, &h.KommuneNr, &h.GardsNr, &h.BruksNr, &h.FesteNr, &h.SeksjonsNr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return h, errors.New("house not found")
		}
		return h, fmt.Errorf("failed to get house: %w", err)
	}

	return h, nil
}

// UpdateHouse updates an existing house record in the database and logs the change.
func UpdateHouse(updated House, userID *int, anonHash string) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	// Fetch old data for audit log
	oldHouse, err := GetHouseByID(updated.ID)
	if err != nil {
		return err
	}
	oldJSON, _ := json.Marshal(oldHouse)

	if updated.UpdatedAt.IsZero() {
		updated.UpdatedAt = time.Now()
	}

	query := `
		UPDATE houses 
		SET address = ?, latitude = ?, longitude = ?, description = ?, ownership_type = ?, last_updated_by = ?, updated_at = ?, user_id = ?, anon_hash = ?, knr = ?, gnr = ?, bnr = ?, fnr = ?, snr = ?
		WHERE id = ?
	`
	_, err = db.Exec(query, updated.Address, updated.Latitude, updated.Longitude, updated.Description, updated.OwnershipType, updated.LastUpdatedBy, updated.UpdatedAt, userID, anonHash, updated.KommuneNr, updated.GardsNr, updated.BruksNr, updated.FesteNr, updated.SeksjonsNr, updated.ID)
	if err != nil {
		return fmt.Errorf("failed to update house: %w", err)
	}

	// Log the change
	newJSON, _ := json.Marshal(updated)
	logQuery := `
		INSERT INTO audit_log (house_id, action, old_data, new_data, user_id, anon_hash, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(logQuery, updated.ID, "update", string(oldJSON), string(newJSON), userID, anonHash, time.Now())
	if err != nil {
		slog.Error("Failed to log update in audit_log", "error", err)
	}

	return nil
}

// AddHouse adds a new house to the database and logs the action.
func AddHouse(newHouse House, userID *int, anonHash string) int {
	if db == nil {
		return 0
	}

	if newHouse.UpdatedAt.IsZero() {
		newHouse.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO houses (address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, knr, gnr, bnr, fnr, snr)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(query, newHouse.Address, newHouse.Latitude, newHouse.Longitude, newHouse.Description, newHouse.OwnershipType, newHouse.LastUpdatedBy, newHouse.UpdatedAt, userID, anonHash, newHouse.KommuneNr, newHouse.GardsNr, newHouse.BruksNr, newHouse.FesteNr, newHouse.SeksjonsNr)
	if err != nil {
		slog.Error("Failed to insert house", "error", err)
		return 0
	}

	id, _ := result.LastInsertId()
	newHouse.ID = int(id)

	// Log the change
	newJSON, _ := json.Marshal(newHouse)
	logQuery := `
		INSERT INTO audit_log (house_id, action, old_data, new_data, user_id, anon_hash, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(logQuery, int(id), "add", nil, string(newJSON), userID, anonHash, time.Now())
	if err != nil {
		slog.Error("Failed to log addition in audit_log", "error", err)
	}

	return int(id)
}

// GetAuditLogs returns all audit logs from the database.
func GetAuditLogs() []AuditLog {
	logs := []AuditLog{}
	if db == nil {
		return logs
	}

	query := `
		SELECT al.id, al.house_id, al.action, IFNULL(al.old_data, '') as old_data, IFNULL(al.new_data, '') as new_data, al.user_id, al.anon_hash, al.timestamp, IFNULL(u.username, '') as username
		FROM audit_log al
		LEFT JOIN users u ON al.user_id = u.id
		ORDER BY al.timestamp DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		slog.Error("Failed to query audit_log", "error", err)
		return logs
	}
	defer rows.Close()

	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.HouseID, &l.Action, &l.OldData, &l.NewData, &l.UserID, &l.AnonHash, &l.Timestamp, &l.Username); err != nil {
			slog.Error("Failed to scan audit_log row", "error", err)
			continue
		}
		logs = append(logs, l)
	}
	return logs
}

// GetAuditLogsForHouse returns all audit logs for a specific house.
func GetAuditLogsForHouse(houseID int) []AuditLog {
	logs := []AuditLog{}
	if db == nil {
		return logs
	}

	query := `
		SELECT al.id, al.house_id, al.action, IFNULL(al.old_data, '') as old_data, IFNULL(al.new_data, '') as new_data, al.user_id, al.anon_hash, al.timestamp, IFNULL(u.username, '') as username
		FROM audit_log al
		LEFT JOIN users u ON al.user_id = u.id
		WHERE al.house_id = ?
		ORDER BY al.timestamp DESC
	`
	rows, err := db.Query(query, houseID)
	if err != nil {
		slog.Error("Failed to query audit_log for house", "houseID", houseID, "error", err)
		return logs
	}
	defer rows.Close()

	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.HouseID, &l.Action, &l.OldData, &l.NewData, &l.UserID, &l.AnonHash, &l.Timestamp, &l.Username); err != nil {
			slog.Error("Failed to scan audit_log row", "error", err)
			continue
		}
		logs = append(logs, l)
	}
	return logs
}

// CreateUser registers a new user with a hashed password.
func CreateUser(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	query := `INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`
	_, err = db.Exec(query, username, string(hashedPassword), time.Now())
	return err
}

// GetUserByUsername finds a user by their username.
func GetUserByUsername(username string) (User, error) {
	var u User
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = ?`
	err := db.QueryRow(query, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

// GetUserByID finds a user by their ID.
func GetUserByID(id int) (User, error) {
	var u User
	query := `SELECT id, username, password_hash, created_at FROM users WHERE id = ?`
	err := db.QueryRow(query, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

// AuthenticateUser checks the password against the stored hash.
func AuthenticateUser(username, password string) (User, error) {
	user, err := GetUserByUsername(username)
	if err != nil {
		return User{}, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return User{}, errors.New("invalid password")
	}
	return user, nil
}
