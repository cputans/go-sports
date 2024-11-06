package nfl

var (
	BASE_URL      = "https://www.pro-football-reference.com%s"
	TEAM_URL      = "https://www.pro-football-reference.com/teams/%s/%d.htm"
	SEASON_URL    = "https://www.pro-football-reference.com/years/%d/"
	SCHEDULE_URL  = "https://www.pro-football-reference.com/teams/%s/%s/gamelog/"
	BOXSCORE_URL  = "https://www.pro-football-reference.com/boxscores/%s.htm"
	BOXSCORES_URL = "https://www.pro-football-reference.com/years/%s/week_%s.htm"
	PLAYER_URL    = "https://www.pro-football-reference.com/players/%s/%s.htm"
	ROSTER_URL    = "https://www.pro-football-reference.com/teams/%s/%s_roster.htm"
)

var TEAM_ABBR_MAP = map[string]string{
	"ARI": "CRD",
	"BAL": "RAV",
	"HOU": "HTX",
	"IND": "CLT",
	"LVR": "RAI",
	"LAC": "SDG",
	"LAR": "RAM",
	"TEN": "OTI",
}
