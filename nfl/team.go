package nfl

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cputans/go-sports/internal"
)

type Team struct {
	Name                         string `fullSelector:"#meta > div:nth-child(2) > h1:nth-child(1) > span:nth-child(2)"`
	ShortName                    string
	PointsFor                    uint       `tableId:"team_stats" row:"0" cell:"0"`
	Yards                        uint       `tableId:"team_stats" row:"0" cell:"1"`
	OffensivePlays               uint       `tableId:"team_stats" row:"0" cell:"2"`
	YardsPerPlay                 float64    `tableId:"team_stats" row:"0" cell:"3"`
	Turnovers                    uint       `tableId:"team_stats" row:"0" cell:"4"`
	FumblesLost                  uint       `tableId:"team_stats" row:"0" cell:"5"`
	FirstDowns                   uint       `tableId:"team_stats" row:"0" cell:"6"`
	Completions                  uint       `tableId:"team_stats" row:"0" cell:"7"`
	PassingAttempts              uint       `tableId:"team_stats" row:"0" cell:"8"`
	PassingYards                 int        `tableId:"team_stats" row:"0" cell:"9"`
	PassingTouchdowns            uint       `tableId:"team_stats" row:"0" cell:"10"`
	Interceptions                uint       `tableId:"team_stats" row:"0" cell:"11"`
	PassingNetYardsPerAttempt    float64    `tableId:"team_stats" row:"0" cell:"12"`
	PassingFirstDowns            uint       `tableId:"team_stats" row:"0" cell:"13"`
	RushingAttempts              uint       `tableId:"team_stats" row:"0" cell:"14"`
	RushingYards                 int        `tableId:"team_stats" row:"0" cell:"15"`
	RushingTouchdowns            uint       `tableId:"team_stats" row:"0" cell:"16"`
	RushingYardsPerAttempt       float64    `tableId:"team_stats" row:"0" cell:"17"`
	RushingFirstDowns            uint       `tableId:"team_stats" row:"0" cell:"18"`
	Penalties                    uint       `tableId:"team_stats" row:"0" cell:"19"`
	PenaltyYards                 uint       `tableId:"team_stats" row:"0" cell:"20"`
	PenaltyFirstDowns            uint       `tableId:"team_stats" row:"0" cell:"21"`
	DriveCount                   uint       `tableId:"team_stats" row:"0" cell:"22"`
	DriveScoringPercentage       float64    `tableId:"team_stats" row:"0" cell:"23"`
	DriveTurnoverPercentage      float64    `tableId:"team_stats" row:"0" cell:"24"`
	DriveAverageStart            string     `tableId:"team_stats" row:"0" cell:"25"`
	DriveAverageTime             string     `tableId:"team_stats" row:"0" cell:"26"`
	DriveAveragePlays            uint       `tableId:"team_stats" row:"0" cell:"27"`
	DriveAverageYards            float64    `tableId:"team_stats" row:"0" cell:"28"`
	DriveAveragePoints           float64    `tableId:"team_stats" row:"0" cell:"29"`
	OppOppPointsFor              uint       `tableId:"team_stats" row:"1" cell:"0"`
	OppYards                     int        `tableId:"team_stats" row:"1" cell:"1"`
	OppOffensivePlays            uint       `tableId:"team_stats" row:"1" cell:"2"`
	OppYardsPerPlay              float64    `tableId:"team_stats" row:"1" cell:"3"`
	OppTurnovers                 uint       `tableId:"team_stats" row:"1" cell:"4"`
	OppFumblesLost               uint       `tableId:"team_stats" row:"1" cell:"5"`
	OppFirstDowns                uint       `tableId:"team_stats" row:"1" cell:"6"`
	OppCompletions               uint       `tableId:"team_stats" row:"1" cell:"7"`
	OppPassingAttempts           uint       `tableId:"team_stats" row:"1" cell:"8"`
	OppPassingYards              int        `tableId:"team_stats" row:"1" cell:"9"`
	OppPassingTouchdowns         uint       `tableId:"team_stats" row:"1" cell:"10"`
	OppInterceptions             uint       `tableId:"team_stats" row:"1" cell:"11"`
	OppPassingNetYardsPerAttempt float64    `tableId:"team_stats" row:"1" cell:"12"`
	OppPassingFirstDowns         uint       `tableId:"team_stats" row:"1" cell:"13"`
	OppRushingAttempts           uint       `tableId:"team_stats" row:"1" cell:"14"`
	OppRushingYards              int        `tableId:"team_stats" row:"1" cell:"15"`
	OppRushingTouchdowns         uint       `tableId:"team_stats" row:"1" cell:"16"`
	OppRushingYardsPerAttempt    float64    `tableId:"team_stats" row:"1" cell:"17"`
	OppRushingFirstDowns         uint       `tableId:"team_stats" row:"1" cell:"18"`
	OppPenalties                 uint       `tableId:"team_stats" row:"1" cell:"19"`
	OppPenaltyYards              uint       `tableId:"team_stats" row:"1" cell:"20"`
	OppPenaltyFirstDowns         uint       `tableId:"team_stats" row:"1" cell:"21"`
	OppDriveCount                uint       `tableId:"team_stats" row:"1" cell:"22"`
	OppDriveScoringPercentage    float64    `tableId:"team_stats" row:"1" cell:"23"`
	OppDriveTurnoverPercentage   float64    `tableId:"team_stats" row:"1" cell:"24"`
	OppDriveAverageStart         string     `tableId:"team_stats" row:"1" cell:"25"`
	OppDriveAverageTime          string     `tableId:"team_stats" row:"1" cell:"26"`
	OppDriveAveragePlays         uint       `tableId:"team_stats" row:"1" cell:"27"`
	OppDriveAverageYards         float64    `tableId:"team_stats" row:"1" cell:"28"`
	OppDriveAveragePoints        float64    `tableId:"team_stats" row:"1" cell:"29"`
	Games                        []TeamGame `tableId:"games" cellSelector:"td,th"`
	URL                          string
	Data                         []byte
}

type TeamGame struct {
	ID                string
	Week              string `tableId:"games" cell:"0"`
	Day               string `tableId:"games" cell:"1"`
	DateStr           string `tableId:"games" cell:"2"`
	TimeStr           string `tableId:"games" cell:"3"`
	Date              time.Time
	BoxscoreLink      string `tableId:"games" cell:"4" dataSelector:"a" attr:"href"`
	Result            string `tableId:"games" cell:"5"`
	Overtime          string `tableId:"games" cell:"6"`
	Location          string `tableId:"games" cell:"8"`
	Opponent          string `tableId:"games" cell:"9"`
	OpponentLink      string `tableId:"games" cell:"9" dataSelector:"a" attr:"href"`
	HomeTeam          string
	HomeTeamShortName string
	AwayTeam          string
	AwayTeamShortName string
}

func NewTeam(shortName string, year uint) *Team {
	url := fmt.Sprintf(TEAM_URL, strings.ToLower(shortName), year)

	return &Team{
		URL:       url,
		ShortName: shortName,
	}
}

func (t *Team) Parse() {
	internal.Parse(t, t.URL)
}

func (t *Team) PostProcess() {
	var newGames []TeamGame
	oppShortNameRegex := regexp.MustCompile("/teams/([A-Za-z]+)/[A-Za-z0-9.]+.htm")
	for _, g := range t.Games {
		if g.BoxscoreLink != "" {
			/* Get opponent short name */
			oppShortNameMatches := oppShortNameRegex.FindStringSubmatch(g.OpponentLink)

			/* ID */
			idRegex := regexp.MustCompile("/boxscores/([^.]+).htm")
			idMatches := idRegex.FindStringSubmatch(g.BoxscoreLink)
			if idMatches != nil {
				g.ID = idMatches[1]
			}

			if strings.Contains(g.Location, "@") {
				g.Location = "away"
				g.HomeTeam = g.Opponent
				g.HomeTeamShortName = oppShortNameMatches[1]
				g.AwayTeam = t.Name
				g.AwayTeamShortName = t.ShortName
			} else {
				g.Location = "home"
				g.HomeTeam = t.Name
				g.HomeTeamShortName = t.ShortName
				g.AwayTeam = g.Opponent
				g.AwayTeamShortName = oppShortNameMatches[1]
			}

			newGames = append(newGames, g)
		}

	}

	t.Games = newGames
}
