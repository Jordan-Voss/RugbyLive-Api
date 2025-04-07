package wikidata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"rugby-live-api/models"
	"strings"
)

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
	}
}

func (c *Client) GetTeams() ([]models.Team, error) {
	query := `
	SELECT ?team ?teamLabel ?country ?countryLabel WHERE {
	  ?team wdt:P31 wd:Q476028.
	  ?team wdt:P17 ?country.
	  SERVICE wikibase:label { bd:serviceParam wikibase:language "[AUTO_LANGUAGE],en". }
	}
	LIMIT 100
	`

	url := fmt.Sprintf("https://query.wikidata.org/sparql?query=%s&format=json",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results struct {
			Bindings []struct {
				Team struct {
					Value string `json:"value"`
				} `json:"team"`
				TeamLabel struct {
					Value string `json:"value"`
				} `json:"teamLabel"`
				Country struct {
					Value string `json:"value"`
				} `json:"country"`
				CountryLabel struct {
					Value string `json:"value"`
				} `json:"countryLabel"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var teams []models.Team
	for _, binding := range result.Results.Bindings {
		countryCode := strings.ToUpper(strings.TrimPrefix(binding.Country.Value, "http://www.wikidata.org/entity/"))
		teamID := fmt.Sprintf("%s-%s", countryCode,
			strings.ToUpper(strings.ReplaceAll(binding.TeamLabel.Value, " ", "")))

		team := models.Team{
			ID:   teamID,
			Name: binding.TeamLabel.Value,
			Country: models.Country{
				Code: countryCode,
				Name: binding.CountryLabel.Value,
			},
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func (c *Client) SearchTeam(name string) (*models.Team, error) {
	query := fmt.Sprintf(`
	SELECT ?team ?teamLabel ?country ?countryLabel WHERE {
	  ?team wdt:P31 wd:Q476028;
			rdfs:label ?label;
			wdt:P17 ?country.
	  FILTER(CONTAINS(LCASE(?label), LCASE("%s"))).
	  SERVICE wikibase:label { bd:serviceParam wikibase:language "[AUTO_LANGUAGE],en". }
	}
	LIMIT 1
	`, name)

	url := fmt.Sprintf("https://query.wikidata.org/sparql?query=%s&format=json",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results struct {
			Bindings []struct {
				Team struct {
					Value string `json:"value"`
				} `json:"team"`
				TeamLabel struct {
					Value string `json:"value"`
				} `json:"teamLabel"`
				Country struct {
					Value string `json:"value"`
				} `json:"country"`
				CountryLabel struct {
					Value string `json:"value"`
				} `json:"countryLabel"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results.Bindings) == 0 {
		return nil, fmt.Errorf("no team found matching: %s", name)
	}

	binding := result.Results.Bindings[0]
	countryCode := strings.ToUpper(strings.TrimPrefix(binding.Country.Value, "http://www.wikidata.org/entity/"))
	teamID := fmt.Sprintf("%s-%s", countryCode,
		strings.ToUpper(strings.ReplaceAll(binding.TeamLabel.Value, " ", "")))

	team := &models.Team{
		ID:   teamID,
		Name: binding.TeamLabel.Value,
		Country: models.Country{
			Code: countryCode,
			Name: binding.CountryLabel.Value,
		},
	}

	return team, nil
}
