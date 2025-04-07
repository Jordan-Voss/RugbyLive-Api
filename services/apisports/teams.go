package apisports

import (
	"encoding/json"
	"fmt"
	"log"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"rugby-live-api/services/rugbydb"
	"strconv"
	"strings"
)

func (c *Client) FetchAndStoreTeams(store *db.Store, updateImages bool, params TeamSearchParams) ([]teamChange, []failedTeam, error) {
	url := "https://v1.rugby.api-sports.io/teams"
	if params.CountryID != "" {
		url += fmt.Sprintf("?country=%s", params.CountryID)
	}
	if params.LeagueID != "" {
		url += fmt.Sprintf("?league=%s&season=%d", params.LeagueID, params.Season)
	}

	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Response []struct {
			Team struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Logo string `json:"logo"`
			} `json:"team"`
			Country struct {
				Name string `json:"name"`
				Code string `json:"code"`
			} `json:"country"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, nil, err
	}

	var changes []teamChange
	var failedTeams []failedTeam

	for _, team := range apiResp.Response {
		countryCode := strings.ToUpper(team.Country.Code)
		teamID := fmt.Sprintf("%s-%s", countryCode,
			strings.ToUpper(strings.ReplaceAll(team.Team.Name, " ", "")))

		dbTeam := &models.Team{
			ID:      teamID,
			Name:    team.Team.Name,
			LogoURL: team.Team.Logo,
			Country: models.Country{
				Code: countryCode,
				Name: team.Country.Name,
			},
		}

		change := teamChange{
			IsNew:    true,
			Changes:  make(map[string]map[string]string),
			TeamName: team.Team.Name,
		}

		existing, err := store.GetTeamByID(teamID)
		if err == nil && existing != nil {
			change.IsNew = false
			if existing.Name != dbTeam.Name {
				change.Changes["name"] = map[string]string{"old": existing.Name, "new": dbTeam.Name}
			}
			if updateImages && existing.LogoURL != dbTeam.LogoURL {
				change.Changes["logo"] = map[string]string{"old": existing.LogoURL, "new": dbTeam.LogoURL}
			}
		}

		if err := store.UpsertTeam(dbTeam); err != nil {
			failedTeams = append(failedTeams, failedTeam{
				Name:        team.Team.Name,
				CountryID:   countryCode,
				CountryName: team.Country.Name,
				Reason:      fmt.Sprintf("Failed to upsert: %v", err),
				TeamData:    team,
			})
			continue
		}

		mapping := &models.APIMapping{
			EntityID:   teamID,
			APIName:    "api_sports",
			APIID:      strconv.Itoa(team.Team.ID),
			EntityType: "team",
		}

		if err := store.UpsertAPIMapping(mapping); err != nil {
			log.Printf("Error creating API mapping for team %s: %v", team.Team.Name, err)
		}

		if change.IsNew {
			log.Printf("Added new team: %s", team.Team.Name)
		} else if len(change.Changes) > 0 {
			log.Printf("Updated team: %s with changes: %v", team.Team.Name, change.Changes)
		}

		changes = append(changes, change)
	}

	return changes, failedTeams, nil
}

func (c *Client) UpdateTeamImages(store *db.Store) error {
	teams, err := store.GetTeams()
	if err != nil {
		return fmt.Errorf("failed to get teams: %v", err)
	}

	for _, team := range teams {
		if team.LogoURL == "" {
			mapping, err := store.GetAPIMappingByEntityID("api_sports", team.ID, "team")
			if err != nil || mapping == nil {
				continue
			}

			url := fmt.Sprintf("https://v1.rugby.api-sports.io/teams?id=%s", mapping.APIID)
			req, err := c.createRequest("GET", url)
			if err != nil {
				continue
			}

			resp, err := c.client.Do(req)
			if err != nil {
				continue
			}

			var apiResp struct {
				Response []struct {
					Team struct {
						Logo string `json:"logo"`
					} `json:"team"`
				} `json:"response"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			if len(apiResp.Response) > 0 {
				team.LogoURL = apiResp.Response[0].Team.Logo
				if err := store.UpsertTeam(team); err != nil {
					log.Printf("Error updating team logo for %s: %v", team.Name, err)
				}
			}
		}
	}

	return nil
}

func (c *Client) GetWikidataTeams() ([]models.Team, error) {
	// This functionality has been moved to a separate wikidata package
	return nil, fmt.Errorf("wikidata functionality moved to wikidata package")
}

func (c *Client) SearchWikidataTeam(name string) (*models.Team, error) {
	// This functionality has been moved to a separate wikidata package
	return nil, fmt.Errorf("wikidata functionality moved to wikidata package")
}

func (c *Client) GetRugbyDBTeams(store *db.Store, priorityTeams []string, countryFilter string) ([]models.Team, error) {
	var teams []models.Team
	for name, countryCode := range rugbydb.TeamCountryMap {
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
