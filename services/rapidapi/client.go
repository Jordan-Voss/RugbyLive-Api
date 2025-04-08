package rapidapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"time"
)

type Client struct {
	client *http.Client
	apiKey string
}

// Add this map at package level
var splitYearLeagues = map[string]bool{
	"United Rugby Championship":    true,
	"Top 14":                       true,
	"Premiership":                  true,
	"European Rugby Champions Cup": true,
	"European Rugby Challenge Cup": true,
	"Six Nations":                  false,
	"The Rugby Championship":       false,
	"Super Rugby":                  false,
	"Pro D2":                       true,
	"EPCR Challenge Cup":           true,
	"Super W (W)":                  false, // Not a split year league
	"Pacific Four Series (W)":      false, // Not a split year league
}

// Add activeLeagues map at package level
var activeLeagues = map[string]bool{
	"Top 14":                       true,
	"United Rugby Championship":    true,
	"Premiership":                  true,
	"European Champions Cup":       true,
	"European Rugby Challenge Cup": true,
	"Six Nations":                  true,
	"The Rugby Championship":       true,
	"Super Rugby Pacific":          true,
	"Pro D2":                       true,
	"EPCR Challenge Cup":           true,
	"Super W (W)":                  true, // Mark as active
	"Pacific Four Series (W)":      true, // Mark as active
}

// Add defaultLeagues map at package level
var defaultLeagues = map[string]models.League{
	"Super W": {
		ID:            "AUS-SUPER-W-(W)",
		Name:          "Super W (W)",
		Country:       models.Country{Code: "AUS"},
		International: false,
		Format:        "Hybrid",
		Tier:          1,
		TeamCountries: []models.Country{{Code: "AUS"}, {Code: "FJI"}},
		Phases:        []string{"League", "Playoffs"},
		Gender:        "Women",
	},
	"Pacific Four Series": {
		ID:            "WLD-PACIFIC-FOUR-SERIES-(W)",
		Name:          "Pacific Four Series (W)",
		Country:       models.Country{Code: "WLD"},
		International: true,
		Format:        "League",
		Tier:          1,
		TeamCountries: []models.Country{{Code: "AUS"}, {Code: "CAN"}, {Code: "NZL"}, {Code: "USA"}},
		Gender:        "Women",
	},
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
		apiKey: os.Getenv("RAPID_API_KEY"),
	}
}

func (c *Client) GetCompetitions() ([]models.RapidAPICompetition, error) {
	req, err := http.NewRequest("GET", "https://rugby-live-data.p.rapidapi.com/competitions", nil)
	if err != nil {
		return nil, err
	}
	log.Printf("RapidAPI Key: %s", c.apiKey)
	req.Header.Add("X-RapidAPI-Host", "rugby-live-data.p.rapidapi.com")
	req.Header.Add("X-RapidAPI-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and log the raw response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Create new reader for JSON decoding
	var result struct {
		Results []models.RapidAPICompetition `json:"results"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result.Results, nil
}

func (c *Client) MapCompetitionsToLeagues(store *db.Store) ([]CompetitionMapping, error) {
	competitions, err := c.GetCompetitions()
	if err != nil {
		return nil, err
	}

	// Group competitions by name
	compsByName := make(map[string]CompetitionGroup)
	for _, comp := range competitions {
		cleanName := standardizeCompetitionName(comp.Name)
		group := compsByName[cleanName]
		group.Name = cleanName
		group.RapidAPIID = comp.ID

		startYear := parseSeasonYears(comp.SeasonName)
		season := models.Season{
			Year:         startYear,
			RapidAPIYear: startYear,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		league, err := store.GetLeagueByName(cleanName)
		if err == nil {
			season.LeagueID = league.ID

			// Adjust internal year for split year leagues
			internalYear := startYear
			if splitYearLeagues[cleanName] {
				internalYear-- // Use previous year for split year leagues
			}
			season.Year = internalYear
			season.ID = fmt.Sprintf("%s-SEASON-%d", league.ID, internalYear)

			isSplitYear := splitYearLeagues[cleanName]
			if isSplitYear {
				season.YearRange = fmt.Sprintf("%d-%d", internalYear, internalYear+1)
				season.StartDate = time.Date(internalYear, 8, 1, 0, 0, 0, 0, time.UTC)
				season.EndDate = time.Date(internalYear+1, 5, 31, 0, 0, 0, 0, time.UTC)
			} else {
				season.YearRange = fmt.Sprintf("%d", internalYear)
				season.StartDate = time.Date(internalYear, 1, 1, 0, 0, 0, 0, time.UTC)
				season.EndDate = time.Date(internalYear, 12, 31, 0, 0, 0, 0, time.UTC)
			}
		}
		group.Seasons = append(group.Seasons, season)
		compsByName[cleanName] = group
	}

	// Create mappings
	var mappings []CompetitionMapping
	for name, group := range compsByName {
		league, err := store.GetLeagueByName(name)
		if err != nil {
			// Check if we should auto-create this league
			if defaultLeague, exists := defaultLeagues[name]; exists {
				if err := store.UpsertLeague(&defaultLeague); err != nil {
					log.Printf("Error creating default league %s: %v", name, err)
				} else {
					league = &defaultLeague
					err = nil // Clear the error since we created the league
					log.Printf("Created default league: %s", name)
				}
			}
		}
		mapping := CompetitionMapping{
			CompetitionGroup: group,
			Matched:          err == nil,
			League:           league,
			Reason:           "no_match",
		}

		// Only process database operations if we have a matching league
		if err == nil {
			mapping.Reason = "direct_match"
			log.Printf("Found matching league: %s (ID: %s)", name, league.ID)

			// Add/update league API mapping
			leagueMapping := &models.APIMapping{
				APIName:    "rapid_api",
				APIID:      fmt.Sprintf("%d", group.RapidAPIID),
				EntityID:   league.ID,
				EntityType: "league",
				IsActive:   activeLeagues[name],
			}
			if err := store.UpsertRapidAPIMapping(leagueMapping); err != nil {
				log.Printf("Error upserting league mapping: %v", err)
			} else {
				log.Printf("Successfully upserted league mapping: %s -> %s", leagueMapping.APIID, leagueMapping.EntityID)
			}
			mapping.APIMappings = append(mapping.APIMappings, leagueMapping)

			// Process seasons for matched leagues only
			for _, season := range group.Seasons {
				log.Printf("Processing season %d for league %s", season.Year, name)

				var seasonID string
				// First check and create season if needed
				existingSeason, err := store.GetSeasonByYear(league.ID, season.Year)
				if err != nil {
					log.Printf("Season %d not found, creating new season", season.Year)
					if err := store.UpsertSeason(&season); err != nil {
						log.Printf("Error upserting season: %v", err)
						continue
					}
					log.Printf("Successfully created season: %s (Year: %d, RapidAPIYear: %d)",
						season.ID, season.Year, season.RapidAPIYear)
					seasonID = season.ID
				} else {
					log.Printf("Season %d already exists", season.Year)
					seasonID = existingSeason.ID
				}

				// Now create/update the season API mapping using the correct season ID
				seasonMapping := &models.APIMapping{
					APIName:    "rapid_api",
					APIID:      fmt.Sprintf("%d-%d", group.RapidAPIID, season.RapidAPIYear),
					EntityID:   seasonID,
					EntityType: "league_season",
					IsActive:   activeLeagues[name],
				}

				log.Printf("Creating mapping for season: %+v", seasonMapping)
				if err := store.UpsertRapidAPIMapping(seasonMapping); err != nil {
					log.Printf("Error upserting season mapping: %v", err)
				} else {
					log.Printf("Successfully upserted season mapping: %s -> %s (ID: %s)",
						seasonMapping.APIID, seasonMapping.EntityID, seasonID)
				}
				mapping.APIMappings = append(mapping.APIMappings, seasonMapping)
			}
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

func parseSeasonYears(seasonName string) int {
	// Parse "Season 2017/2018" format
	var sy, ey int
	n, _ := fmt.Sscanf(seasonName, "Season %d/%d", &sy, &ey)
	if n == 2 {
		return sy
	}
	// Parse "Season 2017" format
	n, _ = fmt.Sscanf(seasonName, "Season %d", &sy)
	if n == 1 {
		return sy
	}
	return 0
}

func standardizeCompetitionName(name string) string {
	// Add common name mappings
	nameMap := map[string]string{
		"T14":                          "Top 14",
		"Guinness Pro14":               "United Rugby Championship",
		"RaboDirect Pro 12":            "United Rugby Championship",
		"Guinness Pro12":               "United Rugby Championship",
		"Heineken Cup":                 "European Rugby Champions Cup",
		"European Challenge Cup":       "EPCR Challenge Cup",
		"Super Rugby":                  "Super Rugby Pacific",
		"Super Rugby Pacific":          "Super Rugby Pacific",
		"European Rugby Champions Cup": "European Champions Cup",
		"Heineken Champions Cup":       "European Champions Cup",
		"Investec Champions Cup":       "European Champions Cup",
		"European Rugby Challenge Cup": "EPCR Challenge Cup",
		"Farah Palmer Cup":             "Farah Palmer Cup (W)",
		"Women's Six Nations":          "Women's Six Nations Championship (W)",
		"British & Irish Lions":        "British & Irish Lions Tour",
		"Six Nations":                  "Six Nations Championship",
		"Aviva Premiership":            "Premiership Rugby",
		"Super W":                      "Super W (W)",
		"Championship":                 "RFU Championship",
		"Super Rugby Aupiki":           "Super Rugby Aupiki (W)",
		"Pacific Four Series":          "Pacific Four Series (W)",
		"Japan Rugby League One D1":    "Japan Rugby League One - Division 1",
		"Japan Rugby League One D2":    "Japan Rugby League One - Division 2",
		"Japan Rugby League One D3":    "Japan Rugby League One - Division 3",
	}

	if standardName, exists := nameMap[name]; exists {
		return standardName
	}

	return name
}
