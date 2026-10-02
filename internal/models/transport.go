package models

// TransportStop represents a scheduled bus stop along a school bus route.
type TransportStop struct {
	Name string  `json:"name"`
	Time string  `json:"time"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// TransportRoute represents a school bus route with driver and GPS tracking info.
type TransportRoute struct {
	Route       string          `json:"route"`
	BusNumber   string          `json:"bus_number"`
	DriverName  string          `json:"driver_name"`
	DriverPhone string          `json:"driver_phone"`
	CurrentLat  float64         `json:"current_lat"`
	CurrentLng  float64         `json:"current_lng"`
	Status      string          `json:"status"`
	Stops       []TransportStop `json:"stops"`
}

// UpdateLocationRequest is the payload for driver GPS updates via POST /v1/transport/routes/{route}/location
type UpdateLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
