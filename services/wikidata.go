package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func (a *APIClient) makeRequestWithRetries(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := a.client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return nil, fmt.Errorf("after %d retries: %v", maxRetries, lastErr)
}

func (a *APIClient) GetWikidataTeams() ([]WikidataTeam, error) {
	// SPARQL query to get all rugby teams
	query := "SELECT ?team ?teamLabel (REPLACE(STR(?team), '^.*/([QqPp][0-9]+)$', '$1') AS ?wikidataID) WHERE { ?team wdt:P31 wd:Q14645593. SERVICE wikibase:label {bd:serviceParam wikibase:language \"en\".}}"

	// URL encode the query
	url := fmt.Sprintf("https://query.wikidata.org/sparql?format=json&query=%s",
		url.QueryEscape(query))

	fmt.Printf("Requesting URL: %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RugbyLiveAPI/1.0")

	resp, err := a.makeRequestWithRetries(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}
	fmt.Printf("Response: %s\n", string(body))

	var sparqlResp WikidataSPARQLResponse
	if err := json.Unmarshal(body, &sparqlResp); err != nil {
		return nil, err
	}

	var teams []WikidataTeam
	for _, result := range sparqlResp.Results.Bindings {
		teamID := result.WikidataID.Value
		team, err := a.getWikidataTeam(teamID)
		if err != nil {
			continue
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func (a *APIClient) getWikidataTeam(teamID string) (WikidataTeam, error) {
	url := fmt.Sprintf("https://www.wikidata.org/wiki/Special:EntityData/%s.json", teamID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return WikidataTeam{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RugbyLiveAPI/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return WikidataTeam{}, err
	}
	defer resp.Body.Close()

	var data WikidataResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		return WikidataTeam{}, err
	}

	entity := data.Entities[teamID]
	team := WikidataTeam{
		ID:   teamID,
		Name: entity.Labels["en"].Value,
	}

	// Extract other fields from claims
	if claims, ok := entity.Claims["P1448"]; ok {
		if val, ok := claims[0].MainSnak.DataValue.Value.(map[string]interface{}); ok {
			if text, ok := val["text"].(string); ok {
				team.Nickname = text
			}
		}
	}
	if claims, ok := entity.Claims["P154"]; ok {
		if val, ok := claims[0].MainSnak.DataValue.Value.(string); ok {
			team.LogoURL = "https://commons.wikimedia.org/wiki/File:" + val
		}
	}
	if claims, ok := entity.Claims["P571"]; ok {
		team.Founded = claims[0].MainSnak.DataValue.Value.(map[string]interface{})["time"].(string)[:4]
	}
	if claims, ok := entity.Claims["P115"]; ok {
		if val, ok := claims[0].MainSnak.DataValue.Value.(map[string]interface{}); ok {
			if id, ok := val["id"].(string); ok {
				team.Stadium = id
			}
		}
	}
	if claims, ok := entity.Claims["P286"]; ok {
		if val, ok := claims[0].MainSnak.DataValue.Value.(map[string]interface{}); ok {
			if id, ok := val["id"].(string); ok {
				team.Coach = id
			}
		}
	}

	return team, nil
}

func (a *APIClient) SearchWikidataTeam(name string) (*WikidataTeam, error) {
	// SPARQL query to search for rugby teams by name
	query := fmt.Sprintf(`
		SELECT DISTINCT ?team ?teamLabel 
		       ?nickname ?logo ?country ?countryLabel
		       ?stadium ?stadiumLabel ?founded ?coach ?coachLabel
		       ?league ?leagueLabel ?competition ?competitionLabel
		       ?player ?playerLabel ?caps ?website
		       ?facebook ?twitter ?instagram
		       ?kitManufacturer ?kitManufacturerLabel
		       ?sponsor ?sponsorLabel ?fifaCode
		       (REPLACE(STR(?team), '^.*/([QqPp][0-9]+)$', '$1') AS ?wikidataID)
		WHERE { 
			{
				?team wdt:P31/wdt:P279* wd:Q14645593.
			}
			{
				?team rdfs:label ?label.
				FILTER(CONTAINS(LCASE(?label), LCASE("%s")))
			} UNION {
				?team skos:altLabel ?altLabel.
				FILTER(CONTAINS(LCASE(?altLabel), LCASE("%s")))
			}
			OPTIONAL { ?team wdt:P1448 ?nickname }
			OPTIONAL { ?team wdt:P154 ?logo }
			OPTIONAL { ?team wdt:P17 ?country }
			OPTIONAL { ?team wdt:P115 ?stadium }
			OPTIONAL { ?team wdt:P571 ?founded }
			OPTIONAL { ?team wdt:P286 ?coach }
			OPTIONAL { ?team wdt:P3450 ?league }
			OPTIONAL { ?team wdt:P1346 ?competition }
			OPTIONAL { ?team wdt:P54 ?player }
			OPTIONAL { ?team wdt:P1350 ?caps }
			OPTIONAL { ?team wdt:P856 ?website }
			OPTIONAL { ?team wdt:P2013 ?facebook }
			OPTIONAL { ?team wdt:P2002 ?twitter }
			OPTIONAL { ?team wdt:P2003 ?instagram }
			OPTIONAL { ?team wdt:P176 ?kitManufacturer }
			OPTIONAL { ?team wdt:P859 ?sponsor }
			OPTIONAL { ?team wdt:P2121 ?fifaCode }
			SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
		} LIMIT 1`, name, name)

	url := fmt.Sprintf("https://query.wikidata.org/sparql?format=json&query=%s",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RugbyLiveAPI/1.0")

	resp, err := a.makeRequestWithRetries(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, body, "", "  ")
	fmt.Printf("Wikidata SPARQL Response:\n%s\n", prettyJSON.String())

	var sparqlResp WikidataSPARQLResponse
	if err := json.Unmarshal(body, &sparqlResp); err != nil {
		return nil, err
	}

	for _, result := range sparqlResp.Results.Bindings {
		teamID := result.WikidataID.Value
		return &WikidataTeam{
			ID:       teamID,
			Name:     result.TeamLabel.Value,
			Nickname: result.Nickname.Value,
			LogoURL:  result.Logo.Value,
			Founded:  result.Founded.Value,
			Stadium:  result.StadiumLabel.Value,
			Coach:    result.CoachLabel.Value,
		}, nil
	}

	return nil, fmt.Errorf("team not found")
}
