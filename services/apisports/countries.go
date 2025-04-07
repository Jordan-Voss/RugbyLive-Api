package apisports

import (
	"encoding/json"
	"log"
	"rugby-live-api/db"
	"rugby-live-api/models"
	"strings"
)

func (c *Client) FetchAndStoreCountries(store *db.Store, updateFlags bool) ([]countryChange, error) {
	url := "https://v1.rugby.api-sports.io/countries"
	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Response []struct {
			Name string `json:"name"`
			Code string `json:"code"`
			Flag string `json:"flag"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	var changes []countryChange
	for _, country := range apiResp.Response {
		countryID := strings.ToUpper(country.Code)
		dbCountry := &models.Country{
			Code: countryID,
			Name: country.Name,
			Flag: country.Flag,
		}

		change := countryChange{
			IsNew:       true,
			Changes:     make(map[string]map[string]string),
			CountryName: country.Name,
		}

		existing, err := store.GetCountryByCode(countryID)
		if err == nil && existing != nil {
			change.IsNew = false
			if existing.Name != dbCountry.Name {
				change.Changes["name"] = map[string]string{"old": existing.Name, "new": dbCountry.Name}
			}
			if updateFlags && existing.Flag != dbCountry.Flag {
				change.Changes["flag"] = map[string]string{"old": existing.Flag, "new": dbCountry.Flag}
			}
		}

		if err := store.UpsertCountry(dbCountry); err != nil {
			log.Printf("Error upserting country %s: %v", country.Name, err)
			continue
		}

		if change.IsNew {
			log.Printf("Added new country: %s", country.Name)
		} else if len(change.Changes) > 0 {
			log.Printf("Updated country: %s with changes: %v", country.Name, change.Changes)
		}

		changes = append(changes, change)
	}

	return changes, nil
}
