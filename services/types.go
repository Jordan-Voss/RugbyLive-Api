package services

import (
	"strings"
)

// Move all type definitions here (CountriesResponse, teamChange, etc.)

type CountriesResponse struct {
	Get        string `json:"get"`
	Parameters []any  `json:"parameters"`
	Results    int    `json:"results"`
	Response   []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Code string `json:"code"`
		Flag string `json:"flag"`
	} `json:"response"`
}

type countryChange struct {
	Name    string
	Code    string
	Changes map[string]interface{}
	IsNew   bool
}

type TeamSearchParams struct {
	CountryID string
	LeagueID  string
	Season    int
}

type TeamsResponse struct {
	Get        string `json:"get"`
	Parameters struct {
		CountryID string `json:"country_id"`
	} `json:"parameters"`
	Results  int         `json:"results"`
	Errors   interface{} `json:"errors"`
	Response []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Logo     string `json:"logo"`
		National bool   `json:"national"`
		Founded  int    `json:"founded"`
		Arena    struct {
			Name     string      `json:"name"`
			Capacity interface{} `json:"capacity"`
			Location string      `json:"location"`
		} `json:"arena"`
		Country struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
			Flag string `json:"flag"`
		} `json:"country"`
	} `json:"response"`
}

type teamChange struct {
	Name    string
	ID      string
	Changes map[string]interface{}
	IsNew   bool
}

type failedTeam struct {
	Name        string
	CountryID   int
	CountryName string
	Reason      string
	TeamData    interface{} // Store full team data for debugging
}

type APISportsLeaguesResponse struct {
	Get        string `json:"get"`
	Parameters []any  `json:"parameters"`
	Results    int    `json:"results"`
	Response   []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Logo    string `json:"logo"`
		Country struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
			Flag string `json:"flag"`
		} `json:"country"`
		Seasons []struct {
			Year    int    `json:"season"`
			Current bool   `json:"current"`
			Start   string `json:"start"`
			End     string `json:"end"`
		} `json:"seasons"`
	} `json:"response"`
}

type leagueChange struct {
	Name    string
	ID      string
	Changes map[string]interface{}
	IsNew   bool
}

type ESPNTeamInfo struct {
	Name        string                 `json:"name"`
	LogoURL     string                 `json:"logo_url"`
	Stadium     string                 `json:"stadium"`
	FoundedYear string                 `json:"founded_year"`
	HeadCoach   string                 `json:"head_coach"`
	Players     []ESPNPlayer           `json:"players,omitempty"`
	RawData     map[string]interface{} `json:"raw_data"` // Store any other data we find
}

type ESPNPlayer struct {
	Name     string `json:"name"`
	Position string `json:"position"`
	Caps     string `json:"caps"`
	Club     string `json:"club"`
}

// ESPN team URL mapping
var ESPNTeamURLs = map[string]string{
	"new zealand":  "https://www.espn.com/rugby/team/_/id/8/new-zealand",
	"south africa": "https://www.espn.com/rugby/team/_/id/9/south-africa",
	"australia":    "https://www.espn.com/rugby/team/_/id/7/australia",
	"england":      "https://www.espn.com/rugby/team/_/id/1/england",
	"ireland":      "https://www.espn.com/rugby/team/_/id/3/ireland",
	"france":       "https://www.espn.com/rugby/team/_/id/2/france",
	"wales":        "https://www.espn.com/rugby/team/_/id/4/wales",
	"scotland":     "https://www.espn.com/rugby/team/_/id/5/scotland",
}

type ESPNLeague struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Add these types
type WikidataTeam struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Nickname        string   `json:"nickname,omitempty"`
	LogoURL         string   `json:"logo_url,omitempty"`
	Country         string   `json:"country,omitempty"`
	Founded         string   `json:"founded,omitempty"`
	Stadium         string   `json:"stadium,omitempty"`
	Coach           string   `json:"coach,omitempty"`
	Leagues         []string `json:"leagues,omitempty"`
	Competitions    []string `json:"competitions,omitempty"`
	Players         []string `json:"players,omitempty"`
	Website         string   `json:"website,omitempty"`
	Facebook        string   `json:"facebook,omitempty"`
	Twitter         string   `json:"twitter,omitempty"`
	Instagram       string   `json:"instagram,omitempty"`
	KitManufacturer string   `json:"kit_manufacturer,omitempty"`
	Sponsors        []string `json:"sponsors,omitempty"`
	FIFACode        string   `json:"fifa_code,omitempty"`
}

type WikidataResponse struct {
	Entities map[string]struct {
		Labels map[string]struct {
			Value string `json:"value"`
		} `json:"labels"`
		Claims map[string][]struct {
			MainSnak struct {
				DataValue struct {
					Value interface{} `json:"value"`
				} `json:"datavalue"`
			} `json:"mainsnak"`
		} `json:"claims"`
	} `json:"entities"`
}

type WikidataSPARQLResponse struct {
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
			WikidataID struct {
				Value string `json:"value"`
			} `json:"wikidataID"`
			Nickname struct {
				Value string `json:"value"`
			} `json:"nickname"`
			Logo struct {
				Value string `json:"value"`
			} `json:"logo"`
			Founded struct {
				Value string `json:"value"`
			} `json:"founded"`
			Stadium struct {
				Value string `json:"value"`
			} `json:"stadium"`
			StadiumLabel struct {
				Value string `json:"value"`
			} `json:"stadiumLabel"`
			Coach struct {
				Value string `json:"value"`
			} `json:"coach"`
			CoachLabel struct {
				Value string `json:"value"`
			} `json:"coachLabel"`
			League struct {
				Value string `json:"value"`
			} `json:"league"`
			LeagueLabel struct {
				Value string `json:"value"`
			} `json:"leagueLabel"`
			Competition struct {
				Value string `json:"value"`
			} `json:"competition"`
			CompetitionLabel struct {
				Value string `json:"value"`
			} `json:"competitionLabel"`
			Player struct {
				Value string `json:"value"`
			} `json:"player"`
			PlayerLabel struct {
				Value string `json:"value"`
			} `json:"playerLabel"`
			Website struct {
				Value string `json:"value"`
			} `json:"website"`
			Facebook struct {
				Value string `json:"value"`
			} `json:"facebook"`
			Twitter struct {
				Value string `json:"value"`
			} `json:"twitter"`
			Instagram struct {
				Value string `json:"value"`
			} `json:"instagram"`
			KitManufacturer struct {
				Value string `json:"value"`
			} `json:"kitManufacturer"`
			KitManufacturerLabel struct {
				Value string `json:"value"`
			} `json:"kitManufacturerLabel"`
			Sponsor struct {
				Value string `json:"value"`
			} `json:"sponsor"`
			SponsorLabel struct {
				Value string `json:"value"`
			} `json:"sponsorLabel"`
			FIFACode struct {
				Value string `json:"value"`
			} `json:"fifaCode"`
		} `json:"bindings"`
	} `json:"results"`
}

type RugbyDBTeam struct {
	ID         string `json:"id"`
	TeamID     string `json:"team_id"`
	InternalID string `json:"internal_id,omitempty"`
	Name       string `json:"name"`
	Country    string `json:"country"`
	LogoURL    string `json:"logo_url,omitempty"`
}

// TeamNameMapping maps RugbyDB team names to standardized team names
var TeamNameMapping = map[string]string{
	"New Zealand":  "All Blacks",
	"South Africa": "Springboks",
	"Australia":    "Wallabies",
	"France":       "Les Bleus",
	"New ZealandW": "Black Ferns W",
}

// TeamNameNormalizer removes common suffixes and standardizes team names
func TeamNameNormalizer(name string) string {
	// First check if we have a direct mapping
	if standardName, exists := TeamNameMapping[name]; exists {
		return standardName
	}

	// Remove common suffixes
	suffixes := []string{
		" Rugby",
		" RFC",
		" Union",
		" XV",
	}

	normalized := name
	for _, suffix := range suffixes {
		normalized = strings.TrimSuffix(normalized, suffix)
	}

	return normalized
}
