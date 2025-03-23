package services

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"rugby-live-api/db"
	"rugby-live-api/models"
	"rugby-live-api/services/rugbydb"

	"github.com/PuerkitoBio/goquery"
)

func hasDifferentSuffix(name1, name2 string) bool {
	// Special case: if one name ends with " (W)" and the other contains "Women"
	if (strings.HasSuffix(name1, " (W)") && strings.Contains(name2, "Women")) ||
		(strings.HasSuffix(name2, " (W)") && strings.Contains(name1, "Women")) {
		// fmt.Printf("- Special case match for Women/(W)\n")
		return false
	}

	name1Suffix := ""
	name2Suffix := ""

	for _, suffix := range rugbydb.TeamSuffixes {
		if strings.HasSuffix(name1, suffix) {
			name1Suffix = suffix
		}
		if strings.HasSuffix(name2, suffix) {
			name2Suffix = suffix
		}
	}

	// Check if suffixes are equivalent
	if name1Suffix != "" && name2Suffix != "" {
		// fmt.Printf("- Checking if suffixes '%s' and '%s' are equivalent\n", name1Suffix, name2Suffix)
		if name1Suffix == name2Suffix {
			// fmt.Printf("- Exact suffix match\n")
			return false
		}
		if equivalents, exists := rugbydb.EquivalentSuffixes[name1Suffix]; exists {
			// fmt.Printf("- Found equivalents for '%s': %v\n", name1Suffix, equivalents)
			for _, equiv := range equivalents {
				if name2Suffix == equiv {
					// fmt.Printf("- Found equivalent suffix match\n")
					return false
				}
			}
		}
	}

	// fmt.Printf("- Result: Different suffixes\n")
	return (name1Suffix != "" || name2Suffix != "") && name1Suffix != name2Suffix
}

func hasOppositeDirections(name1, name2 string) bool {
	name1Lower := strings.ToLower(name1)
	name2Lower := strings.ToLower(name2)

	for word, opposite := range rugbydb.OppositeWords {
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

func normalizeCountryName(name string) string {
	// Map of special cases
	countryMap := map[string]string{
		"the fiji islands":              "Fiji",
		"fiji islands":                  "Fiji",
		"fiji the fiji islands":         "Fiji",
		"France, French Republic":       "France",
		"france, french republic":       "France",
		"united kingdom":                "Europe", // For British & Irish Lions
		"netherlands the":               "Netherlands",
		"portugal, portuguese republic": "Portugal",
		"Portugal Portuguese Republic":  "Portugal",
		"Russian Federation":            "Russia",
		"russia":                        "Russia",
		"russian federation":            "Russia",
		"United States of America":      "USA",
		"united states of america":      "USA",
	}

	// Just trim spaces but preserve case
	normalized := strings.TrimSpace(name)
	// fmt.Printf("Normalizing country name: '%s' -> ", name)

	// Check special cases
	if normalized, exists := countryMap[strings.ToLower(normalized)]; exists {
		// fmt.Printf("'%s' (special case)\n", normalized)
		return normalized
	}

	// fmt.Printf("'%s' (default)\n", normalized)
	return normalized
}

func (a *APIClient) GetRugbyDBTeams(store *db.Store, priorityTeams []string, countryFilter string) ([]RugbyDBTeam, error) {
	url := "https://www.rugbydatabase.co.nz/teams.php"
	var matchedTeams []RugbyDBTeam
	var unmatchedTeams []string
	type Match struct {
		RugbyDBTeam RugbyDBTeam  // Team from RugbyDB
		OurTeam     *models.Team // Our internal team that matches
	}
	var matches []Match

	// Create map of priority teams for quick lookup
	priorityMap := make(map[string]bool)
	for _, name := range priorityTeams {
		priorityMap[name] = true
	}

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
			// Skip if we're filtering by country and this isn't the one we want
			if countryFilter != "" && normalizeCountryName(currentCountry) != normalizeCountryName(countryFilter) {
				return
			}
		} else {
			// Skip if we're not in the desired country
			if countryFilter != "" && normalizeCountryName(currentCountry) != normalizeCountryName(countryFilter) {
				return
			}
			name := s.Find(".playerLink a").Text()
			// If this is a priority team, ensure we process it
			isPriority := priorityMap[name]

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
			normalizedName := normalizeCountryName(currentCountry)
			team := RugbyDBTeam{
				ID:      id,
				TeamID:  teamID,
				Name:    name,
				Country: normalizedName,
				LogoURL: logoURL,
			}
			rugbyDBTeams = append(rugbyDBTeams, team)
			// fmt.Printf("Mappings: %+v\n", rugbyDBTeams)

			// First check if we already have an API mapping
			mapping, _ := store.GetAPIMappingByAPIID("rugbydatabase", team.TeamID, "team")
			if mapping != nil {
				// fmt.Printf("Found mapping for %s\n", team.Name)
				matchingTeam, _ := store.GetTeamByID(mapping.EntityID)
				if matchingTeam != nil {
					team.InternalID = matchingTeam.ID
					matchedTeams = append(matchedTeams, team)
					matches = append(matches, Match{
						RugbyDBTeam: team,
						OurTeam:     matchingTeam,
					})
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
					destPath := fmt.Sprintf("logos/teams/%s/%s/%s.png",
						matchingTeam.Country.Code,
						strings.TrimPrefix(matchingTeam.ID, matchingTeam.Country.Code+"-"),
						matchingTeam.Name,
					)

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
							// fmt.Printf("- Failed to upload: %v\n", err)
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

				return
			} else {
				// If this was a priority team, create it
				if isPriority {
					// Create team regardless of logo
					newTeam, err := a.createTeamFromRugbyDB(store, team)
					if err == nil {
						team.InternalID = newTeam.ID
						matchedTeams = append(matchedTeams, team)
						matches = append(matches, Match{
							RugbyDBTeam: team,
							OurTeam:     newTeam,
						})
						fmt.Printf("Created new team for priority match: %s\n", team.Name)
					} else {
						fmt.Printf("Failed to create priority team %s: %v\n", team.Name, err)
					}
				}
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
		// Use fixed filename
		filename := "unmatched_teams.txt"

		// Create and write to file
		f, err := os.Create(filename)
		if err != nil {
			return matchedTeams, fmt.Errorf("failed to create unmatched teams file: %v", err)
		}
		defer f.Close()

		// Write as JSON-ready format
		fmt.Fprintf(f, "{\n  \"names\": [\n")
		for i, team := range unmatchedTeams {
			// Extract name but preserve W suffix
			name := strings.Split(team, " (")[0] // Remove country part
			if strings.Contains(strings.ToLower(team), "women") || strings.Contains(team, "(W)") {
				name = name + " W"
			}

			if i == len(unmatchedTeams)-1 {
				fmt.Fprintf(f, "    \"%s\"\n", name)
			} else {
				fmt.Fprintf(f, "    \"%s\",\n", name)
			}
		}
		fmt.Fprintf(f, "  ]\n}")

		fmt.Printf("\nUnmatched teams written to: %s\n", filename)
	}

	return matchedTeams, nil
}

func (a *APIClient) FindMatchingTeam(store *db.Store, rugbyDBTeam RugbyDBTeam) (*models.Team, error) {
	country, err := store.GetCountryByName(normalizeCountryName(rugbyDBTeam.Country))
	if err != nil {
		return nil, err
	}

	// Get teams for this country directly
	countryTeams, err := store.GetTeamsByCountryCode(country.Code)
	if err != nil {
		return nil, err
	}

	// First check TeamNameMapping for all teams
	for _, team := range countryTeams {
		if nickname := TeamNameMapping[team.Name]; nickname == rugbyDBTeam.Name {
			return team, nil
		}
	}

	// If this is a strict match team, only allow matching via TeamNameMapping
	if rugbydb.StrictMatchTeams[rugbyDBTeam.Name] {
		return nil, fmt.Errorf("no exact mapping found for strict match team %s", rugbyDBTeam.Name)
	}

	// First try exact match with country
	// fmt.Printf("Country teams: %+v\n", countryTeams)
	// fmt.Printf("Found %d teams in %s\n", len(countryTeams), country.Name)
	for _, team := range countryTeams {
		// Skip if suffixes don't match
		if hasDifferentSuffix(rugbyDBTeam.Name, team.Name) {
			continue
		}

		// Standardize women's team names for comparison only
		compareTeamName := team.Name
		compareRugbyDBName := rugbyDBTeam.Name
		// fmt.Printf("Comparing our team %s with RugbyDB team %s\n", compareTeamName, compareRugbyDBName)
		// Standardize U20/Under 20 variations to "U20" for comparison
		for _, suffix := range []string{" Under 20", " Under20", " U20"} {
			if strings.HasSuffix(compareTeamName, suffix) {
				compareTeamName = strings.TrimSuffix(compareTeamName, suffix) + " U20"
				break
			}
		}
		for _, suffix := range []string{" Under 20", " Under20", " U20"} {
			if strings.HasSuffix(compareRugbyDBName, suffix) {
				compareRugbyDBName = strings.TrimSuffix(compareRugbyDBName, suffix) + " U20"
				break
			}
		}

		// Standardize to single " W" suffix for comparison
		for _, suffix := range []string{" Women (W)", " (W)", " Women"} {
			if strings.HasSuffix(compareTeamName, suffix) {
				compareTeamName = strings.TrimSuffix(compareTeamName, suffix) + " W"
				if strings.Contains(team.Name, "Blues") {
					fmt.Printf("Standardized DB name: %s\n", compareTeamName)
				}
				break // Only replace one suffix
			}
		}
		for _, suffix := range []string{" Women (W)", " (W)", " Women"} {
			if strings.HasSuffix(compareRugbyDBName, suffix) {
				compareRugbyDBName = strings.TrimSuffix(compareRugbyDBName, suffix) + " W"
				if strings.Contains(rugbyDBTeam.Name, "Blues") {
					fmt.Printf("Standardized RugbyDB name: %s\n", compareRugbyDBName)
				}
				break // Only replace one suffix
			}
		}

		// First check if this team maps to the RugbyDB name
		if nickname := TeamNameMapping[team.Name]; nickname == rugbyDBTeam.Name {
			// fmt.Printf("Found nickname match: %s -> %s\n", team.Name, nickname)
			return team, nil
		}

		// Skip if directions are opposite
		if hasOppositeDirections(rugbyDBTeam.Name, team.Name) {
			continue
		}

		// For strict match teams, don't allow exact name matches
		if rugbydb.StrictMatchTeams[rugbyDBTeam.Name] && strings.EqualFold(team.Name, rugbyDBTeam.Name) {
			continue
		}

		if strings.EqualFold(compareTeamName, compareRugbyDBName) {
			// fmt.Printf("Found exact match: %s\n", team.ID)
			// Always add RugbyDB name as alternate if it's different
			if team.Name != rugbyDBTeam.Name {
				if team.AltNames == nil {
					team.AltNames = []string{}
				}
				// Check if name already exists in alternates
				exists := false
				for _, altName := range team.AltNames {
					if altName == rugbyDBTeam.Name {
						exists = true
						break
					}
				}
				if !exists {
					team.AltNames = append(team.AltNames, rugbyDBTeam.Name)
					if err := store.UpsertTeam(team); err != nil {
						fmt.Printf("Error updating team alternate names for %s: %v\n", team.Name, err)
					}
				}
			}
			return team, nil
		}
	}

	return nil, fmt.Errorf("no matching team found for %s (%s)", rugbyDBTeam.Name, rugbyDBTeam.Country)
}

func (a *APIClient) createTeamFromRugbyDB(store *db.Store, rugbyDBTeam RugbyDBTeam) (*models.Team, error) {
	// First ensure country exists
	country, err := store.GetCountryByName(rugbyDBTeam.Country)
	if err != nil {
		return nil, fmt.Errorf("failed to get country: %v", err)
	}

	// Create internal ID
	internalID := fmt.Sprintf("%s-%s",
		country.Code,
		strings.ToUpper(strings.ReplaceAll(rugbyDBTeam.Name, " ", "")),
	)

	// Create new team
	newTeam := &models.Team{
		ID:      internalID,
		Name:    rugbyDBTeam.Name,
		Country: *country,
	}

	// Upload logo if exists
	if rugbyDBTeam.LogoURL != "" && !strings.HasSuffix(rugbyDBTeam.LogoURL, "TeamImage.webp") {
		destPath := fmt.Sprintf("logos/teams/%s/%s/%s.png",
			country.Code,
			strings.TrimPrefix(internalID, country.Code+"-"),
			rugbyDBTeam.Name,
		)
		if uploadedURL, err := a.downloadAndStoreImage(rugbyDBTeam.LogoURL, destPath); err == nil {
			newTeam.LogoURL = uploadedURL
			newTeam.LogoSource = "rugbydatabase"
		}
	}

	// Save team to database
	if err := store.UpsertTeam(newTeam); err != nil {
		return nil, fmt.Errorf("failed to create team: %v", err)
	}

	// Create API mapping
	mapping := &models.APIMapping{
		EntityID:   newTeam.ID,
		APIName:    "rugbydatabase",
		APIID:      rugbyDBTeam.TeamID,
		EntityType: "team",
	}
	if err := store.UpsertAPIMapping(mapping); err != nil {
		fmt.Printf("Warning: failed to create API mapping for team %s: %v\n", newTeam.Name, err)
	}

	return newTeam, nil
}

type TeamCreateRequest struct {
	Names   []string `json:"names"`
	Country string   `json:"country"`
}

func (a *APIClient) GetLeaguesByYear(store *db.Store, year string, dryRun bool) ([]models.League, error) {
	// Check if year is in format "2024" or "2024-2025"
	parts := strings.Split(year, "-")
	var url string
	var seasonYear int
	var yearRange string

	if len(parts) == 2 {
		// Format is "2024-2025"
		singleYear, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid year format: %v", err)
		}
		url = fmt.Sprintf("https://www.rugbydatabase.co.nz/competitions.php?year=%d-%d", singleYear-1, singleYear)
		seasonYear = singleYear
		yearRange = year // Use the full range as provided
	} else {
		// Format is "2024"
		endYear, err := strconv.Atoi(year)
		if err != nil {
			return nil, fmt.Errorf("invalid end year format: %v", err)
		}
		url = fmt.Sprintf("https://www.rugbydatabase.co.nz/competitions.php?year=%s", year)
		seasonYear = endYear
		yearRange = fmt.Sprintf("%d", endYear) // Just use the single year
	}

	leagues, err := a.scrapeLeaguesFromURL(url, seasonYear, yearRange, store, dryRun)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Scraped leagues for year: %s\n", year)

	return leagues, nil
}

type LeagueProcessed struct {
	Name   string
	Status string // "existing", "new", "unmapped"
	Reason string // reason for unmapped status, if any
}

func cleanLeagueName(name string) string {
	// Remove year patterns like (2024-25), (2024-2025), (2024)
	name = regexp.MustCompile(`\s*\(\d{4}(?:-\d{2,4})?\)`).ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

func (a *APIClient) scrapeLeaguesFromURL(url string, year int, yearRange string, store *db.Store, dryRun bool) ([]models.League, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := a.makeRequestWithRetries(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var leagues []models.League
	var processed []LeagueProcessed

	doc.Find(".competition").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Find("h2").Text())
		if name == "" {
			name = strings.TrimSpace(s.Find("a").Text())
		}
		name = cleanLeagueName(name)

		// First check if this is a child league
		var countryInfo rugbydb.LeagueInfo
		if parentName, isChild := rugbydb.LeagueParentMap[name]; isChild {
			// Get country info from parent league
			if parentInfo, exists := rugbydb.LeagueCountryMap[parentName]; exists {
				countryInfo = parentInfo
			} else {
				processed = append(processed, LeagueProcessed{
					Name:   name,
					Status: "unmapped",
					Reason: fmt.Sprintf("parent league %s not found in country map", parentName),
				})
				return
			}
		} else {
			// Not a child league, get country info directly
			var exists bool
			countryInfo, exists = rugbydb.LeagueCountryMap[name]
			if !exists {
				processed = append(processed, LeagueProcessed{
					Name:   name,
					Status: "unmapped",
					Reason: "no country mapping found",
				})
				return
			}
		}

		// Get country details from database
		countryDetails, err := store.GetCountryByCode(countryInfo.Country)
		if err != nil {
			processed = append(processed, LeagueProcessed{
				Name:   name,
				Status: "unmapped",
				Reason: fmt.Sprintf("country %s not found in database", countryInfo.Country),
			})
			fmt.Printf("Error getting country %s from database: %v\n", countryInfo.Country, err)
			return
		}

		// Extract RugbyDB ID from the URL
		rugbyDBID := ""
		if href, exists := s.Find("a").Attr("href"); exists {
			if parts := strings.Split(href, "competitionId="); len(parts) > 1 {
				rugbyDBID = parts[1]
			}
		}

		// Create league ID from name and country
		fmt.Printf("Country: %s\n", countryDetails.Name)
		id := fmt.Sprintf("%s-%s",
			countryDetails.Code,
			strings.ToUpper(strings.ReplaceAll(name, " ", "-")),
		)

		// Try to find existing league by name
		existingLeague, err := store.GetLeagueByName(name)
		var leagueID string

		if err == nil {
			// Use existing league
			leagueID = existingLeague.ID
			leagues = append(leagues, *existingLeague)
			processed = append(processed, LeagueProcessed{
				Name:   name,
				Status: "existing",
			})
		} else {
			// Create new league...
			// Get competition format (League, Cup, etc.)
			format := "League"
			var structure []string
			if formatInfo, exists := rugbydb.LeagueFormats[name]; exists {
				format = formatInfo.Format
				structure = formatInfo.Phases
			} else if strings.Contains(strings.ToLower(name), "cup") {
				format = "Cup"
			}

			// Determine gender
			gender := "Men" // Default
			if strings.Contains(name, "(W)") || strings.Contains(strings.ToLower(name), "women") {
				gender = "Women"
			}

			// Only get and process logo for new leagues
			logoURL := ""
			var logoSource string = "rugbydatabase"
			if imgSrc, exists := s.Find("img").Attr("src"); exists {
				if strings.HasPrefix(imgSrc, "http") {
					logoURL = imgSrc
				} else {
					logoURL = "https://www.rugbydatabase.co.nz/" + strings.TrimPrefix(imgSrc, "/")
				}

				// Download and store the image
				if !dryRun && logoURL != "" {
					newLogoURL, err := a.downloadAndStoreImage(
						logoURL,
						fmt.Sprintf("logos/leagues/%s/logo.png", id),
					)
					if err != nil {
						fmt.Printf("Error downloading logo for league %s: %v\n", name, err)
					} else {
						logoURL = newLogoURL
					}
				}
			}

			// Convert country codes to Country objects
			var teamCountries []models.Country
			for _, code := range countryInfo.Countries {
				country, err := store.GetCountryByCode(code)
				if err == nil {
					teamCountries = append(teamCountries, *country)
				}
			}

			league := models.League{
				ID:            id,
				Name:          name,
				Country:       *countryDetails,
				TeamCountries: teamCountries,
				AltNames:      rugbydb.LeagueAltNames[name],
				Format:        format,
				Phases:        structure,
				Gender:        gender,
				International: rugbydb.InternationalCompetitions[name],
				LogoURL:       logoURL,
				LogoSource:    logoSource,
				ParentID:      nil, // Default to nil
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
				Seasons: []models.Season{{
					ID:        fmt.Sprintf("%s-SEASON-%d", id, year),
					LeagueID:  id,
					Year:      year,
					YearRange: yearRange,
					Current:   time.Now().Year() == year,
					StartDate: time.Date(year-1, 8, 1, 0, 0, 0, 0, time.UTC),
					EndDate:   time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC),
				}},
				Tier: rugbydb.LeagueTiers[name],
			}

			// Check if this league has a parent
			if parentName, hasParent := rugbydb.LeagueParentMap[name]; hasParent {
				// Try to find the parent league
				parentLeague, err := store.GetLeagueByName(parentName)
				if err == nil {
					league.ParentID = &parentLeague.ID
					// Inherit properties from parent
					league.Format = parentLeague.Format
					league.Phases = parentLeague.Phases
					league.TeamCountries = append(league.TeamCountries, parentLeague.TeamCountries...)
				} else {
					fmt.Printf("Warning: Parent league %s not found for %s\n", parentName, name)
				}
			}

			if err := store.UpsertLeague(&league); err != nil {
				fmt.Printf("Error creating league %s: %v\n", league.ID, err)
				return
			}

			// Check if this league has a successor
			if transition, hasSuccessor := rugbydb.LeagueSuccessors[name]; hasSuccessor {
				league.SuccessorID = &transition.SuccessorID
				if err := store.UpsertLeague(&league); err != nil {
					fmt.Printf("Error updating league successor %s: %v\n", league.ID, err)
				}
			}

			leagueID = league.ID
			leagues = append(leagues, league)
			processed = append(processed, LeagueProcessed{
				Name:   name,
				Status: "new",
			})
		}

		// Create season
		seasonID := fmt.Sprintf("%s-SEASON-%d", leagueID, year)
		season := models.Season{
			ID:        seasonID,
			LeagueID:  leagueID,
			Year:      year,
			YearRange: yearRange,
			Current:   time.Now().Year() == year,
			StartDate: time.Date(year-1, 8, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := store.UpsertSeason(&season); err != nil {
			fmt.Printf("Error creating season %s: %v\n", seasonID, err)
			return
		}

		// Update current season flag
		if err := store.UpdateCurrentSeason(leagueID); err != nil {
			fmt.Printf("Error updating current season for league %s: %v\n", leagueID, err)
		}

		// Create season API mapping
		seasonMapping := &models.APIMapping{
			EntityID:   seasonID,
			APIName:    "rugbydatabase",
			APIID:      rugbyDBID,
			EntityType: "league_season",
		}
		if err := store.UpsertAPIMapping(seasonMapping); err != nil {
			fmt.Printf("Error creating API mapping for season %s: %v\n", seasonID, err)
		}
	})

	// Write league statuses to file
	if err := a.writeLeaguesToFile(processed, yearRange); err != nil {
		fmt.Printf("Warning: failed to write leagues to file: %v\n", err)
	}

	return leagues, nil
}

func (a *APIClient) writeLeaguesToFile(processed []LeagueProcessed, year string) error {
	filename := fmt.Sprintf("leagues_%s.txt", year)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll("output", 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Open file for writing
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	// Write each league to file
	for _, p := range processed {
		if p.Status == "unmapped" {
			_, err = fmt.Fprintf(f, "%s (%s: %s)\n", p.Name, p.Status, p.Reason)
		} else {
			_, err = fmt.Fprintf(f, "%s (%s)\n", p.Name, p.Status)
		}
		if err != nil {
			return fmt.Errorf("failed to write to file: %v", err)
		}
	}

	return nil
}
