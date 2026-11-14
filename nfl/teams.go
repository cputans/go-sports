package nfl

import "strings"

type teamInfo struct {
	Abbr       string // nflverse abbreviation
	PFR        string // pro-football-reference abbreviation (upper case)
	Name       string
	Conference string
}

// nflTeams lists the current 32 franchises. PFR abbreviations are kept because
// downstream data (documents already indexed) is keyed on them.
var nflTeams = []teamInfo{
	{"ARI", "CRD", "Arizona Cardinals", "NFC"},
	{"ATL", "ATL", "Atlanta Falcons", "NFC"},
	{"BAL", "RAV", "Baltimore Ravens", "AFC"},
	{"BUF", "BUF", "Buffalo Bills", "AFC"},
	{"CAR", "CAR", "Carolina Panthers", "NFC"},
	{"CHI", "CHI", "Chicago Bears", "NFC"},
	{"CIN", "CIN", "Cincinnati Bengals", "AFC"},
	{"CLE", "CLE", "Cleveland Browns", "AFC"},
	{"DAL", "DAL", "Dallas Cowboys", "NFC"},
	{"DEN", "DEN", "Denver Broncos", "AFC"},
	{"DET", "DET", "Detroit Lions", "NFC"},
	{"GB", "GNB", "Green Bay Packers", "NFC"},
	{"HOU", "HTX", "Houston Texans", "AFC"},
	{"IND", "CLT", "Indianapolis Colts", "AFC"},
	{"JAX", "JAX", "Jacksonville Jaguars", "AFC"},
	{"KC", "KAN", "Kansas City Chiefs", "AFC"},
	{"LA", "RAM", "Los Angeles Rams", "NFC"},
	{"LAC", "SDG", "Los Angeles Chargers", "AFC"},
	{"LV", "RAI", "Las Vegas Raiders", "AFC"},
	{"MIA", "MIA", "Miami Dolphins", "AFC"},
	{"MIN", "MIN", "Minnesota Vikings", "NFC"},
	{"NE", "NWE", "New England Patriots", "AFC"},
	{"NO", "NOR", "New Orleans Saints", "NFC"},
	{"NYG", "NYG", "New York Giants", "NFC"},
	{"NYJ", "NYJ", "New York Jets", "AFC"},
	{"PHI", "PHI", "Philadelphia Eagles", "NFC"},
	{"PIT", "PIT", "Pittsburgh Steelers", "AFC"},
	{"SEA", "SEA", "Seattle Seahawks", "NFC"},
	{"SF", "SFO", "San Francisco 49ers", "NFC"},
	{"TB", "TAM", "Tampa Bay Buccaneers", "NFC"},
	{"TEN", "OTI", "Tennessee Titans", "AFC"},
	{"WAS", "WAS", "Washington Commanders", "NFC"},
}

// legacyAbbr maps older nflverse abbreviations onto current ones.
var legacyAbbr = map[string]string{
	"LAR": "LA", "STL": "LA", "SL": "LA",
	"OAK": "LV",
	"SD":  "LAC",
	"ARZ": "ARI", "BLT": "BAL", "CLV": "CLE", "HST": "HOU",
	"WSH": "WAS",
}

var teamLookup = func() map[string]*teamInfo {
	m := map[string]*teamInfo{}
	for i := range nflTeams {
		t := &nflTeams[i]
		m[t.Abbr] = t
		m[t.PFR] = t
	}
	return m
}()

// findTeam resolves any nflverse or PFR abbreviation, in any case.
func findTeam(abbr string) *teamInfo {
	abbr = strings.ToUpper(abbr)
	if l, ok := legacyAbbr[abbr]; ok {
		abbr = l
	}
	return teamLookup[abbr]
}

// pfrAbbr returns the PFR abbreviation for an nflverse abbreviation.
func pfrAbbr(abbr string) string {
	if t := findTeam(abbr); t != nil {
		return t.PFR
	}
	return strings.ToUpper(abbr)
}

func teamName(abbr string) string {
	if t := findTeam(abbr); t != nil {
		return t.Name
	}
	return abbr
}
