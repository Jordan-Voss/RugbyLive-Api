package rapidapi

import (
	"encoding/json"
)

func (c *Client) GetCompetitions() (map[string]interface{}, error) {
	url := "https://rugby-live-data.p.rapidapi.com/competitions"

	req, err := c.createRequest("GET", url)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
