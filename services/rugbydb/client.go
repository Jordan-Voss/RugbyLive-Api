package rugbydb

import (
	"fmt"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"strings"
)

type Client struct {
	// No configuration needed for now
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) GetTeams(store *db.Store, priorityTeams []string, countryFilter string) ([]models.Team, error) {
	var teams []models.Team
	for name, countryCode := range TeamCountryMap {
		if countryFilter != "" && countryFilter != countryCode {
			continue
		}

		if len(priorityTeams) > 0 {
			found := false
			for _, pName := range priorityTeams {
				if strings.EqualFold(name, pName) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		teamID := fmt.Sprintf("%s-%s", countryCode,
			strings.ToUpper(strings.ReplaceAll(name, " ", "")))

		team := &models.Team{
			ID:   teamID,
			Name: name,
			Country: models.Country{
				Code: countryCode,
			},
		}

		teams = append(teams, *team)
	}

	return teams, nil
}

// Move LeagueCountryMap, TeamCountryMap, and other rugbydb-specific data here
