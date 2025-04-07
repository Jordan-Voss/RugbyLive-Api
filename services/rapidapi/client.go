package rapidapi

import (
	"net/http"
	"os"
)

type Client struct {
	client *http.Client
	apiKey string
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
		apiKey: os.Getenv("RAPID_API_KEY"),
	}
}

func (c *Client) createRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-rapidapi-key", c.apiKey)
	req.Header.Add("x-rapidapi-host", "rugby-live-data.p.rapidapi.com")

	return req, nil
}
