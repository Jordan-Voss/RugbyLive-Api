package services

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"rugby-live-api/db"
	"rugby-live-api/models"

	"github.com/PuerkitoBio/goquery"
)

var teamSuffixes = []string{
	" A",
	" B",
	" C",
	" XV",
	" Women",
	" W",
	" (W)",
	" Women (W)",
}

var oppositeWords = map[string]string{
	"northern": "southern",
	"southern": "northern",
	"eastern":  "western",
	"western":  "eastern",
	"north":    "south",
	"south":    "north",
	"east":     "west",
	"west":     "east",
}

// equivalentSuffixes groups suffixes that should be treated as the same
var equivalentSuffixes = map[string][]string{
	" W":         {" Women", " (W)", " Women (W)"},
	" Women":     {" W", " (W)", " Women (W)"},
	" (W)":       {" W", " Women", " Women (W)"},
	" Women (W)": {" W", " Women", " (W)"},
}

func hasDifferentSuffix(name1, name2 string) bool {
	// Special case: if one name ends with " (W)" and the other contains "Women"
	if (strings.HasSuffix(name1, " (W)") && strings.Contains(name2, "Women")) ||
		(strings.HasSuffix(name2, " (W)") && strings.Contains(name1, "Women")) {
		return false
	}

	name1Suffix := ""
	name2Suffix := ""

	for _, suffix := range teamSuffixes {
		if strings.HasSuffix(name1, suffix) {
			name1Suffix = suffix
		}
		if strings.HasSuffix(name2, suffix) {
			name2Suffix = suffix
		}
	}

	// Check if suffixes are equivalent
	if name1Suffix != "" && name2Suffix != "" {
		if name1Suffix == name2Suffix {
			return false
		}
		if equivalents, exists := equivalentSuffixes[name1Suffix]; exists {
			for _, equiv := range equivalents {
				if name2Suffix == equiv {
					return false
				}
			}
		}
	}

	// If both have suffixes, they must match
	return (name1Suffix != "" || name2Suffix != "") && name1Suffix != name2Suffix
}

func hasOppositeDirections(name1, name2 string) bool {
	name1Lower := strings.ToLower(name1)
	name2Lower := strings.ToLower(name2)

	for word, opposite := range oppositeWords {
		if strings.Contains(name1Lower, word) && strings.Contains(name2Lower, opposite) {
			return true
		}
	}
	return false
}

func wordMatch(name1, name2 string) float64 {
	words1 := strings.Fields(strings.ToLower(name1))
	words2 := strings.Fields(strings.ToLower(name2))

	matches := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				matches++
				break
			}
		}
	}

	// Return percentage of matching words
	totalWords := float64(len(words1)+len(words2)) / 2
	return float64(matches) / totalWords
}

func (a *APIClient) GetRugbyDBTeams(store *db.Store) ([]RugbyDBTeam, error) {
	url := "https://www.rugbydatabase.co.nz/teams.php"
	var matchedTeams []RugbyDBTeam
	var unmatchedTeams []string
	type Match struct {
		RugbyDBTeam RugbyDBTeam  // Team from RugbyDB
		OurTeam     *models.Team // Our internal team that matches
	}
	var matches []Match
	matchCount := 0
	maxMatches := 10 // Only process first 10 matched teams

	done := make(chan bool)

	// Get all existing rugbydatabase team mappings
	existingMappings, err := store.GetAPIMappingsByType("rugbydatabase", "team")
	if err != nil {
		fmt.Printf("Error getting existing mappings: %v\n", err)
		return nil, err
	}

	// Create a map for quick lookup
	existingTeamMappings := make(map[string]string) // map[apiID]entityID
	for _, mapping := range existingMappings {
		if mapping.APIID == "" {
			fmt.Printf("Warning: Found mapping with empty API ID: %+v\n", mapping)
			continue
		}
		existingTeamMappings[mapping.APIID] = mapping.EntityID
	}

	fmt.Printf("\nFound %d existing rugbydatabase team mappings\n", len(existingTeamMappings))
	fmt.Printf("Mappings: %+v\n", existingTeamMappings)

	fmt.Printf("\nFetching teams from RugbyDB...\n")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "RugbyLiveAPI/1.0")

	resp, err := a.makeRequestWithRetries(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	currentCountry := ""
	var rugbyDBTeams []RugbyDBTeam
	doc.Find("h3, .wrapper").Each(func(i int, s *goquery.Selection) {
		select {
		case <-done:
			return
		default:
		}

		if s.Is("h3") {
			currentCountry = s.Text()
			// fmt.Printf("\nProcessing country: %s\n", currentCountry)
		} else {
			name := s.Find(".playerLink a").Text()
			// fmt.Printf("Found team: %s\n", name)
			// Get the full image URL
			logoURL := ""
			if imgSrc, exists := s.Find(".img img").Attr("src"); exists {
				if strings.HasPrefix(imgSrc, "http") {
					logoURL = imgSrc
				} else {
					logoURL = "https://www.rugbydatabase.co.nz/" + strings.TrimPrefix(imgSrc, "/")
				}
			}
			teamLink, _ := s.Find(".playerLink a").Attr("href")

			// Extract teamId from link (team/index.php?teamId=XXX)
			teamID := ""
			if strings.Contains(teamLink, "teamId=") {
				teamID = strings.Split(teamLink, "teamId=")[1]
			}

			// Clean up the name (remove "Logo" suffix)
			name = strings.TrimSuffix(strings.TrimSpace(name), " Logo")

			// Create a URL-friendly ID
			id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

			team := RugbyDBTeam{
				ID:      id,
				TeamID:  teamID,
				Name:    name,
				Country: currentCountry,
				LogoURL: logoURL,
			}
			rugbyDBTeams = append(rugbyDBTeams, team)

			// First check if we already have an API mapping
			mapping, _ := store.GetAPIMappingByAPIID("rugbydatabase", team.TeamID, "team")
			if mapping != nil {
				matchingTeam, _ := store.GetTeamByID(mapping.EntityID)
				if matchingTeam != nil {
					team.InternalID = matchingTeam.ID
					matchedTeams = append(matchedTeams, team)
					matches = append(matches, Match{
						RugbyDBTeam: team,
						OurTeam:     matchingTeam,
					})
					matchCount++
					if matchCount >= maxMatches {
						close(done)
						return
					}
					return
				}
			}

			// Try to find matching team in our database
			if matchingTeam, err := a.FindMatchingTeam(store, team); err == nil {
				team.InternalID = matchingTeam.ID
				matchedTeams = append(matchedTeams, team)
				matches = append(matches, Match{
					RugbyDBTeam: team,
					OurTeam:     matchingTeam,
				})

				// Add RugbyDB name as an alternate name if different
				needsUpdate := false
				if team.Name != matchingTeam.Name {
					if matchingTeam.AltNames == nil {
						matchingTeam.AltNames = []string{}
					}
					// Check if name already exists in alternates
					exists := false
					for _, altName := range matchingTeam.AltNames {
						if altName == team.Name {
							exists = true
							break
						}
					}
					if !exists {
						matchingTeam.AltNames = append(matchingTeam.AltNames, team.Name)
						needsUpdate = true
					}
				}

				if !strings.HasSuffix(team.LogoURL, "TeamImage.webp") {
					fmt.Printf("\nProcessing logo for %s:\n", team.Name)
					fmt.Printf("- Source URL: %s\n", team.LogoURL)
					destPath := fmt.Sprintf("logos/teams/%s/%s/%s.png",
						matchingTeam.Country.Code,
						strings.TrimPrefix(matchingTeam.ID, matchingTeam.Country.Code+"-"),
						matchingTeam.Name,
					)
					fmt.Printf("- Destination: %s\n", destPath)
					// Check if file already exists
					if _, err := os.Stat(destPath); os.IsNotExist(err) {
						// Download and store the image
						uploadedURL, err := a.downloadAndStoreImage(team.LogoURL, destPath)
						if err == nil {
							fmt.Printf("- Success! New URL: %s\n", uploadedURL)
							matchingTeam.LogoURL = uploadedURL
							matchingTeam.LogoSource = "rugbydatabase"
							needsUpdate = true
						} else {
							fmt.Printf("- Failed to upload: %v\n", err)
						}
					} else {
						fmt.Printf("- Image already exists at %s\n", destPath)
					}
				}

				// Update team if either logo or alternate names changed
				if needsUpdate {
					if err := store.UpsertTeam(matchingTeam); err != nil {
						fmt.Printf("Error updating team %s: %v\n", matchingTeam.ID, err)
					}
				}

				// Create API mapping
				mapping := &models.APIMapping{
					EntityID:   matchingTeam.ID,
					APIName:    "rugbydatabase",
					APIID:      team.TeamID,
					EntityType: "team",
				}
				if err := store.UpsertAPIMapping(mapping); err != nil {
					fmt.Printf("Error creating API mapping for team %s: %v\n", team.Name, err)
				}

				matchCount++
				if matchCount >= maxMatches {
					close(done)
					return
				}
			} else {
				// fmt.Printf("\n=== Detailed matching attempt for %s ===\n", team.Name)
				unmatchedTeams = append(unmatchedTeams, fmt.Sprintf("%s (%s)", name, currentCountry))
			}
		}
	})

	// Print all matches at the end
	fmt.Printf("\n=== All Matches ===\n")
	for i, match := range matches {
		fmt.Printf("\n%d. MATCHED: %s\n", i+1, match.RugbyDBTeam.Name)
		fmt.Printf("RugbyDB Team: %s (%s)\n", match.RugbyDBTeam.Name, match.RugbyDBTeam.Country)
		fmt.Printf("RugbyDB ID: %s\n", match.RugbyDBTeam.TeamID)
		fmt.Printf("Our Team: %s (%s)\n", match.OurTeam.Name, match.OurTeam.Country.Name)
		fmt.Printf("Internal ID: %s\n", match.OurTeam.ID)
		fmt.Printf("Current Alternate Names: %v\n", match.OurTeam.AltNames)
		if nickname := TeamNameMapping[match.OurTeam.Name]; nickname == match.RugbyDBTeam.Name {
			fmt.Printf("Trying to add nickname: %s\n", nickname)
		}
		fmt.Printf("===================\n")
	}
	fmt.Printf("\nTotal Matches: %d\n", len(matches))

	if len(unmatchedTeams) > 0 {
		// Create filename with timestamp
		filename := fmt.Sprintf("unmatched_teams_%s.txt", time.Now().Format("2006-01-02_15-04-05"))

		// Create and write to file
		f, err := os.Create(filename)
		if err != nil {
			return matchedTeams, fmt.Errorf("failed to create unmatched teams file: %v", err)
		}
		defer f.Close()

		fmt.Fprintf(f, "Unmatched teams (%d):\n", len(unmatchedTeams))
		for _, team := range unmatchedTeams {
			fmt.Fprintf(f, "- %s\n", team)
		}

		fmt.Printf("\nUnmatched teams written to: %s\n", filename)
	}

	return matchedTeams, nil
}

func (a *APIClient) FindMatchingTeam(store *db.Store, rugbyDBTeam RugbyDBTeam) (*models.Team, error) {
	// fmt.Printf("\nAttempting to match: %s from %s\n", rugbyDBTeam.Name, rugbyDBTeam.Country)

	teams, err := store.GetAllTeams()
	if err != nil {
		return nil, err
	}

	// Filter teams by country first
	var countryTeams []*models.Team
	for _, team := range teams {
		// Clean up country names for comparison
		dbCountry := strings.ReplaceAll(team.Country.Name, "-", " ")
		rugbyDBCountry := strings.ReplaceAll(rugbyDBTeam.Country, "-", " ")
		if strings.EqualFold(dbCountry, rugbyDBCountry) {
			countryTeams = append(countryTeams, team)
		}
	}
	// fmt.Printf("Found %d teams from %s\n", len(countryTeams), rugbyDBTeam.Country)

	// Normalize the RugbyDB team name
	normalizedName := TeamNameNormalizer(rugbyDBTeam.Name)
	// fmt.Printf("- Normalized name: %s\n", normalizedName)

	// First try exact match with country
	for _, team := range countryTeams {
		// First check if this team maps to the RugbyDB name
		fmt.Printf("Checking if %s maps to %s\n", team.Name, rugbyDBTeam.Name)
		if nickname := TeamNameMapping[team.Name]; nickname == rugbyDBTeam.Name {
			fmt.Printf("Found nickname match: %s -> %s\n", team.Name, nickname)
			return team, nil
		}

		// Skip if directions are opposite
		if hasOppositeDirections(rugbyDBTeam.Name, team.Name) {
			continue
		}
		// Skip if suffixes don't match
		if hasDifferentSuffix(rugbyDBTeam.Name, team.Name) {
			// fmt.Printf("  - Skipped: different suffixes\n")
			continue
		}
		if TeamNameNormalizer(team.Name) == normalizedName {
			fmt.Printf("Found exact match: %s\n", team.ID)
			// Check if our team name maps to the RugbyDB team name
			fmt.Printf("Adding alternate name %s for team %s\n", rugbyDBTeam.Name, team.Name)
			if nickname := TeamNameMapping[team.Name]; nickname == rugbyDBTeam.Name {
				if team.AltNames == nil {
					team.AltNames = []string{}
				}
				team.AltNames = append(team.AltNames, rugbyDBTeam.Name)
				fmt.Printf("Adding alternate name %s for team %s\n", rugbyDBTeam.Name, team.Name)
				fmt.Printf("Full team object being saved: %+v\n", team)
				// Update the team in the database
				if err := store.UpsertTeam(team); err != nil {
					fmt.Printf("Error updating team alternate names for %s: %v\n", team.Name, err)
					fmt.Printf("Team data attempted to save: %+v\n", team)
				}
			}
			return team, nil
		}
	}

	return nil, fmt.Errorf("no matching team found for %s (%s)", rugbyDBTeam.Name, rugbyDBTeam.Country)
}
