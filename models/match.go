package models

import "time"

type Match struct {
	ID          string    `json:"id"`
	HomeTeam    *Team     `json:"home_team"`
	AwayTeam    *Team     `json:"away_team"`
	League      *League   `json:"league"`
	HomeTeamID  string    `json:"home_team_id"`
	AwayTeamID  string    `json:"away_team_id"`
	LeagueID    string    `json:"league_id"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	Status      string    `json:"status"`
	KickOff     time.Time `json:"kick_off"`
	Date        string    `json:"date"`
	Time        string    `json:"time"`
	Week        string    `json:"week"`
	Season      int       `json:"season"`
	APISportsID int       `json:"api_sports_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
