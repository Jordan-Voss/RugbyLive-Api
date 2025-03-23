package services

import (
	"fmt"
	"rugby-live-api/db"
	"time"
)

func (a *APIClient) GetLeagues(store *db.Store) error {
	// Get current year leagues from RugbyDB
	currentYear := time.Now().Year()
	rugbyDBLeagues, err := a.GetLeaguesByYear(store, fmt.Sprintf("%d", currentYear), false)
	if err != nil {
		return fmt.Errorf("failed to get RugbyDB leagues: %v", err)
	}

	// Store the leagues
	for _, league := range rugbyDBLeagues {
		if err := store.UpsertLeague(&league); err != nil {
			return fmt.Errorf("failed to store league %s: %v", league.Name, err)
		}
	}

	// Get leagues from APISports
	// apiSportsLeagues, err := a.GetAPISportsLeagues()
	// if err != nil {
	// 	return fmt.Errorf("failed to get APISports leagues: %v", err)
	// }

	// Process and merge leagues...
	return nil
}
