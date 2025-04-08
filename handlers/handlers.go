package handlers

import (
	"fmt"
	"log"
	"net/http"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"rugby-live-api/services"
	"rugby-live-api/services/rapidapi"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	apiClient *services.APIClient
	store     *db.Store
	rapidAPI  *rapidapi.Client
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{
		apiClient: services.NewAPIClient(),
		store:     store,
		rapidAPI:  rapidapi.NewClient(),
	}
}

func (h *Handler) GetMatches(c *gin.Context) {
	matches, err := h.apiClient.FetchFromAPISports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches: " + err.Error()})
		return
	}

	// Store each match and its related data
	for _, match := range matches {
		// Insert country first
		if err := h.store.UpsertCountry(&match.League.Country); err != nil {
			log.Printf("Error upserting country: %v", err)
			continue
		}

		if err := h.store.UpsertLeague(match.League); err != nil {
			log.Printf("Error upserting league: %v", err)
			continue
		}
		if err := h.store.UpsertTeam(match.HomeTeam); err != nil {
			log.Printf("Error upserting home team: %v", err)
			continue
		}
		if err := h.store.UpsertTeam(match.AwayTeam); err != nil {
			log.Printf("Error upserting away team: %v", err)
			continue
		}

		// Set IDs for the match
		match.LeagueID = match.League.ID
		match.HomeTeamID = match.HomeTeam.ID
		match.AwayTeamID = match.AwayTeam.ID

		if err := h.store.UpsertMatch(&match); err != nil {
			log.Printf("Error upserting match: %v", err)
			continue
		}

		// Create API mapping for the match
		apiMapping := &models.MatchAPIMapping{
			MatchID:    match.ID,
			APIName:    "api_sports",
			APIMatchID: fmt.Sprintf("%d", match.APISportsID),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := h.store.UpsertMatchAPIMapping(apiMapping); err != nil {
			log.Printf("Error upserting match API mapping: %v", err)
			continue
		}
	}

	c.JSON(http.StatusOK, matches)
}

func (h *Handler) GetLiveMatches(c *gin.Context) {
	// espnData, err := h.apiClient.FetchFromESPN()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ESPN data: " + err.Error()})
	// 	return
	// }

	// urData, err := h.apiClient.FetchFromUltimateRugby()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Ultimate Rugby data: " + err.Error()})
	// 	return
	// }

	// allMatches := append(espnData, urData...)
	// c.JSON(http.StatusOK, allMatches)
}

func (h *Handler) GetUpcomingMatches(c *gin.Context) {
	// For now, returns same data - you can filter by status later
	// espnData, err := h.apiClient.FetchFromESPN()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ESPN data: " + err.Error()})
	// 	return
	// }

	// urData, err := h.apiClient.FetchFromUltimateRugby()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Ultimate Rugby data: " + err.Error()})
	// 	return
	// }

	// allMatches := append(espnData, urData...)
	// c.JSON(http.StatusOK, allMatches)
}

func (h *Handler) GetCountries(c *gin.Context) {
	countries, err := h.store.GetCountries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch countries: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, countries)
}

func (h *Handler) RefreshCountries(c *gin.Context) {
	// Default to false if not specified
	updateFlags := false
	if c.Query("update_flags") != "" {
		updateFlags = c.Query("update_flags") == "true"
	}

	changes, err := h.apiClient.FetchAndStoreCountries(h.store, updateFlags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh countries: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Countries refresh completed",
		"changes":       changes,
		"total_changes": len(changes),
	})
}

func (h *Handler) RefreshLeagues(c *gin.Context) {
	log.Println("Refreshing leagues update images: ", c.Query("update_images"))
	updateImages := c.Query("update_images") == "true"
	changes, err := h.apiClient.FetchAndStoreLeagues(h.store, updateImages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh leagues: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":       "Leagues refresh completed",
		"changes":       changes,
		"total_changes": len(changes),
	})
}

func (h *Handler) RefreshTeams(c *gin.Context) {
	updateImages := c.Query("update_images") == "true"
	params := services.TeamSearchParams{
		CountryID: c.Query("country"),
		LeagueID:  c.Query("league"),
	}
	if season := c.Query("season"); season != "" {
		if s, err := strconv.Atoi(season); err == nil {
			params.Season = s
		}
	}

	// Validate league parameters
	if params.LeagueID != "" && params.Season == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "season parameter is required when searching by league"})
		return
	}

	changes, failedTeams, err := h.apiClient.FetchAndStoreTeams(h.store, updateImages, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh teams: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Teams refresh completed",
		"changes":        changes,
		"total_changes":  len(changes),
		"failed_teams":   failedTeams,
		"total_failures": len(failedTeams),
	})
}

func (h *Handler) UpdateTeamImages(c *gin.Context) {
	if err := h.apiClient.UpdateTeamImages(h.store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team images: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Team images updated successfully"})
}

func (h *Handler) GetESPNLeagues(c *gin.Context) {
	leagues, err := h.apiClient.ScrapeESPNLeagues()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scrape leagues: %v", err)})
		return
	}
	c.JSON(http.StatusOK, leagues)
}

func (h *Handler) GetWikidataTeams(c *gin.Context) {
	teams, err := h.apiClient.GetWikidataTeams()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch teams: %v", err)})
		return
	}
	c.JSON(http.StatusOK, teams)
}

func (h *Handler) SearchWikidataTeams(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter is required"})
		return
	}

	team, err := h.apiClient.SearchWikidataTeam(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to search teams: %v", err)})
		return
	}
	c.JSON(http.StatusOK, team)
}

func (h *Handler) GetRugbyDBTeams(c *gin.Context) {
	var priorityTeams []string
	var countryFilter string

	// Handle both GET and POST methods
	if c.Request.Method == "POST" {
		var req services.TeamCreateRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
			return
		}
		priorityTeams = req.Names
		countryFilter = req.Country
	} else {
		priorityTeams = c.QueryArray("teams")
		countryFilter = c.Query("country")
	}

	teams, err := h.apiClient.GetRugbyDBTeams(h.store, priorityTeams, countryFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch teams: %v", err)})
		return
	}
	c.JSON(http.StatusOK, teams)
}

func (h *Handler) CreateRugbyDBTeams(c *gin.Context) {
	var req services.TeamCreateRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	teams, err := h.apiClient.GetRugbyDBTeams(h.store, req.Names, req.Country)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process teams: %v", err)})
		return
	}

	c.JSON(http.StatusOK, teams)
}

func (h *Handler) MapAPISportsLeagues(c *gin.Context) {
	results, err := h.apiClient.MapAPISportsLeagues(h.store)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to map leagues: %v", err)})
		return
	}

	// Group results by match status
	matched := make([]services.LeagueMappingResult, 0)
	unmatched := make([]services.LeagueMappingResult, 0)

	for _, result := range results {
		if result.Matched {
			matched = append(matched, result)
		} else {
			unmatched = append(unmatched, result)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matched":   matched,
		"unmatched": unmatched,
		"stats": gin.H{
			"total":      len(results),
			"matched":    len(matched),
			"unmatched":  len(unmatched),
			"match_rate": float64(len(matched)) / float64(len(results)),
		},
	})
}

func (h *Handler) GetLeagueIDsByYear(c *gin.Context) {
	year := c.Param("year")
	if year == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year parameter is required"})
		return
	}

	mappings, err := h.apiClient.GetLeagueIDsByYear(year, h.store)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get league IDs: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"year":    year,
		"leagues": mappings,
	})
}

func (h *Handler) GetMatchesByLeague(c *gin.Context) {
	leagueID := c.Query("league_id")
	if leagueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "league_id parameter is required"})
		return
	}

	date := c.Query("date")
	season := c.Query("season")
	apiLeagueID := c.Query("api_league_id")
	apiSeason := c.Query("api_season")
	apiDate := c.Query("api_date")

	dateParam := apiDate
	if dateParam == "" {
		dateParam = date
	}

	matches, dailyMatchesList, err := h.apiClient.GetMatchesByLeague(leagueID, date, season,
		services.APIParams{
			LeagueID: apiLeagueID,
			Season:   apiSeason,
			Date:     dateParam,
		},
		h.store)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get matches: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"league_id": leagueID,
		"date":      date,
		"season":    season,
		"api_params": gin.H{
			"league_id": apiLeagueID,
			"season":    apiSeason,
			"date":      apiDate,
		},
		"matches":       matches,
		"daily_matches": dailyMatchesList,
	})
}

func (h *Handler) GetRugbyLiveCompetitions(c *gin.Context) {
	mappings, err := h.rapidAPI.MapCompetitionsToLeagues(h.store)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch competitions: %v", err)})
		return
	}

	// Group results by match status
	matched := make([]rapidapi.CompetitionMapping, 0)
	unmatched := make([]rapidapi.CompetitionMapping, 0)

	for _, mapping := range mappings {
		if mapping.Matched {
			matched = append(matched, mapping)
		} else {
			unmatched = append(unmatched, mapping)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matched":   matched,
		"unmatched": unmatched,
		"stats": gin.H{
			"total":      len(mappings),
			"matched":    len(matched),
			"unmatched":  len(unmatched),
			"match_rate": float64(len(matched)) / float64(len(mappings)),
		},
	})
}
