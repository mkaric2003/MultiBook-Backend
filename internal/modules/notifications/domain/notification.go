// Package domain contains notification models free from HTTP, Firebase, and PostgreSQL dependencies.
package domain

import "time"

type Notification struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	CreatedAt time.Time         `json:"createdAt"`
	ReadAt    *time.Time        `json:"readAt"`
}

type Page struct {
	Items      []Notification
	NextCursor *string
}

type Event struct {
	ID          string
	RecipientID string
	Kind        string
	Title       string
	Body        string
	Data        map[string]string
}
