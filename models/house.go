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
	Action    string    `json:"action"` // "add", "update", "remove", "restore"
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
	IsDeleted     bool      `json:"is_deleted"`
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
			is_deleted INTEGER DEFAULT 0,
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

	return nil
}

// GetAllHousesFull returns all houses (including deleted) from the database.
func GetAllHousesFull() []House {
	houses := []House{}
	if db == nil {
		return houses
	}

	rows, err := db.Query(`
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, IFNULL(knr, ''), IFNULL(gnr, 0), IFNULL(bnr, 0), IFNULL(fnr, 0), IFNULL(snr, 0), is_deleted
		FROM houses
	`)
	if err != nil {
		slog.Error("Failed to query all houses", "error", err)
		return houses
	}
	defer rows.Close()

	for rows.Next() {
		var h House
		if err := rows.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt, &h.UserID, &h.AnonHash, &h.KommuneNr, &h.GardsNr, &h.BruksNr, &h.FesteNr, &h.SeksjonsNr, &h.IsDeleted); err != nil {
			slog.Error("Failed to scan house row", "error", err)
			continue
		}
		houses = append(houses, h)
	}
	return houses
}

// GetAllHouses returns all active houses from the database.
func GetAllHouses() []House {
	houses := []House{}
	if db == nil {
		return houses
	}

	rows, err := db.Query(`
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, IFNULL(knr, ''), IFNULL(gnr, 0), IFNULL(bnr, 0), IFNULL(fnr, 0), IFNULL(snr, 0), is_deleted
		FROM houses
		WHERE is_deleted = 0
	`)
	if err != nil {
		slog.Error("Failed to query houses", "error", err)
		return houses
	}
	defer rows.Close()

	for rows.Next() {
		var h House
		if err := rows.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt, &h.UserID, &h.AnonHash, &h.KommuneNr, &h.GardsNr, &h.BruksNr, &h.FesteNr, &h.SeksjonsNr, &h.IsDeleted); err != nil {
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
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, IFNULL(knr, ''), IFNULL(gnr, 0), IFNULL(bnr, 0), IFNULL(fnr, 0), IFNULL(snr, 0), is_deleted
		FROM houses WHERE id = ?
	`
	row := db.QueryRow(query, id)
	err := row.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt, &h.UserID, &h.AnonHash, &h.KommuneNr, &h.GardsNr, &h.BruksNr, &h.FesteNr, &h.SeksjonsNr, &h.IsDeleted)
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
		SET address = ?, latitude = ?, longitude = ?, description = ?, ownership_type = ?, last_updated_by = ?, updated_at = ?, user_id = ?, anon_hash = ?, knr = ?, gnr = ?, bnr = ?, fnr = ?, snr = ?, is_deleted = ?
		WHERE id = ?
	`
	_, err = db.Exec(query, updated.Address, updated.Latitude, updated.Longitude, updated.Description, updated.OwnershipType, updated.LastUpdatedBy, updated.UpdatedAt, userID, anonHash, updated.KommuneNr, updated.GardsNr, updated.BruksNr, updated.FesteNr, updated.SeksjonsNr, updated.IsDeleted, updated.ID)
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

// AddHouse adds a new house to the database or restores a deleted one at the same address.
func AddHouse(newHouse House, userID *int, anonHash string) int {
	if db == nil {
		return 0
	}

	// Check if a house already exists at this address (even if deleted)
	var existingID int
	var isDeleted int
	err := db.QueryRow("SELECT id, is_deleted FROM houses WHERE address = ?", newHouse.Address).Scan(&existingID, &isDeleted)
	
	if err == nil {
		// House exists. If it's deleted, we restore and update it.
		// If it's NOT deleted, the caller should have handled it, but we'll update it anyway to be safe.
		newHouse.ID = existingID
		newHouse.IsDeleted = false
		UpdateHouse(newHouse, userID, anonHash)
		
		// If it was previously deleted, add a specific log for restoration
		if isDeleted == 1 {
			logQuery := `INSERT INTO audit_log (house_id, action, new_data, user_id, anon_hash, timestamp) VALUES (?, ?, ?, ?, ?, ?)`
			db.Exec(logQuery, existingID, "restore", "Gjenrapportert via nytt skjema", userID, anonHash, time.Now())
		}
		return existingID
	}

	// House doesn't exist, proceed with normal insertion
	if newHouse.UpdatedAt.IsZero() {
		newHouse.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO houses (address, latitude, longitude, description, ownership_type, last_updated_by, updated_at, user_id, anon_hash, knr, gnr, bnr, fnr, snr, is_deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(query, newHouse.Address, newHouse.Latitude, newHouse.Longitude, newHouse.Description, newHouse.OwnershipType, newHouse.LastUpdatedBy, newHouse.UpdatedAt, userID, anonHash, newHouse.KommuneNr, newHouse.GardsNr, newHouse.BruksNr, newHouse.FesteNr, newHouse.SeksjonsNr, newHouse.IsDeleted)
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

// RemoveHouse marks a house as deleted.
func RemoveHouse(id int, userID *int, anonHash string) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	house, err := GetHouseByID(id)
	if err != nil {
		return err
	}
	oldJSON, _ := json.Marshal(house)

	_, err = db.Exec("UPDATE houses SET is_deleted = 1 WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Log the change
	logQuery := `
		INSERT INTO audit_log (house_id, action, old_data, new_data, user_id, anon_hash, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(logQuery, id, "remove", string(oldJSON), nil, userID, anonHash, time.Now())
	return err
}

// RestoreHouse restores a deleted house.
func RestoreHouse(id int, userID *int, anonHash string) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	_, err := db.Exec("UPDATE houses SET is_deleted = 0 WHERE id = ?", id)
	if err != nil {
		return err
	}

	house, _ := GetHouseByID(id)
	newJSON, _ := json.Marshal(house)

	// Log the change
	logQuery := `
		INSERT INTO audit_log (house_id, action, old_data, new_data, user_id, anon_hash, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(logQuery, id, "restore", nil, string(newJSON), userID, anonHash, time.Now())
	return err
}

// AddComment adds a comment to a house's history.
func AddComment(houseID int, content string, userID *int, anonHash string) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	logQuery := `
		INSERT INTO audit_log (house_id, action, new_data, user_id, anon_hash, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(logQuery, houseID, "comment", content, userID, anonHash, time.Now())
	return err
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

// Stats represents global application statistics.
type Stats struct {
	TotalHouses      int
	TotalContributors int
}

// GetGlobalStats retrieves totals for the landing page.
func GetGlobalStats() Stats {
	var s Stats
	if db == nil {
		return s
	}

	// Total houses (excluding deleted)
	db.QueryRow("SELECT COUNT(*) FROM houses WHERE is_deleted = 0").Scan(&s.TotalHouses)

	// Total unique contributors (users + unique anon hashes from audit_log)
	db.QueryRow(`
		SELECT COUNT(DISTINCT combined_id) FROM (
			SELECT CAST(user_id AS TEXT) as combined_id FROM audit_log WHERE user_id IS NOT NULL
			UNION
			SELECT anon_hash as combined_id FROM audit_log WHERE user_id IS NULL
		)
	`).Scan(&s.TotalContributors)

	return s
}

// PermanentlyDeleteOldHouses removes houses that have been marked as deleted for more than 2 months.
func PermanentlyDeleteOldHouses() (int64, error) {
	if db == nil {
		return 0, errors.New("database not initialized")
	}

	twoMonthsAgo := time.Now().AddDate(0, -2, 0)
	
	// Find IDs to delete for logging purposes
	rows, err := db.Query("SELECT id FROM houses WHERE is_deleted = 1 AND updated_at < ?", twoMonthsAgo)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return 0, nil
	}

	// Delete from audit_log first (optional, or keep history)
	// For this requirement, we'll keep audit log but remove the houses
	result, err := db.Exec("DELETE FROM houses WHERE is_deleted = 1 AND updated_at < ?", twoMonthsAgo)
	if err != nil {
		return 0, err
	}

	count, _ := result.RowsAffected()
	slog.Info("Permanently deleted old houses", "count", count, "ids", ids)
	return count, nil
}

// UserExists checks if a username already exists.
func UserExists(username string) bool {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = ?`
	err := db.QueryRow(query, username).Scan(&count)
	return err == nil && count > 0
}

// CreateUser registers a new user with a hashed password.
func CreateUser(username, password string) error {
	if UserExists(username) {
		return errors.New("username already exists")
	}
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
