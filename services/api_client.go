package services

import (
	"log"
	"net/http"
	"time"
)

type APIClient struct {
	client *http.Client
}

func NewAPIClient() *APIClient {
	client := &APIClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Ensure bucket exists
	if err := client.createBucketIfNotExists(); err != nil {
		log.Printf("Error creating bucket: %v", err)
	}

	return client
}

// func (a *APIClient) createBucketIfNotExists() error {
// 	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
// 	url := fmt.Sprintf("%s/storage/v1/bucket", baseURL)

// 	payload := map[string]interface{}{
// 		"name":            "rugbylive-api",
// 		"public":          true,
// 		"file_size_limit": 5242880,
// 	}

// 	log.Printf("Creating bucket at URL: %s", url)
// 	log.Printf("Bucket name: %s", payload["name"])

// 	jsonData, err := json.Marshal(payload)
// 	if err != nil {
// 		return err
// 	}

// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return err
// 	}

// 	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
// 	req.Header.Set("Content-Type", "application/json")

// 	resp, err := a.client.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	// 200 means bucket exists, 201 means bucket was created
// 	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
// 		body, _ := io.ReadAll(resp.Body)
// 		return fmt.Errorf("failed to create bucket: %s - %s", resp.Status, string(body))
// 	}

// 	return nil
// }

// func (a *APIClient) FetchFromESPN() ([]models.Match, error) {
// 	// TODO: Implement ESPN fetching
// 	return nil, nil
// }

// func (a *APIClient) FetchFromUltimateRugby() ([]models.Match, error) {
// 	// TODO: Implement Ultimate Rugby fetching
// 	return nil, nil
// }

// func (a *APIClient) FetchFromAPISports() ([]models.Match, error) {
// 	// TODO: Implement API Sports fetching
// 	return nil, nil
// }

// func (a *APIClient) standardizeESPNData(resp models.ESPNResponse) []models.Match {
// 	// TODO: Implement ESPN data standardization
// 	return nil
// }

// func (a *APIClient) standardizeUltimateRugbyData(resp models.UltimateRugbyResponse) []models.Match {
// 	// TODO: Implement Ultimate Rugby data standardization
// 	return nil
// }

// func (a *APIClient) standardizeAPISportsData(resp models.APISportsTodaysMatchesResponse) []models.Match {
// 	// TODO: Implement API Sports data standardization
// 	return nil
// }

// Helper function to store mapping (implement this when adding database)
// func (a *APIClient) storeMappingForMatch(mapping models.APIMapping, matchKey string) {
// 	// This will be implemented when adding database support
// 	// For now, you could store in memory or log for verification
// 	log.Printf("Would store mapping: API=%s, ID=%s for match %s",
// 		mapping.APIName, mapping.APIID, matchKey)
// }

// func (a *APIClient) determineStatus(gameTime string) string {
// 	switch gameTime {
// 	case "":
// 		return "scheduled"
// 	case "FT":
// 		return "completed"
// 	default:
// 		return "live"
// 	}
// }

// func (a *APIClient) uploadToSupabase(imageData []byte, filename string) (string, error) {
// 	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
// 	url := fmt.Sprintf("%s/storage/v1/object/%s/%s",
// 		baseURL,
// 		"rugbylive-api",
// 		filename)

// 	req, err := http.NewRequest("PUT", url, bytes.NewReader(imageData))
// 	if err != nil {
// 		return "", err
// 	}

// 	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
// 	req.Header.Set("Content-Type", "image/svg+xml")
// 	req.Header.Set("Cache-Control", "max-age=3600")
// 	req.Header.Set("x-upsert", "true")

// 	// Add debug logging
// 	log.Printf("Uploading to URL: %s", url)
// 	log.Printf("Auth header: Bearer %s...", os.Getenv("SUPABASE_SERVICE_ROLE_KEY")[:10])

// 	resp, err := a.client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		body, _ := io.ReadAll(resp.Body)
// 		log.Printf("Supabase response: %s", string(body))
// 		return "", fmt.Errorf("failed to upload to Supabase: %s", resp.Status)
// 	}

// 	// Return the public URL
// 	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
// 		baseURL,
// 		"rugbylive-api",
// 		filename), nil
// }

// func (a *APIClient) FetchAndStoreLeagues(store *db.Store, updateImages bool) ([]leagueChange, error) {
// 	// TODO: Implement FetchAndStoreLeagues
// 	return nil, nil
// }

// func (a *APIClient) FetchAndStoreTeams(store *db.Store, updateImages bool, params TeamSearchParams) ([]teamChange, []failedTeam, error) {
// 	// TODO: Implement FetchAndStoreTeams
// 	return nil, nil, nil
// }

// func (a *APIClient) fetchTeamsForCountry(store *db.Store, url string, updateImages bool) ([]teamChange, []failedTeam, error) {
// 	// TODO: Implement fetchTeamsForCountry
// 	return nil, nil, nil
// }

// func (a *APIClient) TeamsResponse(resp TeamsResponse) []models.Team {
// 	// TODO: Implement TeamsResponse
// 	return nil
// }

// func (a *APIClient) failedTeam(resp failedTeam) failedTeam {
// 	// TODO: Implement failedTeam
// 	return failedTeam{}
// }

// func (a *APIClient) teamChange(resp teamChange) teamChange {
// 	// TODO: Implement teamChange
// 	return teamChange{}
// }

// func (a *APIClient) UpdateTeamImages(store *db.Store) error {
// 	// Get all teams from database
// 	teams, err := store.GetAllTeams()
// 	if err != nil {
// 		return fmt.Errorf("failed to get teams: %v", err)
// 	}

// 	for _, team := range teams {
// 		// Skip if no logo URL
// 		if team.LogoURL == "" {
// 			continue
// 		}

// 		log.Printf("Processing team %s", team.ID)

// 		// Download and store the image
// 		newLogoURL, err := a.downloadAndStoreImage(
// 			team.LogoURL,
// 			fmt.Sprintf("logos/teams/%s/%s/logo.png", team.Country.Code, strings.TrimPrefix(team.ID, team.Country.Code+"-")),
// 		)
// 		if err != nil {
// 			log.Printf("Error downloading logo for team %s: %v", team.ID, err)
// 			continue
// 		}

// 		team.LogoURL = newLogoURL
// 		team.LogoSource = "api_sports"

// 		if err := store.UpsertTeam(team); err != nil {
// 			log.Printf("Error updating team %s: %v", team.ID, err)
// 			continue
// 		}

// 		log.Printf("Successfully updated team %s with new logo URL", team.ID)
// 	}

// 	return nil
// }
