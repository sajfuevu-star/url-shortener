package link

import (
	"time"
)

type Link struct {
	Code        string     `json:"code"`
	OriginalURL string     `json:"originalurl"`
	Clicks      int        `json:"clicks"`
	Alias       string     `json:"alias"`
	CreatedAt   time.Time  `json:"createdat"`
	ExpiresAt   *time.Time `json:"expiresat"`
}
