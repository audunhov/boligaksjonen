package models

// House represents a residential property that might be empty.
type House struct {
	ID          int     `json:"id"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lng"`
	Description string  `json:"description"`
}

// GetAllHouses returns a mock list of empty houses.
func GetAllHouses() []House {
	return []House{
		{ID: 1, Address: "Oslogate 1", Latitude: 59.9079, Longitude: 10.7686, Description: "Tom bolig siden 2023"},
		{ID: 2, Address: "Trondheimsveien 5", Latitude: 59.9194, Longitude: 10.7645, Description: "Tomt lokale i 1. etasje"},
		{ID: 3, Address: "Thorvald Meyers gate 10", Latitude: 59.9234, Longitude: 10.7588, Description: "Oppusningsobjekt, ubebodd"},
	}
}
