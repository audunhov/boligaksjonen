package models

import (
	"errors"
	"sync"
	"time"
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

var (
	houses = []House{
		{ID: 1, Address: "Oslogate 1", Latitude: 59.9079, Longitude: 10.7686, Description: "Tom bolig siden 2023", OwnershipType: "kommune", LastUpdatedBy: "System", UpdatedAt: time.Now()},
		{ID: 2, Address: "Trondheimsveien 5", Latitude: 59.9194, Longitude: 10.7645, Description: "Tomt lokale i 1. etasje", OwnershipType: "selskap", LastUpdatedBy: "System", UpdatedAt: time.Now()},
		{ID: 3, Address: "Thorvald Meyers gate 10", Latitude: 59.9234, Longitude: 10.7588, Description: "Oppusningsobjekt, ubebodd", OwnershipType: "privatperson", LastUpdatedBy: "System", UpdatedAt: time.Now()},
	}
	mu sync.Mutex
)

// GetAllHouses returns the current list of houses.
func GetAllHouses() []House {
	mu.Lock()
	defer mu.Unlock()
	// Return a copy to avoid race conditions on the slice itself
	c := make([]House, len(houses))
	copy(c, houses)
	return c
}

// GetHouseByID finds a house by its ID.
func GetHouseByID(id int) (House, error) {
	mu.Lock()
	defer mu.Unlock()
	for _, h := range houses {
		if h.ID == id {
			return h, nil
		}
	}
	return House{}, errors.New("house not found")
}

// UpdateHouse updates an existing house record.
func UpdateHouse(updated House) error {
	mu.Lock()
	defer mu.Unlock()
	for i, h := range houses {
		if h.ID == updated.ID {
			houses[i] = updated
			return nil
		}
	}
	return errors.New("house not found")
}
