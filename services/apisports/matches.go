package apisports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"strconv"
	"strings"
	"time"
)

func (c *Client) GetMatchesByLeague(leagueID string, date string, season string, params APIParams, store *db.Store) ([]Match, []*DailyMatches, error) {
	// Get season from database
	dbSeason, err := store.GetSeasonByLeagueAndYear(leagueID, season)
	if err != nil {
		return nil, nil, fmt.Errorf("season not found: %v", err)
	}

	// Build URL with parameters
	urlParams := make([]string, 0)
	urlParams = append(urlParams, fmt.Sprintf("league=%s", params.LeagueID))

	if params.Season != "" {
		urlParams = append(urlParams, fmt.Sprintf("season=%s", params.Season))
	}
	if params.Date != "" {
		urlParams = append(urlParams, fmt.Sprintf("date=%s", params.Date))
	}

	url := fmt.Sprintf("https://v1.rugby.api-sports.io/games?%s", strings.Join(urlParams, "&"))

	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

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

	var matches []Match
	for _, m := range apiResp.Response {
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

	var dailyMatchesList []*DailyMatches
	matchesByDate := make(map[string][]string)
	for _, m := range matches {
		matchesByDate[m.Date] = append(matchesByDate[m.Date], m.ID)
	}

	for date, matchIDs := range matchesByDate {
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

			if err := store.UpsertDailyMatches(date, matchIDs); err != nil {
				log.Printf("Error upserting daily matches for date %s: %v", date, err)
			}

			dailyMatchesList = append(dailyMatchesList, &DailyMatches{
				Date:     date,
				MatchIDs: matchIDs,
			})
		}
	}

	if len(dailyMatchesList) == 0 && params.Date != "" {
		dailyMatchesList = append(dailyMatchesList, &DailyMatches{
			Date:     params.Date,
			MatchIDs: []string{},
		})
	}

	return matches, dailyMatchesList, nil
}

func (c *Client) FetchFromAPISports() ([]models.Match, error) {
	today := time.Now().Format("2006-01-02")
	url := fmt.Sprintf("https://v1.rugby.api-sports.io/games?date=%s", today)

	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp models.APISportsTodaysMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return c.standardizeAPISportsData(apiResp), nil
}

func (c *Client) standardizeAPISportsData(resp models.APISportsTodaysMatchesResponse) []models.Match {
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
	}
	return matches
}
