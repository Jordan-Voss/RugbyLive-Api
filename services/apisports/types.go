package apisports

import (
	"encoding/json"
	"time"
)

type APIParams struct {
	LeagueID string
	Season   string
	Date     string
}

type DailyMatches struct {
	Date     string   `json:"date"`
	MatchIDs []string `json:"match_ids"`
}

type LeagueMappingResult struct {
	APILeague struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"api_league"`
	Matched     bool   `json:"matched"`
	MatchedName string `json:"matched_name"`
	InternalID  string `json:"internal_id"`
	Reason      string `json:"reason"`
}

type TeamSearchParams struct {
	CountryID string
	LeagueID  string
	Season    int
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

type teamChange struct {
	IsNew    bool                         `json:"is_new"`
	Changes  map[string]map[string]string `json:"changes"`
	TeamName string                       `json:"team_name"`
}

type failedTeam struct {
	Name        string      `json:"name"`
	CountryID   string      `json:"country_id"`
	CountryName string      `json:"country_name"`
	Reason      string      `json:"reason"`
	TeamData    interface{} `json:"team_data"`
}

type leagueChange struct {
	IsNew      bool                         `json:"is_new"`
	Changes    map[string]map[string]string `json:"changes"`
	LeagueName string                       `json:"league_name"`
}

type countryChange struct {
	IsNew       bool                         `json:"is_new"`
	Changes     map[string]map[string]string `json:"changes"`
	CountryName string                       `json:"country_name"`
}
