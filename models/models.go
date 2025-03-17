package models

import (
	"encoding/json"
	"time"
)

type Match struct {
	ID          string    `json:"id"` // e.g., "2024-03-08-SARACENS-EXETER"
	HomeTeamID  string    `json:"home_team_id"`
	AwayTeamID  string    `json:"away_team_id"`
	LeagueID    string    `json:"league_id"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	Status      string    `json:"status"`
	KickOff     time.Time `json:"kick_off"`
	Week        string    `json:"week"`
	Season      int       `json:"season"`
	APISportsID int       `json:"api_sports_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// New fields
	Venue             *string         `json:"venue,omitempty"`
	Referee           *string         `json:"referee,omitempty"`
	Attendance        *int            `json:"attendance,omitempty"`
	WeatherConditions *string         `json:"weather_conditions,omitempty"`
	HeadToHead        json.RawMessage `json:"head_to_head,omitempty"`
	Lineups           json.RawMessage `json:"lineups,omitempty"`
	LiveStats         json.RawMessage `json:"live_stats,omitempty"`
	UniqueKey         string          `json:"unique_key"`

	// Joined fields
	HomeTeam *Team   `json:"home_team,omitempty"`
	AwayTeam *Team   `json:"away_team,omitempty"`
	League   *League `json:"league,omitempty"`
}

type Country struct {
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Flag       string    `json:"flag"`
	FlagSource string    `json:"flag_source"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Team struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	LogoURL    string        `json:"logo_url"`
	LogoSource string        `json:"logo_source"`
	Country    Country       `json:"country"`
	Stadiums   []TeamStadium `json:"stadiums,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	AltNames   []string      `json:"alternate_names"`
}

type APIMapping struct {
	ID         int       `json:"id"`
	EntityID   string    `json:"entity_id"`   // Our internal ID
	APIName    string    `json:"api_name"`    // e.g., "api_sports"
	APIID      string    `json:"api_id"`      // External API's ID
	EntityType string    `json:"entity_type"` // "match", "team", "league"
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type MatchAPIMapping struct {
	ID         int       `json:"id"`
	MatchID    string    `json:"match_id"`
	APIName    string    `json:"api_name"`
	APIMatchID string    `json:"api_match_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type APISportsTodaysMatchesResponse struct {
	Get        string `json:"get"`
	Parameters struct {
		Date string `json:"date"`
	} `json:"parameters"`
	Results  int `json:"results"`
	Response []struct {
		ID        int    `json:"id"`
		Date      string `json:"date"`
		Time      string `json:"time"`
		Timestamp int64  `json:"timestamp"`
		Timezone  string `json:"timezone"`
		Week      string `json:"week"`
		Status    struct {
			Long  string `json:"long"`
			Short string `json:"short"`
		} `json:"status"`
		Country struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
			Flag string `json:"flag"`
		} `json:"country"`
		League struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Type   string `json:"type"`
			Logo   string `json:"logo"`
			Season int    `json:"season"`
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

type Stadium struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Capacity  int       `json:"capacity"`
	Location  string    `json:"location"`
	Country   Country   `json:"country"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamStadium struct {
	Stadium   Stadium   `json:"stadium"`
	IsPrimary bool      `json:"is_primary"`
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   time.Time `json:"end_date,omitempty"`
}
