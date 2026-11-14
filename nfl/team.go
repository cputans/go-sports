package nfl

import (
	"fmt"
	"strings"
	"time"
)

type Team struct {
	Name      string
	ShortName string
	Year      uint

	// Team offense (regular season to date). Rates are as PFR reports them.
	PointsFor                 uint
	Yards                     uint
	OffensivePlays            uint
	YardsPerPlay              float64
	Turnovers                 uint
	FumblesLost               uint
	FirstDowns                uint
	Completions               uint
	PassingAttempts           uint
	PassingYards              int
	PassingTouchdowns         uint
	Interceptions             uint
	PassingNetYardsPerAttempt float64
	PassingFirstDowns         uint
	RushingAttempts           uint
	RushingYards              int
	RushingTouchdowns         uint
	RushingYardsPerAttempt    float64
	RushingFirstDowns         uint
	Penalties                 uint
	PenaltyYards              uint
	PenaltyFirstDowns         uint
	DriveCount                uint
	DriveScoringPercentage    float64
	DriveTurnoverPercentage   float64
	DriveAverageStart         string
	DriveAverageTime          string
	DriveAveragePlays         float64
	DriveAverageYards         float64
	DriveAveragePoints        float64

	// Opponent offense, i.e. what the team's defense allowed. OppOppPointsFor
	// keeps its original (misspelled) name for compatibility; PointsAgainst is
	// the same value.
	OppOppPointsFor              uint
	OppYards                     int
	OppOffensivePlays            uint
	OppYardsPerPlay              float64
	OppTurnovers                 uint
	OppFumblesLost               uint
	OppFirstDowns                uint
	OppCompletions               uint
	OppPassingAttempts           uint
	OppPassingYards              int
	OppPassingTouchdowns         uint
	OppInterceptions             uint
	OppPassingNetYardsPerAttempt float64
	OppPassingFirstDowns         uint
	OppRushingAttempts           uint
	OppRushingYards              int
	OppRushingTouchdowns         uint
	OppRushingYardsPerAttempt    float64
	OppRushingFirstDowns         uint
	OppPenalties                 uint
	OppPenaltyYards              uint
	OppPenaltyFirstDowns         uint
	OppDriveCount                uint
	OppDriveScoringPercentage    float64
	OppDriveTurnoverPercentage   float64
	OppDriveAverageStart         string
	OppDriveAverageTime          string
	OppDriveAveragePlays         float64
	OppDriveAverageYards         float64
	OppDriveAveragePoints        float64

	PointsAgainst uint
	Games         []TeamGame
	URL           string

	// Err is set by Parse when the team could not be loaded.
	Err error
}

type TeamGame struct {
	ID                string // PFR-style boxscore id, e.g. 202609130nyg
	Week              string
	Day               string
	DateStr           string
	Date              time.Time
	BoxscoreLink      string // e.g. /boxscores/202609130nyg.htm
	Result            string // "W", "L" or "T"; empty until the game is played
	Overtime          string
	Location          string // "home" or "away"
	Opponent          string
	OpponentLink      string
	HomeTeam          string
	HomeTeamShortName string
	AwayTeam          string
	AwayTeamShortName string
}

func NewTeam(shortName string, year uint) *Team {
	t := &Team{ShortName: shortName, Year: year}
	if info := findTeam(shortName); info != nil {
		t.Name = info.Name
		t.URL = fmt.Sprintf(TEAM_URL, strings.ToLower(info.PFR), year)
	}
	return t
}

// Parse builds the team's schedule from nflverse data.
func (t *Team) Parse() {
	info := findTeam(t.ShortName)
	if info == nil {
		t.Err = fmt.Errorf("unknown team %q", t.ShortName)
		return
	}

	data, err := loadSeason(int(t.Year))
	if err != nil {
		t.Err = err
		return
	}

	t.Name = info.Name
	t.Games = nil

	var ours, theirs sideTotals

	for _, g := range data.games {
		home, away := findTeam(g.Home), findTeam(g.Away)
		if g.PFRID == "" || home == nil || away == nil || (home != info && away != info) {
			continue
		}

		tg := TeamGame{
			ID:                g.PFRID,
			Week:              fmt.Sprint(g.Week),
			Day:               g.Date.Format("Mon"),
			DateStr:           g.Date.Format("2006-01-02"),
			Date:              g.Date,
			BoxscoreLink:      fmt.Sprintf("/boxscores/%s.htm", g.PFRID),
			HomeTeam:          home.Name,
			HomeTeamShortName: strings.ToLower(home.PFR),
			AwayTeam:          away.Name,
			AwayTeamShortName: strings.ToLower(away.PFR),
		}

		myScore, oppScore, opp := g.HomeScore, g.AwayScore, away
		tg.Location = "home"
		if away == info {
			myScore, oppScore, opp = g.AwayScore, g.HomeScore, home
			tg.Location = "away"
		}
		tg.Opponent = opp.Name
		tg.OpponentLink = fmt.Sprintf("/teams/%s/%d.htm", strings.ToLower(opp.PFR), t.Year)

		if g.Played {
			switch {
			case myScore > oppScore:
				tg.Result = "W"
			case myScore < oppScore:
				tg.Result = "L"
			default:
				tg.Result = "T"
			}
			if g.Overtime {
				tg.Overtime = "OT"
			}
			if g.Type == "REG" {
				ours.add(data.gameSide(g, info.Abbr))
				theirs.add(data.gameSide(g, opp.Abbr))
			}
		}

		t.Games = append(t.Games, tg)
	}

	t.fillStats(ours, theirs)
}

func (t *Team) fillStats(o, a sideTotals) {
	t.PointsFor = uint(o.Points)
	t.PointsAgainst = uint(a.Points)
	t.OppOppPointsFor = uint(a.Points)

	t.Yards = uint(max(o.yards(), 0))
	t.OffensivePlays = uint(o.plays())
	t.YardsPerPlay = o.yardsPerPlay()
	t.Turnovers = uint(o.turnovers())
	t.FumblesLost = uint(o.FumblesLost)
	t.FirstDowns = uint(o.firstDowns())
	t.Completions = uint(o.PassCompletions)
	t.PassingAttempts = uint(o.PassAttempts)
	t.PassingYards = int(o.PassYards)
	t.PassingTouchdowns = uint(o.PassTDs)
	t.Interceptions = uint(o.Interceptions)
	t.PassingNetYardsPerAttempt = o.netYardsPerAttempt()
	t.PassingFirstDowns = uint(o.FirstDownsPass)
	t.RushingAttempts = uint(o.RushAttempts)
	t.RushingYards = int(o.RushYards)
	t.RushingTouchdowns = uint(o.RushTDs)
	t.RushingYardsPerAttempt = o.rushYardsPerAttempt()
	t.RushingFirstDowns = uint(o.FirstDownsRush)
	t.Penalties = uint(o.Penalties)
	t.PenaltyYards = uint(o.PenaltyYards)
	t.PenaltyFirstDowns = uint(o.FirstDownsPenalty)
	t.DriveCount = uint(o.Drives)
	t.DriveScoringPercentage = o.driveScoringPct()
	t.DriveTurnoverPercentage = o.driveTurnoverPct()
	t.DriveAverageStart = o.driveAvgStart()
	t.DriveAverageTime = o.driveAvgTime()
	t.DriveAveragePlays = o.driveAvgPlays()
	t.DriveAverageYards = o.driveAvgYards()
	t.DriveAveragePoints = o.driveAvgPoints()

	t.OppYards = int(a.yards())
	t.OppOffensivePlays = uint(a.plays())
	t.OppYardsPerPlay = a.yardsPerPlay()
	t.OppTurnovers = uint(a.turnovers())
	t.OppFumblesLost = uint(a.FumblesLost)
	t.OppFirstDowns = uint(a.firstDowns())
	t.OppCompletions = uint(a.PassCompletions)
	t.OppPassingAttempts = uint(a.PassAttempts)
	t.OppPassingYards = int(a.PassYards)
	t.OppPassingTouchdowns = uint(a.PassTDs)
	t.OppInterceptions = uint(a.Interceptions)
	t.OppPassingNetYardsPerAttempt = a.netYardsPerAttempt()
	t.OppPassingFirstDowns = uint(a.FirstDownsPass)
	t.OppRushingAttempts = uint(a.RushAttempts)
	t.OppRushingYards = int(a.RushYards)
	t.OppRushingTouchdowns = uint(a.RushTDs)
	t.OppRushingYardsPerAttempt = a.rushYardsPerAttempt()
	t.OppRushingFirstDowns = uint(a.FirstDownsRush)
	t.OppPenalties = uint(a.Penalties)
	t.OppPenaltyYards = uint(a.PenaltyYards)
	t.OppPenaltyFirstDowns = uint(a.FirstDownsPenalty)
	t.OppDriveCount = uint(a.Drives)
	t.OppDriveScoringPercentage = a.driveScoringPct()
	t.OppDriveTurnoverPercentage = a.driveTurnoverPct()
	t.OppDriveAverageStart = a.driveAvgStart()
	t.OppDriveAverageTime = a.driveAvgTime()
	t.OppDriveAveragePlays = a.driveAvgPlays()
	t.OppDriveAverageYards = a.driveAvgYards()
	t.OppDriveAveragePoints = a.driveAvgPoints()
}
