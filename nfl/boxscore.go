package nfl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cputans/go-sports/internal"
)

type Boxscore struct {
	ID                 string
	DateStr            string `fullSelector:"div.scorebox_meta div:nth-child(1)"`
	TimeStr            string `fullSelector:".scorebox_meta div:nth-child(2)"`
	Date               time.Time
	WonToss            string `tableId:"game_info" row:"1" cell:"0"`
	Roof               string `tableId:"game_info" row:"2" cell:"0"`
	Surface            string `tableId:"game_info" row:"3" cell:"0"`
	Duration           string `tableId:"game_info" row:"4" cell:"0"`
	Attendance         string `tableId:"game_info" row:"5" cell:"0"`
	Weather            string `fullSelector:"table#game_info tbody tr th:contains('Weather') ~ td"`
	VegasLine          string `fullSelector:"table#game_info tbody tr th:contains('Vegas Line') ~ td"`
	OverUnder          string `fullSelector:"table#game_info tbody tr th:contains('Over/Under') ~ td"`
	HomeTeam           string `fullSelector:"div.scorebox div:nth-child(2) div:nth-child(1) strong:nth-child(2) a:nth-child(1)"`
	HomeTeamLink       string `fullSelector:"div.scorebox > div:nth-child(2) > div:nth-child(1) > strong:nth-child(2) > a" attr:"href"`
	HomeTeamShortName  string
	HomeTeamScore      uint              `fullSelector:".scorebox div:nth-child(2) div:nth-child(2) div:nth-child(1)"`
	HomeTeamStats      BoxscoreTeamStats `tableId:"team_stats" cell:"1"`
	AwayTeam           string            `fullSelector:"div.scorebox div:nth-child(1) div:nth-child(1) strong:nth-child(2) a:nth-child(1)"`
	AwayTeamLink       string            `fullSelector:"div.scorebox > div:nth-child(1) > div:nth-child(1) > strong:nth-child(2) > a" attr:"href"`
	AwayTeamShortName  string
	AwayTeamScore      uint                           `fullSelector:"div.scorebox div:nth-child(1) div:nth-child(2) div:nth-child(1)"`
	AwayTeamStats      BoxscoreTeamStats              `tableId:"team_stats" cell:"0"`
	PlayerStatsOffense []BoxscorePlayerOffensiveStats `tableId:"player_offense" rowSelector:"tr:not(.thead)" cellSelector:"td,th"`
	PlayerStatsDefense []BoxscorePlayerDefensiveStats `tableId:"player_defense" rowSelector:"tr:not(.thead)" cellSelector:"td,th"`
	URL                string
}

type BoxscoreTeamStats struct {
	FirstDowns           uint   `row:"0"`
	RushingInfo          string `row:"1"`
	RushingAttempts      uint
	RushingYards         int
	RushingTouchdowns    uint
	PassingInfo          string `row:"2"`
	PassingCompletions   uint
	PassingAttempts      uint
	PassingYards         int
	PassingTouchdowns    uint
	PassingInterceptions uint
}

type BoxscorePlayerOffensiveStats struct {
	Name                   string `cell:"0"`
	ID                     string
	TeamShortName          string  `cell:"1"`
	PassingCompletions     uint    `cell:"2"`
	PassingAttempts        uint    `cell:"3"`
	PassingYards           int     `cell:"4"`
	PassingTouchdowns      uint    `cell:"5"`
	PassingInterceptions   uint    `cell:"6"`
	PassingSacked          uint    `cell:"7"`
	PassingSackedLostYards uint    `cell:"8"`
	PassingLong            uint    `cell:"9"`
	PassingRating          float64 `cell:"10"`
	RushingAttempts        uint    `cell:"11"`
	RushingYards           int     `cell:"12"`
	RushingTouchdowns      uint    `cell:"13"`
	RushingLong            uint    `cell:"14"`
	ReceivingTargets       uint    `cell:"15"`
	ReceivingReceptions    uint    `cell:"16"`
	ReceivingYards         int     `cell:"17"`
	ReceivingTouchdowns    uint    `cell:"18"`
	ReceivingLong          uint    `cell:"19"`
	Fumbles                uint    `cell:"20"`
	FumblesLost            uint    `cell:"21"`
	URL                    string  `cell:"0" dataSelector:"a" attr:"href"`
}

type BoxscorePlayerDefensiveStats struct {
	Name                   string `cell:"0"`
	ID                     string
	TeamShortName          string  `cell:"1"`
	Interceptions          uint    `cell:"2"`
	InterceptionYards      int     `cell:"3"`
	InterceptionTouchdowns uint    `cell:"4"`
	PassDeflections        uint    `cell:"6"`
	Sacks                  float64 `cell:"7"`
	TacklesCombined        uint    `cell:"8"`
	TacklesSolo            uint    `cell:"9"`
	TacklesAssists         uint    `cell:"10"`
	TacklesForLoss         uint    `cell:"11"`
	QuarterbackHits        uint    `cell:"12"`
	FumblesRecovered       uint    `cell:"13"`
	URL                    string  `cell:"0" dataSelector:"a" attr:"href"`
}

func NewBoxscore(name string) *Boxscore {
	url := fmt.Sprintf(BASE_URL, strings.ToLower(name))

	return &Boxscore{
		URL: url,
	}
}

func (b *Boxscore) Parse() {
	internal.Parse(b, b.URL)
}

func (g *Boxscore) PostProcess() {
	/* Convert date strings to time.Time */
	timeString := strings.Replace(g.TimeStr, "Start Time: ", "", -1)
	dateString := fmt.Sprintf("%s %s", g.DateStr, timeString)
	date, _ := time.Parse("Monday Jan 2, 2006 3:04pm", dateString)
	g.Date = date

	/* ID */
	idRegex := regexp.MustCompile("/boxscores/([^.]+).htm")
	idMatches := idRegex.FindStringSubmatch(g.URL)
	if idMatches != nil {
		g.ID = idMatches[1]
	}

	/* Parse team stats */
	g.parseRushingStats(&g.AwayTeamStats)
	g.parseRushingStats(&g.HomeTeamStats)
	g.parsePassingStats(&g.AwayTeamStats)
	g.parsePassingStats(&g.HomeTeamStats)

	/* Set player IDs */
	playerIdRegex := regexp.MustCompile("players/[A-Za-z0-9]+/([A-Za-z0-9.]+).htm")
	for i, p := range g.PlayerStatsOffense {
		idMatches := playerIdRegex.FindStringSubmatch(p.URL)
		if idMatches != nil {
			player := g.PlayerStatsOffense[i]
			player.ID = idMatches[1]

			/* Set team abbr */
			if val, ok := TEAM_ABBR_MAP[player.TeamShortName]; ok {
				player.TeamShortName = val
			}

			g.PlayerStatsOffense[i] = player
		}
	}

	for i, p := range g.PlayerStatsDefense {
		idMatches := playerIdRegex.FindStringSubmatch(p.URL)
		if idMatches != nil {
			player := g.PlayerStatsDefense[i]
			player.ID = idMatches[1]

			/* Set team abbr */
			if val, ok := TEAM_ABBR_MAP[player.TeamShortName]; ok {
				player.TeamShortName = val
			}

			g.PlayerStatsDefense[i] = player
		}

	}

	/* Get short names */
	shortNameRegex := regexp.MustCompile("/teams/([A-Za-z]+)/[A-Za-z0-9.]+.htm")
	homeMatches := shortNameRegex.FindStringSubmatch(g.HomeTeamLink)
	if homeMatches != nil {
		g.HomeTeamShortName = strings.ToUpper(homeMatches[1])
	}

	awayMatches := shortNameRegex.FindStringSubmatch(g.AwayTeamLink)
	if awayMatches != nil {
		g.AwayTeamShortName = strings.ToUpper(awayMatches[1])
	}
}

func (g *Boxscore) parseRushingStats(s *BoxscoreTeamStats) {
	rushing := strings.Split(s.RushingInfo, "-")

	if len(rushing) >= 3 {
		rushingAttempts, _ := strconv.ParseUint(rushing[0], 10, 64)
		s.RushingAttempts = uint(rushingAttempts)

		rushingYards, _ := strconv.ParseInt(rushing[1], 10, 64)
		s.RushingYards = int(rushingYards)

		rushingTouchdowns, _ := strconv.ParseUint(rushing[2], 10, 64)
		s.RushingTouchdowns = uint(rushingTouchdowns)
	}
}

func (g *Boxscore) parsePassingStats(s *BoxscoreTeamStats) {
	passing := strings.Split(s.PassingInfo, "-")

	if len(passing) >= 5 {
		passingComp, _ := strconv.ParseUint(passing[0], 10, 64)
		s.PassingCompletions = uint(passingComp)

		passingAtt, _ := strconv.ParseUint(passing[1], 10, 64)
		s.PassingAttempts = uint(passingAtt)

		passingYards, _ := strconv.ParseInt(passing[2], 10, 64)
		s.PassingYards = int(passingYards)

		passingTDs, _ := strconv.ParseUint(passing[3], 10, 64)
		s.PassingTouchdowns = uint(passingTDs)

		passingInts, _ := strconv.ParseUint(passing[4], 10, 64)
		s.PassingInterceptions = uint(passingInts)
	}
}
