package rugbydb

import (
	"regexp"
	"strings"
)

var TeamSuffixes = []string{
	" Women (W)",
	" Women",
	" W",
	" (W)",
	" A",
	" B",
	" C",
	" XV",
	" U20",
	" Under 20",
	" Under20",
}

var InternationalCompetitions = map[string]bool{
	"Six Nations Championship":             true,
	"Six Nations":                          true,
	"Rugby Europe Championship":            true,
	"Rugby Europe":                         true,
	"Rugby World Cup":                      true,
	"Rugby Championship":                   true,
	"Pacific Nations Cup":                  true,
	"European Rugby Champions Cup":         true,
	"European Challenge Cup":               true,
	"EPCR Challenge Cup":                   true,
	"Nations Cup":                          true,
	"Autumn Nations Cup":                   true,
	"Autumn Nations Series":                true,
	"Pacific Four Series":                  true,
	"WXV":                                  true,
	"WOMEN'S SIX NATIONS CHAMPIONSHIP (W)": true,
	"British & Irish Lions Tour":           true,
	"International Friendly":               true,
	"International Friendly (W)":           true,
	"WXV (W)":                              true,
	"World Rugby U20 Championship":         true,
}

var LeagueAltNames = map[string][]string{
	"United Rugby Championship": {
		"URC",
	},
	"Rugby Championship": {
		"Tri Nations",
		"SANZAR Tri Nations",
		"The Rugby Championship",
	},
	"Pro14": {
		"Pro 14",
		"Guinness Pro14",
	},
	"Pro12": {
		"Pro 12",
		"RaboDirect Pro12",
		"Magners League",
	},
	"Celtic League": {
		"Celtic Rugby League",
	},
	"European Champions Cup": {
		"Champions Cup",
		"Heineken Champions Cup",
	},
	"European Rugby Champions Cup": {
		"Champions Cup",
		"Heineken Champions Cup",
	},
	"Heineken Cup": {
		"European Cup",
	},
	"EPCR Challenge Cup": {
		"European Challenge Cup",
		"Challenge Cup",
		"Investec Rugby Challenge Cup",
	},
	"British & Irish Lions Tour": {
		"British & Irish Lions",
	},
	"National Provincial Championship": {
		"Bunnings NPC",
		"NPC",
	},
	"Super Rugby Aupiki (W)": {
		"Super Rugby Aupiki",
	},
	"WXV (W)": {
		"WXV 2024 (W)",
	},
}

type LeagueInfo struct {
	Country   string
	Countries []string
}

var LeagueCountryMap = map[string]LeagueInfo{
	"Super Rugby Pacific": {
		Country:   "OCE",
		Countries: []string{"AUS", "NZL", "FJI", "SAM"},
	},
	"Super Rugby Aupiki (W)": {
		Country:   "OCE",
		Countries: []string{"NZL"},
	},
	"United Rugby Championship": {
		Country:   "EUR",
		Countries: []string{"IRL", "ITA", "SCO", "RSA", "WAL"},
	},
	"Premiership Rugby": {
		Country:   "ENG",
		Countries: []string{"ENG"},
	},
	"RFU Championship": {
		Country:   "ENG",
		Countries: []string{"ENG"},
	},
	"Top 14": {
		Country:   "FRA",
		Countries: []string{"FRA"},
	},
	"Currie Cup": {
		Country:   "RSA",
		Countries: []string{"RSA"},
	},
	"National Provincial Championship": {
		Country:   "NZL",
		Countries: []string{"NZL"},
	},
	"Major League Rugby": {
		Country:   "USA",
		Countries: []string{"USA"},
	},
	"Ranfurly Shield": {
		Country:   "NZL",
		Countries: []string{"NZL"},
	},

	"Rugby Europe Championship": {
		Country:   "EUR",
		Countries: []string{"GEO", "ROU", "POR", "ESP", "GER", "SWI", "NED", "BEL"},
	},
	"Women's Six Nations Championship (W)": {
		Country:   "EUR",
		Countries: []string{"ENG", "FRA", "IRL", "SCO", "WAL", "ITA"},
	},
	"Six Nations Championship": {
		Country:   "EUR",
		Countries: []string{"ENG", "FRA", "IRL", "SCO", "WAL", "ITA"},
	},
	"Six Nations Under 20s Championship": {
		Country:   "EUR",
		Countries: []string{"ENG", "FRA", "IRL", "SCO", "WAL", "ITA"},
	},
	"Autumn Nations Series": {
		Country:   "WLD",
		Countries: []string{"ARG", "ASM", "AUS", "AUT", "BEL", "BRA", "CAN", "CHL", "CHN", "CIV", "COK", "COL", "CZE", "ENG", "ESP", "FJI", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"},
	},
	"Summer Test Series": {
		Country:   "WLD",
		Countries: []string{}, // Will inherit from parent "Summer Tests"
	},
	"Summer Tests": {
		Country:   "WLD",
		Countries: []string{"ARG", "ASM", "AUS", "AUT", "BEL", "BRA", "CAN", "CHL", "CHN", "CIV", "COK", "COL", "CZE", "ENG", "ESP", "FJI", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"}, // Will inherit from parent "Summer Tests"
	},
	"International Friendly": {
		Country:   "WLD",
		Countries: []string{"ARG", "ASM", "AUS", "AUT", "BEL", "BRA", "CAN", "CHL", "CHN", "CIV", "COK", "COL", "CZE", "ENG", "ESP", "FJI", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"},
	},
	"International Friendly (W)": {
		Country:   "WLD",
		Countries: []string{"ARG", "ASM", "AUS", "AUT", "BEL", "BRA", "CAN", "CHL", "CHN", "CIV", "COK", "COL", "CZE", "ENG", "ESP", "FJI", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"},
	},
	"Rugby World Cup": {
		Country:   "WLD",
		Countries: []string{"ARG", "ASM", "AUS", "AUT", "BEL", "BRA", "CAN", "CHL", "CHN", "CIV", "COK", "COL", "CZE", "ENG", "ESP", "FJI", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"}, // Will inherit from parent "Summer Tests"
	},
	"The Rugby Championship": {
		Country:   "WLD",
		Countries: []string{"ARG", "AUS", "RSA", "NZL"},
	},
	"The Rugby Championship U20": {
		Country:   "WLD",
		Countries: []string{"ARG", "AUS", "RSA", "NZL"},
	},
	"British & Irish Lions Tour": {
		Country:   "WLD",
		Countries: []string{"AUS", "ENG", "SCO", "WAL", "IRL", "NZL", "RSA"},
	},
	"Premiership Rugby Cup": {
		Country:   "ENG",
		Countries: []string{"ENG"},
	},
	"EPCR Challenge Cup": {
		Country:   "EUR",
		Countries: []string{"FRA", "ITA", "ENG", "SCO", "WAL", "IRL", "GEO"},
	},
	"European Champions Cup": {
		Country:   "EUR",
		Countries: []string{"FRA", "ITA", "ENG", "SCO", "WAL", "IRL"},
	},
	"Pro D2": {
		Country:   "FRA",
		Countries: []string{"FRA"},
	},
	"Japan Rugby League One - Division 1": {
		Country:   "JPN",
		Countries: []string{"JPN"},
	},
	"Japan Rugby League One - Division 2": {
		Country:   "JPN",
		Countries: []string{"JPN"},
	},
	"Japan Rugby League One - Division 3": {
		Country:   "JPN",
		Countries: []string{"JPN"},
	},
	"Bledisloe Cup": {
		Country:   "OCE",
		Countries: []string{"AUS", "NZL"},
	},
	"Laurie O'Reilly Cup (W)": {
		Country:   "OCE",
		Countries: []string{"AUS", "NZL"},
	},
	"Farah Palmer Cup (W)": {
		Country:   "NZL",
		Countries: []string{"NZL"},
	},
	"Heartland Championship": {
		Country:   "NZL",
		Countries: []string{"NZL"},
	},
	"JJ Stewart Trophy (W)": {
		Country:   "NZL",
		Countries: []string{"NZL"},
	},
	"Pacific Nations Cup": {
		Country:   "WLD",
		Countries: []string{"FJI", "SAM", "TGA", "USA", "JPN", "CAN"},
	},
	"World Rugby U20 Championship": {
		Country:   "WLD",
		Countries: []string{"ARG", "AUS", "BRA", "CAN", "CHL", "CHN", "CIV", "CZE", "ENG", "ESP", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "TUN", "TUR", "TUV", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"},
	},
	"WXV (W)": {
		Country:   "WLD",
		Countries: []string{"ARG", "AUS", "BRA", "CAN", "CHL", "CHN", "CIV", "CZE", "ENG", "ESP", "FRA", "GEO", "GER", "HKG", "IRL", "ITA", "JPN", "KAZ", "KEN", "KOR", "LKA", "MDG", "NAM", "NIU", "NLD", "NZL", "PHL", "PNG", "POL", "POR", "PRY", "ROU", "RSA", "RUS", "SAM", "SAU", "SCO", "SGP", "SWE", "SWI", "TGA", "THA", "TUN", "TUR", "TUV", "UAE", "UGA", "UGY", "USA", "VEN", "VUT", "WAL"},
	},
}
var OppositeWords = map[string]string{
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
var EquivalentSuffixes = map[string][]string{
	" W":         {" Women", " (W)", " Women (W)"},
	" Women":     {" W", " (W)", " Women (W)"},
	" (W)":       {" W", " Women", " Women (W)"},
	" Women (W)": {" W", " Women", " (W)"},
	" U20":       {" Under 20", " Under20"},
	" Under 20":  {" U20", " Under 20"},
	" Under20":   {" U20", " Under 20"},
}

// Teams that should only match with their exact mapping
var StrictMatchTeams = map[string]bool{
	"Cardiff": true, // Should only match with "Cardiff Rugby"
}

var LeagueTiers = map[string]int{
	"Super Rugby Pacific":          1,
	"United Rugby Championship":    1,
	"Premiership Rugby":            1,
	"Top 14":                       1,
	"European Rugby Champions Cup": 1,
	"Rugby Championship":           1,
	"Six Nations Championship":     1,
	"Rugby World Cup":              1,

	"European Challenge Cup":           2,
	"Currie Cup":                       2,
	"Rugby Europe Championship":        2,
	"National Provincial Championship": 2,
	"Major League Rugby":               1,

	"Super Rugby Aupiki (W)":              1,
	"Six Nations Championship (W)":        1,
	"ProD2":                               2,
	"The Rugby Championship":              1,
	"The Rugby Championship U20":          1,
	"British & Irish Lions Tour":          1,
	"Pro D2":                              2,
	"Japan Rugby League One - Division 1": 1,
	"Japan Rugby League One - Division 2": 2,
	"Japan Rugby League One - Division 3": 3,
	"Farah Palmer Cup (W)":                2,
	"Heartland Championship":              3,
	"JJ Stewart Trophy (W)":               2,
	"Pacific Nations Cup":                 1,
	"World Rugby U20 Championship":        1,
	"WXV (W)":                             1,
}

type CompetitionFormat struct {
	Format string   // "League", "Cup", "Hybrid", "Series", "Friendly"
	Phases []string // ["League", "Knockout"] or ["Series"] or ["Friendly"]
}

var LeagueFormats = map[string]CompetitionFormat{
	"United Rugby Championship": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"European Rugby Champions Cup": {
		Format: "Hybrid",
		Phases: []string{"Pools", "Playoffs"},
	},
	"Top 14": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"Super Rugby Pacific": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"ProD2": {
		Format: "League",
		Phases: []string{"League", "Playoffs"},
	},
	"Six Nations Championship": {
		Format: "League",
		Phases: []string{"League"},
	},
	"Six Nations Championship (W)": {
		Format: "League",
		Phases: []string{"League"},
	},
	"Rugby World Cup": {
		Format: "Hybrid",
		Phases: []string{"Pools", "Playoffs"},
	},
	"Autumn Nations Series": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"Autumn Nations Cup": {
		Format: "League",
		Phases: []string{"League"},
	},
	"Autumn Nations League": {
		Format: "League",
		Phases: []string{"League"},
	},
	"Summer Test Series": {
		Format: "Series",
		Phases: []string{"Series"},
	},
	"Autumn Test Series": {
		Format: "Series",
		Phases: []string{"Series"},
	},
	"International Test Match": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"Summer Tests": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"Autumn Test": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"Major League Rugby": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"The Rugby Championship": {
		Format: "League",
		Phases: []string{"League"},
	},
	"The Rugby Championship U20": {
		Format: "League",
		Phases: []string{"League"},
	},
	"British & Irish Lions Tour": {
		Format: "Series",
		Phases: []string{"Tour Match", "Test Match"},
	},
	"Premiership Rugby Cup": {
		Format: "Cup",
		Phases: []string{"Playoffs"},
	},
	"International Friendly": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"International Friendly (W)": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"Pro D2": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"RFU Championship": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"Bledisloe Cup": {
		Format: "Series",
		Phases: []string{"Test Match"},
	},
	"Laurie O'Reilly Cup (W)": {
		Format: "Series",
		Phases: []string{"Test Match"},
	},
	"National Provincial Championship": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"Farah Palmer Cup (W)": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"Heartland Championship": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"Ranfurly Shield": {
		Format: "Lineal",
		Phases: []string{"Lineal"},
	},
	"Pacific Nations Cup": {
		Format: "Hybrid",
		Phases: []string{"League", "Playoffs"},
	},
	"World Rugby U20 Championship": {
		Format: "Hybrid",
		Phases: []string{"Pools", "Playoffs"},
	},
	"WXV (W)": {
		Format: "League",
		Phases: []string{"League"},
	},
	"WXV Qualifiers (W)": {
		Format: "Knockout",
		Phases: []string{"Knockout"},
	},
	"WXV Warm Up Games (W)": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
}

var LeagueParentMap = map[string]string{
	"All Blacks in Europe":               "Autumn Nations Series",
	"Summer Test Series":                 "Summer Tests",
	"British & Irish Lions in Australia": "British & Irish Lions Tour",
	"Australia A in England":             "International Friendly",
	"Ireland A in England":               "International Friendly",
	"All Blacks & Fiji in United States": "International Friendly",
	"Bledisloe Cup":                      "The Rugby Championship",
	"All Blacks XV in Europe":            "International Friendly",
	"Argentina in Europe":                "Autumn Nations Series",
	"Argentina XV in Europe":             "International Friendly",
	"Argentina in Uruguay":               "International Friendly",
	"Australia in Europe":                "Autumn Nations Series",
	"Belgium in South America":           "Summer Tests",
	"Black Ferns in England (W)":         "International Friendly (W)",
	"Brazil in Hong Kong":                "International Friendly",
	"Canada in Europe":                   "International Friendly",
	"Chile in Europe":                    "International Friendly",
	"England in Japan":                   "Summer Tests",
	"England in New Zealand":             "Summer Test Series",
	"Bunnings NPC":                       "National Provincial Championship",
	"All Blacks in Japan":                "International Friendly",
	"Fiji in Australia (W)":              "International Friendly (W)",
	"Fiji in Europe":                     "Summer Tests",
	"Fiji in Europe (2)":                 "Autumn Nations Series",
	"Fiji in Scotland (W)":               "International Friendly (W)",
	"France in England (W)":              "International Friendly (W)",
	"France in South America":            "Summer Tests",
	"Georgia in Australia & Japan":       "Summer Tests",
	"Georgia in Italy":                   "Autumn Nations Series",
	"Germany in Europe":                  "International Friendly",
	"Hong Kong in South America":         "Summer Tests",
	"Ireland in South Africa":            "Summer Test Series",
	"Italy in Pacific Islands & Japan":   "Summer Tests",
	"Japan in Europe":                    "Autumn Nations Series",
	"Japan in Italy (W)":                 "International Friendly (W)",
	"Kenya in Uganda":                    "International Friendly",
	"Killik Cup":                         "International Friendly",
	"Laurie O'Reilly Cup (W)":            "Pacific Four Series(W)",
	"Maori All Blacks in Japan":          "International Friendly",
	"Portugal in Africa":                 "Summer Tests",
	"Portugal in Scotland":               "Autumn Nations Series",
	"Reds in Tonga":                      "International Friendly",
	"Romania in North America":           "Summer Tests",
	"Scotland in North & South America":  "Summer Tests",
	"South Africa in Europe":             "Autumn Nations Series",
	"South Africa in United Kingdom":     "Autumn Nations Series",
	"Spain in Pacific Islands":           "Summer Tests",
	"Switzerland in Europe":              "International Friendly",
	"Tonga in Europe":                    "International Friendly",
	"Uganda in Kenya":                    "International Friendly",
	"United States in Europe":            "International Friendly",
	"Uruguay in Europe":                  "International Friendly",
	"Wales in Australia":                 "Summer Test Series",
	"Wales in Scotland (W)":              "International Friendly (W)",
	"WXV Qualifiers (W)":                 "WXV (W)",
	"WXV Warm Up Games (W)":              "International Friendly (W)",
	"Zimbabwe in Asia":                   "International Friendly",
}

type LeagueTransition struct {
	SuccessorID string
	Year        int    // Year when the transition happened
	DisplayName string // What to show in the UI for matches before this year
}

var LeagueSuccessors = map[string]LeagueTransition{
	"Tri Nations": {
		SuccessorID: "WLD-THE-RUGBY-CHAMPIONSHIP",
		Year:        2012,
		DisplayName: "Tri Nations",
	},
	"November Internationals": {
		SuccessorID: "WLD-AUTUMN-NATIONS-SERIES",
		Year:        2020,
		DisplayName: "November Internationals",
	},
	// "Autumn Nations Series": {
	// 	SuccessorID: "World Nations League",
	// },
	// "Autumn Nations Cup": {
	// 	SuccessorID: "WLD-AUTUMN-NATIONS-LEAGUE",
	// },

}

// StandardizeLeagueName maps abbreviated/alternate names to their full database names
var LeagueNameStandardization = map[string]string{
	"JRLO - Division 1": "Japan Rugby League One - Division 1",
	"JRLO - Division 2": "Japan Rugby League One - Division 2",
	"JRLO - Division 3": "Japan Rugby League One - Division 3",
	"WXV 2024 (W)":      "WXV (W)",
	"World Rugby Pacific Nations Cup": "Pacific Nations Cup",
}

func CleanLeagueName(name string) string {
	// Remove year patterns
	name = regexp.MustCompile(`\s*\(\d{4}(?:-\d{2,4})?\)`).ReplaceAllString(name, "")

	// Standardize the name if it exists in the mapping
	if standardName, ok := LeagueNameStandardization[name]; ok {
		return standardName
	}

	return strings.TrimSpace(name)
}

// Competitions that can share matches with other competitions
var SharedMatchCompetitions = map[string]bool{
	"Bledisloe Cup":           true,
	"Laurie O'Reilly Cup (W)": true,
}
