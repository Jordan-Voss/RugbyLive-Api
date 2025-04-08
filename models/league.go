package models

import "time"

type League struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Country       Country   `json:"country" db:"country"`
	Region        string    `json:"region" db:"region"`
	TeamCountries []Country `json:"team_countries" db:"-"`
	Tier          int       `json:"tier" db:"tier"`
	Format        string    `json:"format" db:"format"`
	Phases        []string  `json:"phases,omitempty" db:"phases"`
	AltNames      []string  `json:"alt_names,omitempty" db:"alt_names"`
	LogoURL       string    `json:"logo_url,omitempty" db:"logo_url"`
	LogoSource    string    `json:"logo_source" db:"logo_source"`
	International bool      `json:"international" db:"international"`
	Gender        string    `json:"gender" db:"gender"`
	ParentID      *string   `json:"parent_id" db:"parent_league_id"`
	Seasons       []Season  `json:"seasons,omitempty"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	RugbyDBID     string    `json:"rugby_db_id" db:"rugby_db_id"`
	SuccessorID   *string   `json:"successor_id,omitempty" db:"successor_league_id"`
	AllTime       bool      `json:"all_time" db:"all_time"`
	AllTimeID     string    `json:"all_time_id,omitempty" db:"all_time_league_id"`
}

type LeagueTeam struct {
	ID       string `json:"id" db:"id"`
	SeasonID string `json:"season_id" db:"season_id"`
	TeamID   string `json:"team_id" db:"team_id"`
}
