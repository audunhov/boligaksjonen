package models

// House represents a residential property that might be empty.
type House struct {
	ID            int     `json:"id"`
	Address       string  `json:"address"`
	Latitude      float64 `json:"lat"`
	Longitude     float64 `json:"lng"`
	Description   string  `json:"description"`
	OwnershipType string  `json:"ownership_type"` // Options: "kommune", "fylke", "stat", "privatperson", "selskap", or freetext
}

// GetAllHouses returns a mock list of empty houses.
func GetAllHouses() []House {
	return []House{
		{ID: 1, Address: "Oslogate 1", Latitude: 59.9079, Longitude: 10.7686, Description: "Tom bolig siden 2023", OwnershipType: "kommune"},
		{ID: 2, Address: "Trondheimsveien 5", Latitude: 59.9194, Longitude: 10.7645, Description: "Tomt lokale i 1. etasje", OwnershipType: "selskap"},
		{ID: 3, Address: "Thorvald Meyers gate 10", Latitude: 59.9234, Longitude: 10.7588, Description: "Oppusningsobjekt, ubebodd", OwnershipType: "privatperson"},
	}
}
