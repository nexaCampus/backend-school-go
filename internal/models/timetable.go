package models

// TimetableSlot represents a scheduled classroom period in the bell schedule.
type TimetableSlot struct {
	ID          string `json:"id"`
	Grade       string `json:"grade,omitempty"`
	Section     string `json:"section,omitempty"`
	DayOfWeek   int    `json:"day_of_week"`
	PeriodLabel string `json:"label"`
	Time        string `json:"time"`
	Subject     string `json:"subject"`
	Detail      string `json:"detail"`
	RoomNumber  string `json:"room_number,omitempty"`
	Teacher     string `json:"teacher,omitempty"`
	IsLive      bool   `json:"is_live"`
}
