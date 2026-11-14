package nfl

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/cputans/go-sports/v2/internal/dataset"
)

// Data in this package comes from the nflverse project
// (https://github.com/nflverse/nflverse-data), which publishes NFL data as
// static files on GitHub releases. A season is downloaded once per process.

var (
	// ErrStatsNotAvailable is returned when a game has been played (the
	// schedule has a score) but nflverse hasn't published its player stats yet.
	ErrStatsNotAvailable = dataset.ErrStatsNotAvailable
	// ErrGameNotFound is returned when a game ID is not in the season schedule.
	ErrGameNotFound = dataset.ErrGameNotFound
)

// seasons caches each season's data for the life of the process.
var seasons dataset.Cache[seasonData]

func loadSeason(year int) (*seasonData, error) { return seasons.Load(year, downloadSeason) }

const nflverseBase = "https://github.com/nflverse/nflverse-data/releases/download"

func scheduleURL() string { return nflverseBase + "/schedules/games.csv" }
func playersURL() string  { return nflverseBase + "/players/players.csv" }
func statsURL(year int) string {
	return fmt.Sprintf("%s/stats_player/stats_player_week_%d.csv", nflverseBase, year)
}
func snapsURL(year int) string {
	return fmt.Sprintf("%s/snap_counts/snap_counts_%d.csv", nflverseBase, year)
}
func pbpURL(year int) string {
	return fmt.Sprintf("%s/pbp/play_by_play_%d.csv.gz", nflverseBase, year)
}

// gameRow is a row of the nflverse schedule.
type gameRow struct {
	ID        string // nflverse game id, e.g. 2026_01_PHI_NYG
	PFRID     string // pro-football-reference id, e.g. 202609130nyg
	Season    int
	Type      string
	Week      int
	Date      time.Time
	Away      string // nflverse abbreviations
	Home      string
	AwayScore int
	HomeScore int
	Played    bool
	Overtime  bool
	Spread    float64 // positive = home favored
	HasSpread bool
	Total     float64
	HasTotal  bool
	Roof      string
	Surface   string
	Temp      string
	Wind      string
}

type statRow struct {
	GSIS     string
	Name     string
	Position string
	Team     string // nflverse abbreviation

	PassCompletions, PassAttempts, PassYards, PassTDs, PassInts float64
	Sacked, SackedYards                                         float64
	RushAttempts, RushYards, RushTDs                            float64
	Targets, Receptions, RecYards, RecTDs                       float64
	Fumbles, FumblesLost                                        float64
	Penalties, PenaltyYards                                     float64

	DefInts, DefIntYards, DefIntTDs, PassDefended float64
	Sacks, TacklesSolo, TackleAssists, TFL        float64
	QBHits, FumblesRecovered, FumblesForced       float64
}

type snapRow struct {
	PFRID    string
	Name     string
	Position string
	Team     string
	OffSnaps float64
	OffPct   float64
	DefSnaps float64
	DefPct   float64
	STSnaps  float64
	STPct    float64
}

type longest struct {
	Pass, Rush, Receive int
}

// seasonData is everything loaded for one season.
type seasonData struct {
	year   int
	games  []*gameRow
	byPFR  map[string]*gameRow
	stats  map[string][]statRow // by nflverse game id
	snaps  map[string][]snapRow
	longs  map[string]map[string]*longest // game id -> gsis id -> longest plays
	pfrIDs map[string]string              // gsis id -> pfr id
	pbp    map[string]map[string]*pbpTeam // game id -> PFR team abbr -> play-by-play totals
}

func downloadSeason(year int) (*seasonData, error) {
	d := &seasonData{
		year:   year,
		byPFR:  map[string]*gameRow{},
		stats:  map[string][]statRow{},
		snaps:  map[string][]snapRow{},
		longs:  map[string]map[string]*longest{},
		pfrIDs: map[string]string{},
		pbp:    map[string]map[string]*pbpTeam{},
	}

	if err := d.loadSchedule(); err != nil {
		return nil, fmt.Errorf("loading schedule: %w", err)
	}
	if len(d.games) == 0 {
		return nil, fmt.Errorf("no games in schedule for season %d", year)
	}

	// Stats, snaps and play-by-play 404 until the first games are published;
	// treat that as "no data yet" rather than a failure.
	for name, load := range map[string]func() error{
		"players":      d.loadPlayers,
		"stats":        d.loadStats,
		"snap counts":  d.loadSnaps,
		"play-by-play": d.loadPBP,
	} {
		if err := load(); err != nil && !errors.Is(err, dataset.ErrNotFound) {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
	}

	return d, nil
}

func (d *seasonData) loadSchedule() error {
	t, err := dataset.OpenCSV(scheduleURL())
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

		if t.Str("season") != strconv.Itoa(d.year) {
			continue
		}

		g := &gameRow{
			ID:       t.Str("game_id"),
			PFRID:    t.Str("pfr"),
			Season:   d.year,
			Type:     t.Str("game_type"),
			Week:     int(t.Num("week")),
			Away:     t.Str("away_team"),
			Home:     t.Str("home_team"),
			Overtime: t.Str("overtime") == "1",
			Roof:     t.Str("roof"),
			Surface:  t.Str("surface"),
			Temp:     t.Str("temp"),
			Wind:     t.Str("wind"),
		}

		clock := t.Str("gametime")
		if clock == "" {
			clock = "00:00"
		}
		g.Date, _ = time.Parse("2006-01-02 15:04", t.Str("gameday")+" "+clock)

		if t.Str("home_score") != "" && t.Str("away_score") != "" {
			g.Played = true
			g.HomeScore = int(t.Num("home_score"))
			g.AwayScore = int(t.Num("away_score"))
		}
		if v := t.Str("spread_line"); v != "" {
			g.Spread, g.HasSpread = t.Num("spread_line"), true
		}
		if v := t.Str("total_line"); v != "" {
			g.Total, g.HasTotal = t.Num("total_line"), true
		}

		d.games = append(d.games, g)
		if g.PFRID != "" {
			d.byPFR[g.PFRID] = g
		}
	}
}

func (d *seasonData) loadPlayers() error {
	t, err := dataset.OpenCSV(playersURL())
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
		if g, p := t.Str("gsis_id"), t.Str("pfr_id"); g != "" && p != "" {
			d.pfrIDs[g] = p
		}
	}
}

func (d *seasonData) loadStats() error {
	t, err := dataset.OpenCSV(statsURL(d.year))
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

		name := t.Str("player_display_name")
		if name == "" {
			name = t.Str("player_name")
		}

		r := statRow{
			GSIS:     t.Str("player_id"),
			Name:     name,
			Position: t.Str("position"),
			Team:     t.Str("team"),

			PassCompletions: t.Num("completions"),
			PassAttempts:    t.Num("attempts"),
			PassYards:       t.Num("passing_yards"),
			PassTDs:         t.Num("passing_tds"),
			PassInts:        t.Num("passing_interceptions"),
			Sacked:          t.Num("sacks_suffered"),
			SackedYards:     math.Abs(t.Num("sack_yards_lost")),

			RushAttempts: t.Num("carries"),
			RushYards:    t.Num("rushing_yards"),
			RushTDs:      t.Num("rushing_tds"),

			Targets:    t.Num("targets"),
			Receptions: t.Num("receptions"),
			RecYards:   t.Num("receiving_yards"),
			RecTDs:     t.Num("receiving_tds"),

			Fumbles:      t.Num("rushing_fumbles") + t.Num("receiving_fumbles") + t.Num("sack_fumbles"),
			Penalties:    t.Num("penalties"),
			PenaltyYards: t.Num("penalty_yards"),
			FumblesLost:  t.Num("rushing_fumbles_lost") + t.Num("receiving_fumbles_lost") + t.Num("sack_fumbles_lost"),
			DefInts:      t.Num("def_interceptions"),
			DefIntYards:  t.Num("def_interception_yards"),
			DefIntTDs:    t.Num("def_tds"),
			PassDefended: t.Num("def_pass_defended"),

			Sacks:            t.Num("def_sacks"),
			TacklesSolo:      t.Num("def_tackles_solo"),
			TackleAssists:    t.Num("def_tackle_assists"),
			TFL:              t.Num("def_tackles_for_loss"),
			QBHits:           t.Num("def_qb_hits"),
			FumblesRecovered: t.Num("fumble_recovery_own") + t.Num("fumble_recovery_opp"),
			FumblesForced:    t.Num("def_fumbles_forced"),
		}

		gid := t.Str("game_id")
		if gid == "" || r.GSIS == "" {
			continue
		}
		d.stats[gid] = append(d.stats[gid], r)
	}
}

func (d *seasonData) loadSnaps() error {
	t, err := dataset.OpenCSV(snapsURL(d.year))
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

		gid := t.Str("game_id")
		pfr := t.Str("pfr_player_id")
		if gid == "" || pfr == "" {
			continue
		}

		d.snaps[gid] = append(d.snaps[gid], snapRow{
			PFRID:    pfr,
			Name:     t.Str("player"),
			Position: t.Str("position"),
			Team:     t.Str("team"),
			OffSnaps: t.Num("offense_snaps"),
			OffPct:   t.Num("offense_pct"),
			DefSnaps: t.Num("defense_snaps"),
			DefPct:   t.Num("defense_pct"),
			STSnaps:  t.Num("st_snaps"),
			STPct:    t.Num("st_pct"),
		})
	}
}

// pbpTeam is one team's play-by-play derived totals for a game.
type pbpTeam struct {
	FirstDownsRush, FirstDownsPass, FirstDownsPenalty int
	Drives                                            []drive
}

// drive is one offensive possession.
type drive struct {
	Result   string
	Score    bool
	Plays    int
	Seconds  int
	StartOwn float64 // starting field position, yards from own goal line
	Yards    int
	Points   int
}

func (p *pbpTeam) firstDowns() int {
	return p.FirstDownsRush + p.FirstDownsPass + p.FirstDownsPenalty
}

// parseClock converts "6:11" to seconds.
func parseClock(s string) int {
	m, sec, ok := strings.Cut(s, ":")
	if !ok {
		return 0
	}
	mi, _ := strconv.Atoi(m)
	si, _ := strconv.Atoi(sec)
	return mi*60 + si
}

// parseYardLine converts "ARI 30" into yards from the possessing team's own
// goal line (30), or "LAC 44" while ARI has the ball into 56.
func parseYardLine(s, offense string) (float64, bool) {
	side, num, ok := strings.Cut(s, " ")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, false
	}
	if findTeam(side) == findTeam(offense) {
		return n, true
	}
	return 100 - n, true
}

// loadPBP scans play-by-play for things the weekly stat files don't include:
// each player's longest pass, rush and reception, and each team's first downs
// and drives.
func (d *seasonData) loadPBP() error {
	t, err := dataset.OpenCSV(pbpURL(d.year))
	if err != nil {
		return err
	}
	defer t.Close()

	bump := func(gid, id string, set func(l *longest)) {
		if id == "" {
			return
		}
		m := d.longs[gid]
		if m == nil {
			m = map[string]*longest{}
			d.longs[gid] = m
		}
		l := m[id]
		if l == nil {
			l = &longest{}
			m[id] = l
		}
		set(l)
	}

	team := func(gid, abbr string) *pbpTeam {
		m := d.pbp[gid]
		if m == nil {
			m = map[string]*pbpTeam{}
			d.pbp[gid] = m
		}
		key := pfrAbbr(abbr)
		p := m[key]
		if p == nil {
			p = &pbpTeam{}
			m[key] = p
		}
		return p
	}

	// A drive's rows are spread through the game; accumulate them by id and
	// attach them to the team once the file is read.
	type driveAcc struct {
		team  string
		drive drive
		set   bool
	}
	drives := map[string]*driveAcc{}
	var order []string

	for {
		ok, err := t.Next()
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		if t.Str("play_deleted") == "1" {
			continue
		}

		gid := t.Str("game_id")
		playType := t.Str("play_type")
		posteam := t.Str("posteam")
		yards := int(t.Num("yards_gained"))

		if posteam != "" {
			p := team(gid, posteam)
			if t.Str("first_down_rush") == "1" {
				p.FirstDownsRush++
			}
			if t.Str("first_down_pass") == "1" {
				p.FirstDownsPass++
			}
			if t.Str("first_down_penalty") == "1" {
				p.FirstDownsPenalty++
			}
		}

		// Drives: attributes repeat on every row of the drive, so read them
		// once. The possessing team comes from a real offensive play, since
		// kickoff rows can carry the kicking team.
		if fd := t.Str("fixed_drive"); fd != "" && t.Str("fixed_drive_result") != "" {
			key := gid + "|" + fd
			a := drives[key]
			if a == nil {
				a = &driveAcc{}
				drives[key] = a
				order = append(order, key)

				a.drive.Result = t.Str("fixed_drive_result")
				a.drive.Score = t.Str("drive_ended_with_score") == "1"
				a.drive.Plays = int(t.Num("drive_play_count"))
				a.drive.Seconds = parseClock(t.Str("drive_time_of_possession"))
				a.drive.Yards = int(t.Num("drive_yards_penalized"))
				switch a.drive.Result {
				case "Touchdown":
					a.drive.Points = 7
				case "Field goal":
					a.drive.Points = 3
				}
			}

			if posteam != "" && (!a.set || a.team == "") &&
				(playType == "pass" || playType == "run" || playType == "punt" || playType == "field_goal") {
				a.team, a.set = posteam, true
				if sy, ok := parseYardLine(t.Str("drive_start_yard_line"), posteam); ok {
					a.drive.StartOwn = sy
				}
			}

			if a.set && posteam == a.team && (playType == "pass" || playType == "run") && t.Str("two_point_attempt") != "1" {
				a.drive.Yards += yards
			}
		}

		if t.Str("two_point_attempt") == "1" || t.Str("sack") == "1" {
			continue
		}

		switch {
		case t.Str("pass_attempt") == "1" && t.Str("complete_pass") == "1":
			bump(gid, t.Str("passer_player_id"), func(l *longest) { l.Pass = max(l.Pass, yards) })
			bump(gid, t.Str("receiver_player_id"), func(l *longest) { l.Receive = max(l.Receive, yards) })
		case t.Str("rush_attempt") == "1":
			bump(gid, t.Str("rusher_player_id"), func(l *longest) { l.Rush = max(l.Rush, yards) })
		}
	}

	for _, key := range order {
		a := drives[key]
		if !a.set {
			continue
		}
		gid, _, _ := strings.Cut(key, "|")
		p := team(gid, a.team)
		p.Drives = append(p.Drives, a.drive)
	}

	return nil
}

// passerRating computes the NFL passer rating (0-158.3).
func passerRating(cmp, att, yds, td, ints float64) float64 {
	if att <= 0 {
		return 0
	}
	clamp := func(v float64) float64 { return math.Max(0, math.Min(2.375, v)) }

	a := clamp((cmp/att - 0.3) * 5)
	b := clamp((yds/att - 3) * 0.25)
	c := clamp(td / att * 20)
	d := clamp(2.375 - ints/att*25)

	return math.Round((a+b+c+d)/6*1000) / 10
}
