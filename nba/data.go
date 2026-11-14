package nba

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cputans/go-sports/v2/internal/dataset"
)

// Data in this package comes from the sportsdataverse project (the data behind
// hoopR, https://github.com/sportsdataverse/hoopR-nba-data), which publishes
// ESPN-derived NBA data as static files on GitHub releases (CC BY 4.0). A season
// is downloaded once per process.
//
// Seasons are keyed by the year they END in: season 2027 is 2026-27.

var (
	// ErrStatsNotAvailable is returned when a game has been played (the
	// schedule says it is final) but its box score hasn't been published yet.
	ErrStatsNotAvailable = dataset.ErrStatsNotAvailable
	// ErrGameNotFound is returned when a game ID is not in the season schedule.
	ErrGameNotFound = dataset.ErrGameNotFound
)

// seasons caches each season's data for the life of the process.
var seasons dataset.Cache[seasonData]

func loadSeason(year int) (*seasonData, error) { return seasons.Load(year, downloadSeason) }

const dataBase = "https://github.com/sportsdataverse/sportsdataverse-data/releases/download"

func scheduleURL(year int) string {
	return fmt.Sprintf("%s/espn_nba_schedules/nba_schedule_%d.csv", dataBase, year)
}
func teamBoxURL(year int) string {
	return fmt.Sprintf("%s/espn_nba_team_boxscores/team_box_%d.csv", dataBase, year)
}
func playerBoxURL(year int) string {
	return fmt.Sprintf("%s/espn_nba_player_boxscores/player_box_%d.csv", dataBase, year)
}

// Season types as published by ESPN.
const (
	seasonPreseason  = "1"
	seasonRegular    = "2"
	seasonPostseason = "3"
	seasonPlayIn     = "5"
)

func seasonTypeName(t string) string {
	switch t {
	case seasonPreseason:
		return "preseason"
	case seasonRegular:
		return "regular"
	case seasonPostseason:
		return "postseason"
	case seasonPlayIn:
		return "play-in"
	}
	return t
}

// gameRow is a row of the schedule.
type gameRow struct {
	ID         string
	Season     int
	SeasonType string
	Date       time.Time
	Home, Away teamRef
	Played     bool
}

type teamRef struct {
	ID    string
	Abbr  string // ESPN abbreviation, e.g. "GS", "NY", "UTAH"
	Name  string // e.g. "Golden State Warriors"
	Score int
}

// teamRow is one team's side of a box score.
type teamRow struct {
	Abbr  string
	Stats TeamStats
}

// seasonData is everything loaded for one season.
type seasonData struct {
	year    int
	games   []*gameRow
	byID    map[string]*gameRow
	teams   map[string]map[string]*teamRow // game id -> team abbr
	players map[string][]BoxscorePlayer    // game id
}

func downloadSeason(year int) (*seasonData, error) {
	d := &seasonData{
		year:    year,
		byID:    map[string]*gameRow{},
		teams:   map[string]map[string]*teamRow{},
		players: map[string][]BoxscorePlayer{},
	}

	if err := d.loadSchedule(); err != nil {
		return nil, fmt.Errorf("loading schedule: %w", err)
	}
	if len(d.games) == 0 {
		return nil, fmt.Errorf("no games in schedule for season %d", year)
	}

	// Box scores 404 until the first games are published; treat that as
	// "no data yet" rather than a failure.
	for name, load := range map[string]func() error{
		"team box scores":   d.loadTeamBox,
		"player box scores": d.loadPlayerBox,
	} {
		if err := load(); err != nil && !errors.Is(err, dataset.ErrNotFound) {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
	}

	return d, nil
}

// dateLayout is how ESPN timestamps appear in the box score and schedule files,
// e.g. 2026-06-13T20:30:00.000000-0400.
const dateLayout = "2006-01-02T15:04:05.999999-0700"

func (d *seasonData) loadSchedule() error {
	t, err := dataset.OpenCSV(scheduleURL(d.year))
	if err != nil {
		return err
	}
	defer t.Close()

	for {
		ok, err := t.Next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		// All-star games are published with the regular season.
		if strings.EqualFold(t.Str("type_abbreviation"), "ALLSTAR") {
			continue
		}

		// Placeholder games whose teams aren't decided yet (e.g. NBA Cup
		// knockout rounds) are published with a "TBD" team.
		if t.Str("home_abbreviation") == "TBD" || t.Str("away_abbreviation") == "TBD" {
			continue
		}

		g := &gameRow{
			ID:         t.Str("game_id"),
			Season:     d.year,
			SeasonType: t.Str("season_type"),
			Home: teamRef{
				ID: t.Str("home_id"), Abbr: t.Str("home_abbreviation"),
				Name: t.Str("home_display_name"), Score: t.Int("home_score"),
			},
			Away: teamRef{
				ID: t.Str("away_id"), Abbr: t.Str("away_abbreviation"),
				Name: t.Str("away_display_name"), Score: t.Int("away_score"),
			},
			Played: t.Bool("status_type_completed"),
		}
		if g.ID == "" {
			g.ID = t.Str("id")
		}

		g.Date, _ = time.Parse(dateLayout, t.Str("game_date_time"))

		d.games = append(d.games, g)
		d.byID[g.ID] = g
	}
}

func (d *seasonData) loadTeamBox() error {
	t, err := dataset.OpenCSV(teamBoxURL(d.year))
	if err != nil {
		return err
	}
	defer t.Close()

	for {
		ok, err := t.Next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		id := t.Str("game_id")
		if _, ok := d.byID[id]; !ok {
			continue
		}

		if d.teams[id] == nil {
			d.teams[id] = map[string]*teamRow{}
		}
		abbr := t.Str("team_abbreviation")
		d.teams[id][abbr] = &teamRow{
			Abbr: abbr,
			Stats: TeamStats{
				Points:             t.Int("team_score"),
				FieldGoalsMade:     t.Int("field_goals_made"),
				FieldGoalsAttempts: t.Int("field_goals_attempted"),
				ThreePointersMade:  t.Int("three_point_field_goals_made"),
				ThreePointAttempts: t.Int("three_point_field_goals_attempted"),
				FreeThrowsMade:     t.Int("free_throws_made"),
				FreeThrowAttempts:  t.Int("free_throws_attempted"),
				OffensiveRebounds:  t.Int("offensive_rebounds"),
				DefensiveRebounds:  t.Int("defensive_rebounds"),
				Rebounds:           t.Int("total_rebounds"),
				Assists:            t.Int("assists"),
				Steals:             t.Int("steals"),
				Blocks:             t.Int("blocks"),
				Turnovers:          t.Int("turnovers"),
				Fouls:              t.Int("fouls"),
				PointsInPaint:      t.Int("points_in_paint"),
				FastBreakPoints:    t.Int("fast_break_points"),
				LargestLead:        t.Int("largest_lead"),
			},
		}
	}
}

func (d *seasonData) loadPlayerBox() error {
	t, err := dataset.OpenCSV(playerBoxURL(d.year))
	if err != nil {
		return err
	}
	defer t.Close()

	for {
		ok, err := t.Next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		id := t.Str("game_id")
		if _, ok := d.byID[id]; !ok {
			continue
		}

		d.players[id] = append(d.players[id], BoxscorePlayer{
			ID:                 t.Str("athlete_id"),
			Name:               t.Str("athlete_display_name"),
			TeamShortName:      t.Str("team_abbreviation"),
			Position:           t.Str("athlete_position_abbreviation"),
			Starter:            t.Bool("starter"),
			DidNotPlay:         t.Bool("did_not_play"),
			DNPReason:          dnpReason(t),
			Minutes:            t.Num("minutes"),
			Points:             t.Int("points"),
			FieldGoalsMade:     t.Int("field_goals_made"),
			FieldGoalsAttempts: t.Int("field_goals_attempted"),
			ThreePointersMade:  t.Int("three_point_field_goals_made"),
			ThreePointAttempts: t.Int("three_point_field_goals_attempted"),
			FreeThrowsMade:     t.Int("free_throws_made"),
			FreeThrowAttempts:  t.Int("free_throws_attempted"),
			OffensiveRebounds:  t.Int("offensive_rebounds"),
			DefensiveRebounds:  t.Int("defensive_rebounds"),
			Rebounds:           t.Int("rebounds"),
			Assists:            t.Int("assists"),
			Steals:             t.Int("steals"),
			Blocks:             t.Int("blocks"),
			Turnovers:          t.Int("turnovers"),
			Fouls:              t.Int("fouls"),
			PlusMinus:          t.Int("plus_minus"),
		})
	}
}

// dnpReason is only meaningful for players who didn't play; ESPN fills it in
// for some who did.
func dnpReason(t *dataset.Table) string {
	if t.Bool("did_not_play") {
		return t.Str("reason")
	}
	return ""
}
