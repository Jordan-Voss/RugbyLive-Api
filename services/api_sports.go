package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"rugby-live-api/services/rugbydb"
	"strconv"
	"strings"
	"time"
)

type APISportsTeam struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Logo  string `json:"logo"`
	Score int    `json:"score"`
}

type APISportsTeams struct {
	Home APISportsTeam `json:"home"`
	Away APISportsTeam `json:"away"`
}

type APISportsMatch struct {
	ID       string         `json:"id"`
	Date     string         `json:"date"`
	LeagueID string         `json:"league_id"`
	Status   string         `json:"status"`
	Teams    APISportsTeams `json:"teams"`
}

type APIParams struct {
	LeagueID string
	Season   string
	Date     string
}

type Match struct {
	ID                string          `json:"id"`
	HomeTeamID        string          `json:"home_team_id"`
	AwayTeamID        string          `json:"away_team_id"`
	LeagueID          string          `json:"league_id"`
	HomeScore         int             `json:"home_score"`
	AwayScore         int             `json:"away_score"`
	Status            string          `json:"status"`
	KickOff           time.Time       `json:"kick_off"`
	Date              string          `json:"date"`
	Time              string          `json:"time"`
	Venue             string          `json:"venue,omitempty"`
	Referee           string          `json:"referee,omitempty"`
	Attendance        int             `json:"attendance,omitempty"`
	WeatherConditions string          `json:"weather_conditions,omitempty"`
	HeadToHead        json.RawMessage `json:"head_to_head,omitempty"`
	Lineups           json.RawMessage `json:"lineups,omitempty"`
	LiveStats         json.RawMessage `json:"live_stats,omitempty"`
}

type DailyMatches struct {
	Date     string   `json:"date"`
	MatchIDs []string `json:"match_ids"`
}

func (a *APIClient) FetchFromAPISports() ([]models.Match, error) {
	today := time.Now().Format("2006-01-02")
	url := fmt.Sprintf("https://v1.rugby.api-sports.io/games?date=%s", today)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("API_SPORTS_KEY"))
	req.Header.Add("x-rapidapi-host", "v1.rugby.api-sports.io")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp models.APISportsTodaysMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return a.standardizeAPISportsData(apiResp), nil
}

func (a *APIClient) standardizeAPISportsData(resp models.APISportsTodaysMatchesResponse) []models.Match {
	var matches []models.Match
	for _, game := range resp.Response {
		kickOff, _ := time.Parse("2006-01-02T15:04:05-07:00", game.Date)

		homeTeam := &models.Team{
			ID:      fmt.Sprintf("%s-%s", game.Country.Code, strings.ToUpper(strings.ReplaceAll(game.Teams.Home.Name, " ", ""))),
			Name:    game.Teams.Home.Name,
			LogoURL: game.Teams.Home.Logo,
			Country: models.Country{
				Code: game.Country.Code,
				Name: game.Country.Name,
				Flag: game.Country.Flag,
			},
		}

		awayTeam := &models.Team{
			ID:      fmt.Sprintf("%s-%s", game.Country.Code, strings.ToUpper(strings.ReplaceAll(game.Teams.Away.Name, " ", ""))),
			Name:    game.Teams.Away.Name,
			LogoURL: game.Teams.Away.Logo,
			Country: models.Country{
				Code: game.Country.Code,
				Name: game.Country.Name,
				Flag: game.Country.Flag,
			},
		}

		league := &models.League{
			ID: fmt.Sprintf("%s-%s", game.Country.Code, strings.ToUpper(strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(game.League.Name, " ", "-"),
					"'", "",
				),
				".", "",
			))),
			Name:    game.League.Name,
			Format:  game.League.Type,
			LogoURL: game.League.Logo,
			Country: models.Country{
				Code: game.Country.Code,
				Name: game.Country.Name,
				Flag: game.Country.Flag,
			},
			Seasons: []models.Season{{
				Year:      game.League.Season,
				Current:   true,
				StartDate: time.Now(),
				EndDate:   time.Now().AddDate(0, 6, 0),
			}},
		}

		match := models.Match{
			ID:          fmt.Sprintf("%s-%s-%s", kickOff.Format("2006-01-02"), homeTeam.ID, awayTeam.ID),
			HomeTeam:    homeTeam,
			AwayTeam:    awayTeam,
			League:      league,
			HomeScore:   game.Scores.Home,
			AwayScore:   game.Scores.Away,
			Status:      game.Status.Long,
			KickOff:     kickOff,
			Date:        kickOff.Format("2006-01-02"),
			Time:        kickOff.Format("15:04"),
			Week:        game.Week,
			Season:      game.League.Season,
			APISportsID: game.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		matches = append(matches, match)
		log.Printf("Processing match: %d %s vs %s in %s (%s)",
			match.APISportsID,
			homeTeam.Name,
			awayTeam.Name,
			league.Name,
			match.Status,
		)
	}
	return matches
}

func (a *APIClient) FetchAndStoreCountries(store *db.Store, updateFlags bool) ([]countryChange, error) {
	url := "https://v1.rugby.api-sports.io/countries"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("API_SPORTS_KEY"))
	req.Header.Add("x-rapidapi-host", "v1.rugby.api-sports.io")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var countriesResp CountriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&countriesResp); err != nil {
		return nil, err
	}

	var changes []countryChange

	for _, c := range countriesResp.Response {
		countryCode := c.Code
		switch c.Name {
		case "Australia-Oceania":
			countryCode = "OC"
		case "England":
			countryCode = "ENG"
		case "Scotland":
			countryCode = "SCO"
		case "Wales":
			countryCode = "WAL"
		case "Northern Ireland":
			countryCode = "NIR"
		case "Europe":
			countryCode = "EU"
		case "World":
			countryCode = "WRLD"
		}

		existing, err := store.GetCountryByCode(countryCode)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("error checking existing country: %v", err)
		}

		flagURL := c.Flag
		if existing == nil {
			newFlagURL, err := a.downloadAndStoreImage(c.Flag, fmt.Sprintf("flags/%s.svg", strings.ToLower(countryCode)))
			if err != nil {
				log.Printf("Error downloading flag for %s: %v", countryCode, err)
			} else {
				flagURL = newFlagURL
			}
		} else if updateFlags && existing.FlagSource == "api_sports" && existing.Flag != c.Flag {
			newFlagURL, err := a.downloadAndStoreImage(c.Flag, fmt.Sprintf("flags/%s.svg", strings.ToLower(countryCode)))
			if err != nil {
				log.Printf("Error downloading flag for %s: %v", countryCode, err)
			} else {
				flagURL = newFlagURL
			}
		}

		country := &models.Country{
			Code:       countryCode,
			Name:       c.Name,
			Flag:       flagURL,
			FlagSource: "api_sports",
		}

		// Track changes
		change := countryChange{
			Name:    c.Name,
			Code:    countryCode,
			Changes: make(map[string]interface{}),
			IsNew:   existing == nil,
		}

		if existing != nil {
			if existing.Name != country.Name {
				change.Changes["name"] = map[string]string{"old": existing.Name, "new": country.Name}
				if err := store.UpsertCountry(country); err != nil {
					log.Printf("Error upserting country %s: %v", countryCode, err)
					continue
				}
			}
			if existing.Flag != country.Flag {
				change.Changes["flag"] = map[string]string{"old": existing.Flag, "new": country.Flag}
				if err := store.UpsertCountry(country); err != nil {
					log.Printf("Error upserting country %s: %v", countryCode, err)
					continue
				}
			}
		} else if existing == nil {
			if err := store.UpsertCountry(country); err != nil {
				log.Printf("Error upserting country %s: %v", countryCode, err)
				continue
			}
			changes = append(changes, change)
		}

		if len(change.Changes) > 0 {
			changes = append(changes, change)
		}

		// Store API mapping
		mapping := &models.APIMapping{
			EntityID:   countryCode,
			APIName:    "api_sports",
			APIID:      fmt.Sprintf("%d", c.ID),
			EntityType: "country",
		}
		if err := store.UpsertAPIMapping(mapping); err != nil {
			log.Printf("Error creating API mapping for country %s: %v", countryCode, err)
		}
	}

	return changes, nil
}

func (a *APIClient) FetchAndStoreLeagues(store *db.Store, updateImages bool) ([]leagueChange, error) {
	url := "https://v1.rugby.api-sports.io/leagues"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("API_SPORTS_KEY"))
	req.Header.Add("x-rapidapi-host", "v1.rugby.api-sports.io")

	// Increase timeout for large requests
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var leaguesResp APISportsLeaguesResponse
	if err := json.NewDecoder(resp.Body).Decode(&leaguesResp); err != nil {
		return nil, err
	}

	var changes []leagueChange

	// Process leagues in batches
	batchSize := 10
	for i := 0; i < len(leaguesResp.Response); i += batchSize {
		end := i + batchSize
		if end > len(leaguesResp.Response) {
			end = len(leaguesResp.Response)
		}

		log.Printf("Processing leagues %d to %d of %d", i+1, end, len(leaguesResp.Response))
		batch := leaguesResp.Response[i:end]
		for _, l := range batch {
			log.Printf("Processing leagues for %s (Country code: %s)", l.Country.Name, l.Country.Code)
			// Clean up country code
			countryCode := l.Country.Code
			switch l.Country.Name {
			case "Australia-Oceania":
				countryCode = "OC"
			case "England":
				countryCode = "ENG"
			case "Scotland":
				countryCode = "SCO"
			case "Wales":
				countryCode = "WAL"
			case "Northern Ireland":
				countryCode = "NIR"
			case "Europe":
				countryCode = "EU"
			case "World":
				countryCode = "WRLD"
			}

			// Create a clean league name for the ID
			cleanLeagueName := strings.ToUpper(strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(l.Name, " ", "-"),
					"'", "",
				),
				".", "",
			))

			// First ensure country exists
			country := &models.Country{
				Code: countryCode,
				Name: l.Country.Name,
				Flag: l.Country.Flag,
			}

			existing, err := store.GetLeagueByID(fmt.Sprintf("%s-%s", countryCode, cleanLeagueName))
			if err != nil && err != sql.ErrNoRows {
				log.Printf("Error checking existing league %s: %v", l.Name, err)
				continue
			}

			change := leagueChange{
				Name:    l.Name,
				ID:      fmt.Sprintf("%s-%s", countryCode, cleanLeagueName),
				Changes: make(map[string]interface{}),
				IsNew:   existing == nil,
			}

			logoURL := l.Logo
			if existing == nil || (updateImages && existing.LogoSource == "api_sports" && existing.LogoURL != l.Logo) {
				// Clean the league name for the filename
				cleanName := strings.ToLower(strings.ReplaceAll(
					strings.ReplaceAll(
						strings.ReplaceAll(l.Name, " ", "-"),
						"'", "",
					),
					".", "",
				))

				newLogoURL, err := a.downloadAndStoreImage(l.Logo, fmt.Sprintf("logos/leagues/%s/%s.png", countryCode, cleanName))
				if err != nil {
					log.Printf("Error downloading logo for %s: %v", l.Name, err)
				} else {
					logoURL = newLogoURL
					if existing != nil {
						change.Changes["logo"] = map[string]string{"old": existing.LogoURL, "new": logoURL}
					}
				}
			}

			// Create league with all seasons
			var seasons []models.Season
			for _, s := range l.Seasons {
				start, _ := time.Parse("2006-01-02", s.Start)
				end, _ := time.Parse("2006-01-02", s.End)
				seasons = append(seasons, models.Season{
					Year:      s.Year,
					Current:   s.Current,
					StartDate: start,
					EndDate:   end,
				})
			}

			league := &models.League{
				ID:         change.ID,
				Name:       l.Name,
				Format:     l.Type,
				LogoURL:    logoURL,
				LogoSource: "api_sports",
				Country:    *country,
				Seasons:    seasons,
			}

			if existing != nil {
				// Compare seasons using a map for easier lookup
				existingSeasons := make(map[int]models.Season)
				for _, s := range existing.Seasons {
					existingSeasons[s.Year] = s
				}

				for _, newSeason := range seasons {
					if oldSeason, exists := existingSeasons[newSeason.Year]; exists {
						if newSeason.Current != oldSeason.Current {
							change.Changes[fmt.Sprintf("season_%d", newSeason.Year)] = map[string]interface{}{
								"type": "status_change",
								"old":  oldSeason.Current,
								"new":  newSeason.Current,
							}
						}
					} else {
						change.Changes[fmt.Sprintf("season_%d", newSeason.Year)] = map[string]interface{}{
							"type": "new_season",
							"year": newSeason.Year,
						}
					}
				}

				if existing.Name != league.Name {
					change.Changes["name"] = map[string]string{"old": existing.Name, "new": league.Name}
				}
				if existing.Format != league.Format {
					change.Changes["type"] = map[string]string{"old": existing.Format, "new": league.Format}
				}
			}

			if err := store.UpsertLeague(league); err != nil {
				log.Printf("Error upserting league %s: %v", l.Name, err)
				continue
			}

			if change.IsNew {
				log.Printf("Added new league: %s", l.Name)
			} else if len(change.Changes) > 0 {
				log.Printf("Updated league: %s with changes: %v", l.Name, change.Changes)
			}

			// Store API mapping for the league
			mapping := &models.APIMapping{
				EntityID:   league.ID,
				APIName:    "api_sports",
				APIID:      fmt.Sprintf("%d", l.ID),
				EntityType: "league",
			}

			if err := store.UpsertAPIMapping(mapping); err != nil {
				log.Printf("Error creating API mapping for league %s: %v", l.Name, err)
			}
		}
	}

	return changes, nil
}

func (a *APIClient) FetchAndStoreTeams(store *db.Store, updateImages bool, params TeamSearchParams) ([]teamChange, []failedTeam, error) {
	var allChanges []teamChange
	var allFailedTeams []failedTeam

	// Rate limiting
	rateLimiter := time.NewTicker(time.Second / 10) // 10 requests per second
	defer rateLimiter.Stop()

	if params.CountryID != "" {
		// Fetch teams for specific country
		url := fmt.Sprintf("https://v1.rugby.api-sports.io/teams?country_id=%s", params.CountryID)
		changes, failedTeams, err := a.fetchTeamsForCountry(store, url, updateImages)
		if err != nil {
			return nil, nil, err
		}
		allChanges = append(allChanges, changes...)
		allFailedTeams = append(allFailedTeams, failedTeams...)
	} else {
		// Fetch teams for all countries
		countryMappings, err := store.GetAPIMappingsByType("api_sports", "country")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get country mappings: %v", err)
		}

		for _, mapping := range countryMappings {
			log.Printf("Fetching teams for country: %s (API ID: %s)", mapping.EntityID, mapping.APIID)
			<-rateLimiter.C

			url := fmt.Sprintf("https://v1.rugby.api-sports.io/teams?country_id=%s", mapping.APIID)
			changes, failedTeams, err := a.fetchTeamsForCountry(store, url, updateImages)
			if err != nil {
				log.Printf("Error fetching teams for country %s: %v", mapping.EntityID, err)
				continue
			}
			log.Printf("Fetched %d teams for country %s", len(changes), mapping.EntityID)
			allChanges = append(allChanges, changes...)
			allFailedTeams = append(allFailedTeams, failedTeams...)
		}
	}

	if len(allFailedTeams) > 0 {
		log.Printf("\nTeams that could not be processed:")
		for _, ft := range allFailedTeams {
			log.Printf("- %s\n  Country: (ID: %d, Name: %s)\n  Reason: %s\n  Data: %+v\n",
				ft.Name, ft.CountryID, ft.CountryName, ft.Reason, ft.TeamData)
		}
		log.Printf("Total failed teams: %d", len(allFailedTeams))
	}

	return allChanges, allFailedTeams, nil
}

func (a *APIClient) fetchTeamsForCountry(store *db.Store, url string, updateImages bool) ([]teamChange, []failedTeam, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("API_SPORTS_KEY"))
	req.Header.Add("x-rapidapi-host", "v1.rugby.api-sports.io")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var teamsResp TeamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&teamsResp); err != nil {
		return nil, nil, err
	}
	log.Printf("Teams response: %+v", teamsResp)

	if teamsResp.Errors != nil {
		if errArray, ok := teamsResp.Errors.([]interface{}); ok {
			if len(errArray) > 0 {
				log.Printf("API returned errors: %v", teamsResp.Errors)
				return nil, nil, fmt.Errorf("API errors: %v", teamsResp.Errors)
			}
		} else if errMap, ok := teamsResp.Errors.(map[string]string); ok && len(errMap) > 0 {
			log.Printf("API returned errors: %v", teamsResp.Errors)
			return nil, nil, fmt.Errorf("API errors: %v", teamsResp.Errors)
		}
	}

	if len(teamsResp.Response) == 0 {
		log.Printf("No teams returned for this country")
		return nil, nil, nil
	}

	var changes []teamChange
	var failedTeams []failedTeam

	log.Printf("Processing %d teams from response", len(teamsResp.Response))

	for _, t := range teamsResp.Response {
		if t.Country.ID == 0 || t.Country.Code == "" {
			failedTeams = append(failedTeams, failedTeam{
				Name:        t.Name,
				CountryID:   t.Country.ID,
				CountryName: t.Country.Name,
				Reason:      "Missing or invalid country data",
				TeamData:    t,
			})
			continue
		}

		mapping, err := store.GetAPIMappingByAPIID("api_sports", fmt.Sprintf("%d", t.Country.ID), "country")
		if err != nil {
			failedTeams = append(failedTeams, failedTeam{
				Name:        t.Name,
				CountryID:   t.Country.ID,
				CountryName: t.Country.Name,
				Reason:      fmt.Sprintf("No country mapping found: %v", err),
				TeamData:    t,
			})
			continue
		}
		countryCode := mapping.EntityID

		country, err := store.GetCountryByCode(countryCode)
		if err != nil {
			log.Printf("Error getting country %s: %v", countryCode, err)
			continue
		}

		log.Printf("Processing team: %s for country: %s", t.Name, countryCode)

		teamID := fmt.Sprintf("%s-%s", countryCode, strings.ToUpper(strings.ReplaceAll(t.Name, " ", "")))
		existing, err := store.GetTeamByID(teamID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existing team %s: %v", t.Name, err)
			continue
		}

		change := teamChange{
			Name:    t.Name,
			ID:      teamID,
			Changes: make(map[string]interface{}),
			IsNew:   existing == nil,
		}

		logoURL := t.Logo
		if existing == nil || (updateImages && existing.LogoSource == "api_sports" && existing.LogoURL != t.Logo) {
			newLogoURL, err := a.downloadAndStoreImage(
				t.Logo,
				fmt.Sprintf("logos/teams/%s/%s/logo.png", countryCode, strings.TrimPrefix(teamID, countryCode+"-")),
			)
			if err != nil {
				log.Printf("Error downloading logo for team %s: %v", t.Name, err)
			} else {
				logoURL = newLogoURL
				if existing != nil {
					change.Changes["logo"] = map[string]string{"old": existing.LogoURL, "new": logoURL}
				}
			}
		}

		var stadiums []models.TeamStadium
		if t.Arena.Name != "" {
			cleanArenaName := strings.ToUpper(strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(t.Arena.Name, ",", ""),
					" ", "-"),
				"'", "",
			))

			location := t.Arena.Location
			if idx := strings.Index(location, ","); idx > 0 {
				location = location[:idx]
			}
			cleanLocation := strings.ToUpper(strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(location, ",", ""),
					" ", "-"),
				"'", "",
			))

			stadium := &models.Stadium{
				ID:       fmt.Sprintf("%s-%s-%s", countryCode, cleanLocation, cleanArenaName),
				Name:     t.Arena.Name,
				Location: location,
				Country:  *country,
			}

			if cap, ok := t.Arena.Capacity.(float64); ok {
				stadium.Capacity = int(cap)
			}

			if err := store.UpsertStadium(stadium); err != nil {
				log.Printf("Error upserting stadium for team %s: %v", t.Name, err)
			} else {
				stadiums = append(stadiums, models.TeamStadium{
					Stadium:   *stadium,
					IsPrimary: true,
				})
			}
		}

		team := &models.Team{
			ID:       teamID,
			Name:     t.Name,
			LogoURL:  logoURL,
			Country:  *country,
			Stadiums: stadiums,
		}

		if existing != nil {
			if existing.Name != team.Name {
				change.Changes["name"] = map[string]string{"old": existing.Name, "new": team.Name}
			}
			if existing.LogoURL != team.LogoURL {
				change.Changes["logo"] = map[string]string{"old": existing.LogoURL, "new": team.LogoURL}
			}
		}

		if err := store.UpsertTeam(team); err != nil {
			failedTeams = append(failedTeams, failedTeam{
				Name:        t.Name,
				CountryID:   t.Country.ID,
				CountryName: t.Country.Name,
				Reason:      fmt.Sprintf("Failed to upsert: %v", err),
				TeamData:    t,
			})
			continue
		}

		if len(team.Stadiums) > 0 {
			for _, stadium := range team.Stadiums {
				if err := store.UpsertTeamStadium(team.ID, &stadium); err != nil {
					log.Printf("Error upserting team stadium relationship for team %s: %v", team.Name, err)
				}
			}
		}

		if change.IsNew {
			log.Printf("Added new team: %s", t.Name)
		} else if len(change.Changes) > 0 {
			log.Printf("Updated team: %s with changes: %v", t.Name, change.Changes)
		}

		teamMapping := &models.APIMapping{
			EntityID:   team.ID,
			APIName:    "api_sports",
			APIID:      fmt.Sprintf("%d", t.ID),
			EntityType: "team",
		}

		if err := store.UpsertAPIMapping(teamMapping); err != nil {
			log.Printf("Error creating API mapping for team %s: %v", t.Name, err)
		}
	}

	return changes, failedTeams, nil
}

// Add new function to fetch and map leagues
func (a *APIClient) MapAPISportsLeagues(store *db.Store) ([]LeagueMappingResult, error) {
	url := "https://v1.rugby.api-sports.io/leagues"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("API_SPORTS_KEY"))
	req.Header.Add("x-rapidapi-host", "v1.rugby.api-sports.io")

	resp, err := a.client.Do(req)
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
			APILeague: APILeague{
				ID:   strconv.Itoa(league.ID),
				Name: league.Name,
				Type: league.Type,
			},
			Matched:     false,
			MatchedName: "",
			Reason:      "no_match",
		}

		// Try to match with our league mappings
		cleanName := rugbydb.CleanLeagueName(league.Name)
		matchFound := false

		if _, exists := rugbydb.LeagueCountryMap[cleanName]; exists {
			matchFound = true
			result.Matched = true
			result.MatchedName = cleanName
			result.Reason = "direct_match"
		}

		if !matchFound {
			// Check alt names
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

				// Create API mapping for both direct and alt name matches
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

func (a *APIClient) GetMatchesByLeague(leagueID string, date string, season string, apiParams APIParams, store *db.Store) ([]Match, []*DailyMatches, error) {
	// Get API Sports league ID from our internal ID if not provided
	apiLeagueID := apiParams.LeagueID
	if apiLeagueID == "" {
		mapping, err := store.GetAPIMappingByEntityID("api_sports", leagueID, "league")
		if err != nil || mapping == nil {
			return nil, nil, fmt.Errorf("league not found in API Sports mappings")
		}
		apiLeagueID = mapping.APIID
	}

	// Get season from database
	dbSeason, err := store.GetSeasonByLeagueAndYear(leagueID, season)
	if err != nil {
		return nil, nil, fmt.Errorf("season not found: %v", err)
	}

	// Build URL with parameters
	params := make([]string, 0)
	params = append(params, fmt.Sprintf("league=%s", apiLeagueID))
	apiSeason := apiParams.Season
	if apiSeason == "" {
		apiSeason = season
	}
	if apiSeason != "" {
		params = append(params, fmt.Sprintf("season=%s", apiSeason))
	}
	if apiParams.Date != "" {
		params = append(params, fmt.Sprintf("date=%s", apiParams.Date))
	}

	url := fmt.Sprintf("https://v1.rugby.api-sports.io/games?%s", strings.Join(params, "&"))

	log.Printf("Calling API Sports URL: %s", url)

	// Reuse existing API response struct and processing logic
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("API_SPORTS_KEY"))
	req.Header.Add("x-rapidapi-host", "v1.rugby.api-sports.io")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// Read and log the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("API Sports response: %s", string(respBody))

	// Create a new reader with the response body for json.Decode
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// Use the same response processing as GetMatchesByDate
	var apiResp struct {
		Response []struct {
			ID     int    `json:"id"`
			Date   string `json:"date"`
			Status struct {
				Long string `json:"long"`
			} `json:"status"`
			League struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"league"`
			Teams struct {
				Home struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Logo string `json:"logo"`
				} `json:"home"`
				Away struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Logo string `json:"logo"`
				} `json:"away"`
			} `json:"teams"`
			Scores struct {
				Home int `json:"home"`
				Away int `json:"away"`
			} `json:"scores"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, nil, err
	}

	// Process matches using same logic as GetMatchesByDate
	var matches []Match
	for _, m := range apiResp.Response {
		// Get team mappings
		homeTeamMapping, err := store.GetAPIMappingByAPIID("api_sports", strconv.Itoa(m.Teams.Home.ID), "team")
		if err != nil || homeTeamMapping == nil {
			continue
		}

		awayTeamMapping, err := store.GetAPIMappingByAPIID("api_sports", strconv.Itoa(m.Teams.Away.ID), "team")
		if err != nil || awayTeamMapping == nil {
			continue
		}

		status := "upcoming"
		if m.Status.Long == "Finished" {
			status = "finished"
		} else if m.Status.Long == "In Play" {
			status = "live"
		}

		kickOff, _ := time.Parse("2006-01-02T15:04:05-07:00", m.Date)
		matchID := fmt.Sprintf("%s-%s-%s-%s",
			dbSeason.ID,
			homeTeamMapping.EntityID,
			awayTeamMapping.EntityID,
			kickOff.Format("20060102"),
		)

		match := Match{
			ID:         matchID,
			HomeTeamID: homeTeamMapping.EntityID,
			AwayTeamID: awayTeamMapping.EntityID,
			LeagueID:   dbSeason.ID,
			HomeScore:  m.Scores.Home,
			AwayScore:  m.Scores.Away,
			Status:     status,
			KickOff:    kickOff,
			Date:       kickOff.Format("2006-01-02"),
			Time:       kickOff.Format("15:04"),
		}
		matches = append(matches, match)

		// Create API mapping for match
		matchMapping := &models.APIMapping{
			EntityID:   matchID,
			APIName:    "api_sports",
			APIID:      strconv.Itoa(m.ID),
			EntityType: "match",
		}
		if err := store.UpsertAPIMapping(matchMapping); err != nil {
			log.Printf("Error creating API mapping for match %s: %v", matchID, err)
		}
	}

	// Create daily matches summary if date is provided
	var dailyMatchesList []*DailyMatches

	// Group matches by date
	matchesByDate := make(map[string][]string)
	for _, m := range matches {
		matchesByDate[m.Date] = append(matchesByDate[m.Date], m.ID)
	}

	// Create daily matches entries
	for date, matchIDs := range matchesByDate {
		// Insert matches
		for _, match := range matches {
			dbMatch := &models.Match{
				ID:         match.ID,
				HomeTeamID: match.HomeTeamID,
				AwayTeamID: match.AwayTeamID,
				LeagueID:   match.LeagueID,
				HomeScore:  match.HomeScore,
				AwayScore:  match.AwayScore,
				Status:     match.Status,
				KickOff:    match.KickOff,
				Date:       match.Date,
				Time:       match.Time,
			}
			if err := store.UpsertMatch(dbMatch); err != nil {
				log.Printf("Error upserting match %s: %v", match.ID, err)
			}
		}

		// Insert daily matches
		if err := store.UpsertDailyMatches(date, matchIDs); err != nil {
			log.Printf("Error upserting daily matches for date %s: %v", date, err)
		}

		dailyMatchesList = append(dailyMatchesList, &DailyMatches{
			Date:     date,
			MatchIDs: matchIDs,
		})
	}
	// If no matches but date provided, add empty entry
	if len(dailyMatchesList) == 0 && apiParams.Date != "" {
		dailyMatchesList = append(dailyMatchesList, &DailyMatches{
			Date:     apiParams.Date,
			MatchIDs: []string{},
		})
	}

	return matches, dailyMatchesList, nil
}
