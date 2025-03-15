package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"strings"
	"time"
)

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
			Name: game.League.Name,
			Type: game.League.Type,
			Logo: game.League.Logo,
			Country: models.Country{
				Code: game.Country.Code,
				Name: game.Country.Name,
				Flag: game.Country.Flag,
			},
			Seasons: []models.Season{{
				Year:    game.League.Season,
				Current: true,
				Start:   time.Now(),
				End:     time.Now().AddDate(0, 6, 0),
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
			if existing == nil || (updateImages && existing.LogoSource == "api_sports" && existing.Logo != l.Logo) {
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
						change.Changes["logo"] = map[string]string{"old": existing.Logo, "new": logoURL}
					}
				}
			}

			// Create league with all seasons
			var seasons []models.Season
			for _, s := range l.Seasons {
				start, _ := time.Parse("2006-01-02", s.Start)
				end, _ := time.Parse("2006-01-02", s.End)
				seasons = append(seasons, models.Season{
					Year:    s.Year,
					Current: s.Current,
					Start:   start,
					End:     end,
				})
			}

			league := &models.League{
				ID:         change.ID,
				Name:       l.Name,
				Type:       l.Type,
				Logo:       logoURL,
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
				if existing.Type != league.Type {
					change.Changes["type"] = map[string]string{"old": existing.Type, "new": league.Type}
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
