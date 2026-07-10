package dto

// --- SOS DTOs ---

// SOSTriggerRequest SOS触发
type SOSTriggerRequest struct {
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	LocationSource string  `json:"location_source"`
	LastKnownLat   float64 `json:"last_known_lat,omitempty"`
	LastKnownLng   float64 `json:"last_known_lng,omitempty"`
	Address        string  `json:"address,omitempty"`
	GuestPhone     string  `json:"guest_phone,omitempty"`
}

// SOSConfirmRequest SOS确认
type SOSConfirmRequest struct {
	Status string `json:"status" binding:"required,oneof=confirmed cancelled false_alarm"`
}

// SOSResponse SOS响应
type SOSResponse struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	RequestID      string `json:"request_id,omitempty"`
	LocationSource string `json:"location_source"`
}
