package nba

import (
	"fmt"
	"time"
)

// TeamStats are a team's totals for one game.
type TeamStats struct {
	Points             int
	FieldGoalsMade     int
	FieldGoalsAttempts int
	ThreePointersMade  int
	ThreePointAttempts int
	FreeThrowsMade     int
	FreeThrowAttempts  int
	OffensiveRebounds  int
	DefensiveRebounds  int
	Rebounds           int
	Assists            int
	Steals             int
	Blocks             int
	Turnovers          int
	Fouls              int
	PointsInPaint      int
	FastBreakPoints    int
	LargestLead        int
}

// BoxscorePlayer is one player's line for a game. Players on the roster who
// did not play are included with DidNotPlay set.
type BoxscorePlayer struct {
	ID            string
	Name          string
	TeamShortName string
	Position      string
	Starter       bool
	DidNotPlay    bool
	DNPReason     string
	Minutes       float64
	PlusMinus     int

	Points             int
	FieldGoalsMade     int
	FieldGoalsAttempts int
	ThreePointersMade  int
	ThreePointAttempts int
	FreeThrowsMade     int
	FreeThrowAttempts  int
	OffensiveRebounds  int
	DefensiveRebounds  int
	Rebounds           int
	Assists            int
	Steals             int
	Blocks             int
	Turnovers          int
	Fouls              int
}

type Boxscore struct {
	ID         string
	Year       uint
	SeasonType string // "preseason", "regular", "postseason" or "play-in"
	Date       time.Time

	HomeTeam          string
	HomeTeamShortName string
	HomeTeamScore     uint
	HomeTeamStats     TeamStats
	AwayTeam          string
	AwayTeamShortName string
	AwayTeamScore     uint
	AwayTeamStats     TeamStats

	Players []BoxscorePlayer

	// Err is set by Parse when the boxscore could not be built. The other
	// fields must not be used when it is non-nil.
	Err error
}

// NewBoxscore creates a boxscore for an ESPN game ID (as found in
// TeamGame.ID) in the given season.
func NewBoxscore(id string, year uint) *Boxscore {
	return &Boxscore{ID: id, Year: year}
}

// Parse builds the boxscore from the published data. It sets Err with
// ErrGameNotFound if the game isn't in the schedule, and ErrStatsNotAvailable
// if the game is final but its box score hasn't been published yet.
func (b *Boxscore) Parse() {
	data, err := loadSeason(int(b.Year))
	if err != nil {
		b.Err = err
		return
	}

	g, ok := data.byID[b.ID]
	if !ok {
		b.Err = fmt.Errorf("%s: %w", b.ID, ErrGameNotFound)
		return
	}

	teams := data.teams[b.ID]
	home, away := teams[g.Home.Abbr], teams[g.Away.Abbr]
	if g.Played && (home == nil || away == nil || len(data.players[b.ID]) == 0) {
		b.Err = fmt.Errorf("%s: %w", b.ID, ErrStatsNotAvailable)
		return
	}

	b.SeasonType = seasonTypeName(g.SeasonType)
	b.Date = g.Date
	b.HomeTeam, b.HomeTeamShortName, b.HomeTeamScore = g.Home.Name, g.Home.Abbr, uint(g.Home.Score)
	b.AwayTeam, b.AwayTeamShortName, b.AwayTeamScore = g.Away.Name, g.Away.Abbr, uint(g.Away.Score)
	if home != nil {
		b.HomeTeamStats = home.Stats
	}
	if away != nil {
		b.AwayTeamStats = away.Stats
	}
	b.Players = data.players[b.ID]
}
