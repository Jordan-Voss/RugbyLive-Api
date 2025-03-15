package services

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly"
)

func (a *APIClient) ScrapeESPNTeam(teamURL string) (*ESPNTeamInfo, error) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	teamInfo := &ESPNTeamInfo{
		RawData: make(map[string]interface{}),
	}

	// Get team logo
	c.OnHTML(".TeamHeader__Image img", func(e *colly.HTMLElement) {
		teamInfo.LogoURL = e.Attr("src")
	})

	// Get team name
	c.OnHTML(".ClubhouseHeader__Name", func(e *colly.HTMLElement) {
		teamInfo.Name = e.Text
	})

	// Get founded year
	c.OnHTML(".ClubhouseHeader__Meta span", func(e *colly.HTMLElement) {
		text := e.Text
		if strings.HasPrefix(text, "Est.") {
			teamInfo.FoundedYear = strings.TrimPrefix(text, "Est. ")
		}
	})

	// Get stadium and head coach
	c.OnHTML(".stat-headline", func(e *colly.HTMLElement) {
		label := strings.TrimSpace(e.Text)
		value := strings.TrimSpace(e.DOM.Next().Text())

		switch label {
		case "Stadium":
			teamInfo.Stadium = value
		case "Head Coach":
			teamInfo.HeadCoach = value
		default:
			teamInfo.RawData[label] = value
		}
	})

	// Get players
	c.OnHTML(".player__row", func(e *colly.HTMLElement) {
		player := ESPNPlayer{
			Name:     e.ChildText(".player__name"),
			Position: e.ChildText(".player__position"),
			Caps:     e.ChildText(".player__caps"),
			Club:     e.ChildText(".player__club"),
		}
		teamInfo.Players = append(teamInfo.Players, player)
	})

	err := c.Visit(teamURL)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape ESPN team page: %v", err)
	}

	return teamInfo, nil
}

func (a *APIClient) ScrapeESPNLeagues() ([]ESPNLeague, error) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	var leagues []ESPNLeague
	c.OnHTML(".dropdown-menu.med li a", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		leagueID := strings.TrimPrefix(
			strings.TrimSuffix(
				strings.Split(href, "league/")[1],
				"\"",
			),
			"\"",
		)

		leagues = append(leagues, ESPNLeague{
			ID:   leagueID,
			Name: e.Text,
			URL:  "https://www.espn.com" + href,
		})
	})

	err := c.Visit("https://www.espn.com/rugby/standings")
	if err != nil {
		return nil, fmt.Errorf("failed to scrape ESPN leagues: %v", err)
	}

	return leagues, nil
}
