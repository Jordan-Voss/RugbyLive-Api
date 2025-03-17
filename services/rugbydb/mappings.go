package rugbydb

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
}

var LeagueAltNames = map[string][]string{
	"United Rugby Championship": {
		"URC",
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
	"European Rugby Champions Cup": {
		"Champions Cup",
		"Heineken Champions Cup",
	},
	"Heineken Cup": {
		"European Cup",
	},
	"European Challenge Cup": {
		"EPCR Challenge Cup",
		"Challenge Cup",
		"Investec Rugby Challenge Cup",
	},
}

type LeagueInfo struct {
	Country   string
	Countries []string
}

var LeagueCountryMap = map[string]LeagueInfo{
	"Super Rugby Pacific": {
		Country:   "OCE",
		Countries: []string{"AU", "NZL", "FJI", "SAM"},
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

	"Super Rugby Aupiki (W)":       1,
	"Six Nations Championship (W)": 1,
	"ProD2":                        2,
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
		Format: "League",
		Phases: []string{"League"},
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
	"Summer Test": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
	"Autumn Test": {
		Format: "Friendly",
		Phases: []string{"Friendly"},
	},
}
