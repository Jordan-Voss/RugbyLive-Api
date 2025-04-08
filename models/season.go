package models

import "time"

type Season struct {
	ID           string    `json:"id" db:"id"`
	LeagueID     string    `json:"league_id" db:"league_id"`
	Year         int       `json:"year" db:"year"`
	RapidAPIYear int       `json:"rapid_api_year" db:"rapid_api_year"`
	Current      bool      `json:"current" db:"current"`
	StartDate    time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate      time.Time `json:"end_date,omitempty" db:"end_date"`
	YearRange    string    `json:"year_range,omitempty" db:"year_range"`
	CreatedAt    time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
