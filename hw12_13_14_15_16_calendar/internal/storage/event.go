package storage

import "time"

type Event struct {
	ID          string     `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"` // опционально → *string → NULL
	OwnerID     int64      `json:"owner_id" db:"owner_id"`
	DateStart   time.Time  `json:"date_start" db:"date_start"`
	DateEnd     time.Time  `json:"date_end" db:"date_end"`
	NotifyAt    *time.Time `json:"notify_at" db:"notify_at"` // опционально → *time.Time → NULL
	CreatedAt   time.Time  `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty" db:"updated_at"`
}
