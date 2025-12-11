package domain

import "time"

type Project struct {
	ID        string
	Key       string
	Name      string
	CreatedAt time.Time
}
