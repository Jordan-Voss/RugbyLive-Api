package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rugby-live-api/db"
	"strings"
)

type StorageFile struct {
	Name         string `json:"name"`
	ID           string `json:"id"`
	UpdatedAt    string `json:"updated_at"`
	CreatedAt    string `json:"created_at"`
	LastAccessed string `json:"last_accessed"`
	Metadata     struct {
		Size         int    `json:"size"`
		MimeType     string `json:"mimetype"`
		ETag         string `json:"eTag"`
		CacheControl string `json:"cacheControl"`
	} `json:"metadata"`
}

type ListResponse struct {
	Data []struct {
		Name string `json:"name"`
	} `json:"data"`
}

func (c *APIClient) downloadAndStoreImage(sourceURL string, destinationPath string) (string, error) {
	fmt.Printf("- Downloading image...\n")
	// Create request with headers
	req, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.rugbydatabase.co.nz/")

	// Download image
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status: %s", resp.Status)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %v", err)
	}

	// Upload to Supabase Storage
	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
	url := fmt.Sprintf("%s/storage/v1/object/rugbylive-api/%s", baseURL, destinationPath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(destinationPath)
	if dir != "." {
		createDirURL := fmt.Sprintf("%s/storage/v1/object/rugbylive-api/%s/", baseURL, dir)
		req, err := http.NewRequest("POST", createDirURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create directory request: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
		resp, err := c.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
		resp.Body.Close()
	}

	// Upload file
	req, err = http.NewRequest("POST", url, bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("Content-Type", http.DetectContentType(imageData))

	resp, err = c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload image: %s - %s", resp.Status, string(body))
	}

	// Return public URL
	return fmt.Sprintf("%s/storage/v1/object/public/rugbylive-api/%s", baseURL, destinationPath), nil
}

func (a *APIClient) UpdateTeamImages(store *db.Store) error {
	teams, err := store.GetAllTeams()
	if err != nil {
		return fmt.Errorf("failed to get teams: %v", err)
	}

	for _, team := range teams {
		log.Printf("Processing team %s", team.ID)

		// Get logo URL from rugbydb
		rugbydbURL := fmt.Sprintf("https://www.rugbydatabase.co.nz/images/teams/%s.png", strings.ToLower(team.ID))

		// Skip if rugbydb URL returns the generic team.webp
		resp, err := a.client.Head(rugbydbURL)
		if err != nil || resp.StatusCode != http.StatusOK || strings.Contains(resp.Header.Get("Content-Type"), "webp") {
			log.Printf("Skipping team %s - generic webp image or error", team.ID)
			continue
		}

		newLogoURL, err := a.downloadAndStoreImage(
			rugbydbURL,
			fmt.Sprintf("logos/teams/%s/%s/logo.png", team.Country.Code, strings.TrimPrefix(team.ID, team.Country.Code+"-")),
		)
		if err != nil {
			log.Printf("Error downloading logo for team %s: %v", team.ID, err)
			continue
		}

		team.LogoURL = newLogoURL
		team.LogoSource = "rugbydb"

		if err := store.UpsertTeam(team); err != nil {
			log.Printf("Error updating team %s: %v", team.ID, err)
			continue
		}

		log.Printf("Successfully updated team %s with new logo URL", team.ID)
	}

	return nil
}

func (c *APIClient) listStorageFiles(prefix string) ([]string, error) {
	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
	listURL := fmt.Sprintf("%s/storage/v1/bucket/list/rugbylive-api/%s", baseURL, strings.TrimPrefix(prefix, "/"))
	fmt.Printf("Listing files at URL: %s\n", listURL)

	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s\n", resp.Status)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Raw response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list files: %s", string(body))
	}

	// Reset the response body for subsequent reading
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	var listResponse ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		fmt.Printf("Decode error: %v\n", err)
		return nil, err
	}

	fmt.Printf("Found %d total files\n", len(listResponse.Data))
	fmt.Printf("All files in storage:\n")
	fileNames := make([]string, len(listResponse.Data))
	matchingFiles := 0
	for _, file := range listResponse.Data {
		fmt.Printf("- %s\n", file.Name)
		if strings.HasPrefix(file.Name, prefix) {
			fileNames[matchingFiles] = file.Name
			matchingFiles++
			fmt.Printf("Matching file %d: %s\n", matchingFiles, file.Name)
		}
	}
	fileNames = fileNames[:matchingFiles]
	fmt.Printf("Found %d files matching prefix %s\n", matchingFiles, prefix)

	return fileNames, nil
}

func (c *APIClient) deleteStorageFile(path string) error {
	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
	deleteURL := fmt.Sprintf("%s/storage/v1/object/rugbylive-api/%s", baseURL, path)

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete file: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *APIClient) MigrateStoragePaths(store *db.Store) error {
	fmt.Println("Starting storage migration...")

	rows, err := store.DB.Query(`
        SELECT old_code, new_code
        FROM country_code_mapping
    `)
	if err != nil {
		return fmt.Errorf("failed to get country mappings: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var oldCode, newCode string
		if err := rows.Scan(&oldCode, &newCode); err != nil {
			return fmt.Errorf("failed to scan country mapping: %v", err)
		}
		fmt.Printf("Processing country code: %s -> %s\n", oldCode, newCode)

		// List all files in the old country code folder
		oldPrefix := fmt.Sprintf("logos/teams/%s/", oldCode)
		files, err := c.listStorageFiles(oldPrefix)
		if err != nil {
			fmt.Printf("Error listing files for %s: %v\n", oldCode, err)
			continue
		}

		// Move each file to the new country code folder
		for _, file := range files {
			oldPath := file // file already contains full path
			newPath := strings.Replace(oldPath,
				oldCode,
				newCode, 1)

			oldURL := fmt.Sprintf("%s/storage/v1/object/public/rugbylive-api/%s",
				strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3"),
				oldPath)
			newURL := fmt.Sprintf("%s/storage/v1/object/public/rugbylive-api/%s",
				strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3"),
				newPath)

			if err := c.moveStorageFile(oldURL, newURL); err != nil {
				fmt.Printf("Error moving file %s: %v\n", file, err)
				continue
			}
			fmt.Printf("Moved file %s to %s\n", oldPath, newPath)
		}
	}

	return nil
}

func (c *APIClient) moveStorageFile(oldURL, newURL string) error {
	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
	moveURL := fmt.Sprintf("%s/storage/v1/object/move", baseURL)

	// Extract bucket paths
	oldPath := strings.TrimPrefix(oldURL, fmt.Sprintf("%s/storage/v1/object/public/rugbylive-api/", baseURL))
	newPath := strings.TrimPrefix(newURL, fmt.Sprintf("%s/storage/v1/object/public/rugbylive-api/", baseURL))

	payload := map[string]interface{}{
		"bucketId":       "rugbylive-api",
		"sourceKey":      oldPath,
		"destinationKey": newPath,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", moveURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to move file: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *APIClient) createBucketIfNotExists() error {
	baseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/storage/v1/s3")
	url := fmt.Sprintf("%s/storage/v1/bucket", baseURL)

	payload := map[string]interface{}{
		"name":            "rugbylive-api",
		"public":          true,
		"file_size_limit": 5242880,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return nil
	}

	return nil
}
