package models

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

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
}

var db *sql.DB

// InitDB initializes the SQLite database connection and creates the necessary tables.
func InitDB(filepath string) error {
	var err error
	// Use modernc.org/sqlite driver
	db, err = sql.Open("sqlite", filepath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS houses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		address TEXT NOT NULL,
		latitude REAL NOT NULL,
		longitude REAL NOT NULL,
		description TEXT,
		ownership_type TEXT,
		last_updated_by TEXT,
		updated_at DATETIME
	);
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create houses table: %w", err)
	}

	// Seed data if table is completely empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM houses").Scan(&count)
	if err == nil && count == 0 {
		slog.Info("Database is empty, seeding initial mock data")
		seedData := []House{
			{Address: "Oslogate 1", Latitude: 59.9079, Longitude: 10.7686, Description: "Tom bolig siden 2023", OwnershipType: "kommune", LastUpdatedBy: "System", UpdatedAt: time.Now()},
			{Address: "Trondheimsveien 5", Latitude: 59.9194, Longitude: 10.7645, Description: "Tomt lokale i 1. etasje", OwnershipType: "selskap", LastUpdatedBy: "System", UpdatedAt: time.Now()},
			{Address: "Thorvald Meyers gate 10", Latitude: 59.9234, Longitude: 10.7588, Description: "Oppusningsobjekt, ubebodd", OwnershipType: "privatperson", LastUpdatedBy: "System", UpdatedAt: time.Now()},
		}
		for _, h := range seedData {
			AddHouse(h)
		}
	}

	return nil
}

// GetAllHouses returns all houses from the database.
func GetAllHouses() []House {
	houses := []House{}
	
	if db == nil {
		slog.Error("Database not initialized")
		return houses
	}

	rows, err := db.Query(`
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at 
		FROM houses
	`)
	if err != nil {
		slog.Error("Failed to query houses", "error", err)
		return houses
	}
	defer rows.Close()

	for rows.Next() {
		var h House
		if err := rows.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt); err != nil {
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
		SELECT id, address, latitude, longitude, description, ownership_type, last_updated_by, updated_at 
		FROM houses WHERE id = ?
	`
	row := db.QueryRow(query, id)
	err := row.Scan(&h.ID, &h.Address, &h.Latitude, &h.Longitude, &h.Description, &h.OwnershipType, &h.LastUpdatedBy, &h.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return h, errors.New("house not found")
		}
		return h, fmt.Errorf("failed to get house: %w", err)
	}

	return h, nil
}

// UpdateHouse updates an existing house record in the database.
func UpdateHouse(updated House) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	query := `
		UPDATE houses 
		SET address = ?, latitude = ?, longitude = ?, description = ?, ownership_type = ?, last_updated_by = ?, updated_at = ?
		WHERE id = ?
	`
	
	if updated.UpdatedAt.IsZero() {
		updated.UpdatedAt = time.Now()
	}

	result, err := db.Exec(query, updated.Address, updated.Latitude, updated.Longitude, updated.Description, updated.OwnershipType, updated.LastUpdatedBy, updated.UpdatedAt, updated.ID)
	if err != nil {
		return fmt.Errorf("failed to update house: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("house not found")
	}

	slog.Info("Successfully updated house in database", "id", updated.ID)
	return nil
}

// AddHouse adds a new house to the database and returns its new ID.
func AddHouse(newHouse House) int {
	if db == nil {
		slog.Error("Database not initialized")
		return 0
	}

	if newHouse.UpdatedAt.IsZero() {
		newHouse.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO houses (address, latitude, longitude, description, ownership_type, last_updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(query, newHouse.Address, newHouse.Latitude, newHouse.Longitude, newHouse.Description, newHouse.OwnershipType, newHouse.LastUpdatedBy, newHouse.UpdatedAt)
	if err != nil {
		slog.Error("Failed to insert house", "error", err)
		return 0
	}

	id, err := result.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert ID", "error", err)
		return 0
	}

	slog.Info("Successfully inserted new house", "id", id)
	return int(id)
}
