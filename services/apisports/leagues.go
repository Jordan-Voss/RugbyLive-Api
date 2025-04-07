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

func (c *Client) MapAPISportsLeagues(store *db.Store) ([]LeagueMappingResult, error) {
	url := "https://v1.rugby.api-sports.io/leagues"

	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Response []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
			Logo string `json:"logo"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	var results []LeagueMappingResult
	for _, league := range apiResp.Response {
		result := LeagueMappingResult{
			APILeague: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			}{
				ID:   strconv.Itoa(league.ID),
				Name: league.Name,
				Type: league.Type,
			},
			Matched:     false,
			MatchedName: "",
			Reason:      "no_match",
		}

		cleanName := rugbydb.CleanLeagueName(league.Name)
		matchFound := false

		if _, exists := rugbydb.LeagueCountryMap[cleanName]; exists {
			matchFound = true
			result.Matched = true
			result.MatchedName = cleanName
			result.Reason = "direct_match"
		}

		if !matchFound {
			for ourName, altNames := range rugbydb.LeagueAltNames {
				for _, altName := range altNames {
					if strings.EqualFold(cleanName, altName) {
						result.Matched = true
						result.MatchedName = ourName
						result.Reason = "alt_name_match"
						break
					}
				}
				if result.Matched {
					break
				}
			}
		}

		if result.Matched {
			if countryInfo, ok := rugbydb.LeagueCountryMap[result.MatchedName]; ok {
				result.InternalID = fmt.Sprintf("%s-%s", countryInfo.Country,
					strings.ToUpper(strings.ReplaceAll(result.MatchedName, " ", "-")))

				if result.Reason == "direct_match" || result.Reason == "alt_name_match" {
					mapping := &models.APIMapping{
						APIName:    "api_sports",
						APIID:      strconv.Itoa(league.ID),
						EntityType: "league",
						EntityID:   result.InternalID,
					}

					if err := store.UpsertAPIMapping(mapping); err != nil {
						log.Printf("Warning: failed to create mapping for league %s: %v", result.InternalID, err)
					}
				}
			}
		}

		results = append(results, result)
	}

	return results, nil
}

func (c *Client) FetchAndStoreLeagues(store *db.Store, updateImages bool) ([]leagueChange, error) {
	url := "https://v1.rugby.api-sports.io/leagues"
	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Response []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
			Logo string `json:"logo"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	var changes []leagueChange
	for _, league := range apiResp.Response {
		cleanName := rugbydb.CleanLeagueName(league.Name)
		if countryInfo, ok := rugbydb.LeagueCountryMap[cleanName]; ok {
			leagueID := fmt.Sprintf("%s-%s", countryInfo.Country,
				strings.ToUpper(strings.ReplaceAll(cleanName, " ", "-")))

			dbLeague := &models.League{
				ID:      leagueID,
				Name:    league.Name,
				Format:  league.Type,
				LogoURL: league.Logo,
				Country: models.Country{
					Code: countryInfo.Country,
				},
			}

			change := leagueChange{
				IsNew:      true,
				Changes:    make(map[string]map[string]string),
				LeagueName: league.Name,
			}

			existing, err := store.GetLeagueByID(leagueID)
			if err == nil && existing != nil {
				change.IsNew = false
				if existing.Name != dbLeague.Name {
					change.Changes["name"] = map[string]string{"old": existing.Name, "new": dbLeague.Name}
				}
				if existing.LogoURL != dbLeague.LogoURL {
					change.Changes["logo"] = map[string]string{"old": existing.LogoURL, "new": dbLeague.LogoURL}
				}
			}

			if err := store.UpsertLeague(dbLeague); err != nil {
				log.Printf("Error upserting league %s: %v", league.Name, err)
				continue
			}

			mapping := &models.APIMapping{
				EntityID:   leagueID,
				APIName:    "api_sports",
				APIID:      strconv.Itoa(league.ID),
				EntityType: "league",
			}

			if err := store.UpsertAPIMapping(mapping); err != nil {
				log.Printf("Error creating API mapping for league %s: %v", league.Name, err)
			}

			if change.IsNew {
				log.Printf("Added new league: %s", league.Name)
			} else if len(change.Changes) > 0 {
				log.Printf("Updated league: %s with changes: %v", league.Name, change.Changes)
			}

			changes = append(changes, change)
		}
	}

	return changes, nil
}

func (c *Client) GetLeagueIDsByYear(year string, store *db.Store) ([]models.League, error) {
	var leagues []models.League
	for name, countryInfo := range rugbydb.LeagueCountryMap {
		// For now, return all leagues since we don't have year info
		leagueID := fmt.Sprintf("%s-%s", countryInfo.Country,
			strings.ToUpper(strings.ReplaceAll(name, " ", "-")))

		league, err := store.GetLeagueByID(leagueID)
		if err != nil {
			continue
		}

		leagues = append(leagues, *league)
	}
	return leagues, nil
}

func (c *Client) ScrapeESPNLeagues() ([]models.League, error) {
	// This is a placeholder - the original function was for scraping ESPN
	// If you want to implement ESPN scraping, we can add it here
	return nil, fmt.Errorf("ESPN scraping not implemented")
}
