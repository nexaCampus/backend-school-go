package models

import (
	"time"
)

// Notice represents an official school circular or announcement.
type Notice struct {
	ID             string    `json:"id"`
	Category       string    `json:"category"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	Date           string    `json:"date"`
	Author         string    `json:"author"`
	IsUrgent       bool      `json:"is_urgent"`
	AttachmentName *string   `json:"attachment_name,omitempty"`
	AttachmentURL  *string   `json:"attachment_url,omitempty"`
	PublishedAt    time.Time `json:"published_at"`
}
