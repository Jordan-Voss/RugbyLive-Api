package rapidapi

import (
	"rugby-live-api/models"
)

type APIParams struct {
	CompetitionID string
	Season        string
	Date          string
}

type CompetitionResponse struct {
	Results []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		// Add other fields as needed
	} `json:"results"`
}

type Season struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CompetitionGroup struct {
	Name       string          `json:"name"`
	RapidAPIID int             `json:"rapid_api_id"`
	Seasons    []models.Season `json:"seasons"`
}

type CompetitionMapping struct {
	CompetitionGroup CompetitionGroup     `json:"competition_group"`
	League           *models.League       `json:"league,omitempty"`
	Matched          bool                 `json:"matched"`
	Reason           string               `json:"reason"`
	APIMappings      []*models.APIMapping `json:"api_mappings,omitempty"`
}
