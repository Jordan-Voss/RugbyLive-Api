package services

import (
	"encoding/json"
	"net/http"
	"os"
)

type RugbyLiveAPI struct {
	client *http.Client
}

func NewRugbyLiveAPI() *RugbyLiveAPI {
	return &RugbyLiveAPI{
		client: &http.Client{},
	}
}

func (a *RugbyLiveAPI) GetCompetitions() (map[string]interface{}, error) {
	url := "https://rugby-live-data.p.rapidapi.com/competitions"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-rapidapi-key", os.Getenv("RAPID_API_KEY"))
	req.Header.Add("x-rapidapi-host", "rugby-live-data.p.rapidapi.com")

	resp, err := a.client.Do(req)
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
