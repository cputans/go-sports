package nfl

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Data in this package comes from the nflverse project
// (https://github.com/nflverse/nflverse-data), which publishes NFL data as
// static files on GitHub releases. A season is downloaded once per process.

var (
	// ErrStatsNotAvailable is returned when a game has been played (the
	// schedule has a score) but nflverse hasn't published its player stats yet.
	ErrStatsNotAvailable = errors.New("player stats not yet available for game")
	// ErrGameNotFound is returned when a game ID is not in the season schedule.
	ErrGameNotFound = errors.New("game not found")

	errNotFound = errors.New("file not found")
)

// fetch downloads a URL. It is a variable so tests can replace it.
var fetch = func(url string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			return resp.Body, nil
		case resp.StatusCode == http.StatusNotFound:
			resp.Body.Close()
			return nil, fmt.Errorf("%s: %w", url, errNotFound)
		default:
			resp.Body.Close()
			lastErr = fmt.Errorf("%s: unexpected status %d", url, resp.StatusCode)
		}
	}

	return nil, lastErr
}

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

// csvTable is a streaming CSV reader with access to columns by header name.
type csvTable struct {
	r   *csv.Reader
	idx map[string]int
	rec []string
	rc  io.Closer
}

func openCSV(url string) (*csvTable, error) {
	body, err := fetch(url)
	if err != nil {
		return nil, err
	}

	var src io.Reader = body
	closer := io.Closer(body)
	if strings.HasSuffix(url, ".gz") {
		gz, err := gzip.NewReader(body)
		if err != nil {
			body.Close()
			return nil, fmt.Errorf("%s: %w", url, err)
		}
		src = gz
	}

	r := csv.NewReader(src)
	r.FieldsPerRecord = -1
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		closer.Close()
		return nil, fmt.Errorf("%s: reading header: %w", url, err)
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}

	return &csvTable{r: r, idx: idx, rc: closer}, nil
}

func (t *csvTable) close() { t.rc.Close() }

// next advances to the next record. It returns an error on a malformed file.
func (t *csvTable) next() (bool, error) {
	rec, err := t.r.Read()
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	t.rec = rec
	return true, nil
}

func (t *csvTable) str(col string) string {
	i, ok := t.idx[col]
	if !ok || i >= len(t.rec) {
		return ""
	}
	v := t.rec[i]
	if v == "NA" {
		return ""
	}
	return v
}

func (t *csvTable) num(col string) float64 {
	v, _ := strconv.ParseFloat(t.str(col), 64)
	return v
}

func (t *csvTable) has(col string) bool {
	_, ok := t.idx[col]
	return ok
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

type seasonEntry struct {
	once sync.Once
	data *seasonData
	err  error
}

var (
	seasonsMu sync.Mutex
	seasons   = map[int]*seasonEntry{}
)

func resetSeasonCache() {
	seasonsMu.Lock()
	defer seasonsMu.Unlock()
	seasons = map[int]*seasonEntry{}
}

// loadSeason returns the (cached) data for a season, downloading it on first use.
func loadSeason(year int) (*seasonData, error) {
	seasonsMu.Lock()
	e, ok := seasons[year]
	if !ok {
		e = &seasonEntry{}
		seasons[year] = e
	}
	seasonsMu.Unlock()

	e.once.Do(func() { e.data, e.err = downloadSeason(year) })
	return e.data, e.err
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
		if err := load(); err != nil && !errors.Is(err, errNotFound) {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
	}

	return d, nil
}

func (d *seasonData) loadSchedule() error {
	t, err := openCSV(scheduleURL())
	if err != nil {
		return err
	}
	defer t.close()

	for {
		ok, err := t.next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		if t.str("season") != strconv.Itoa(d.year) {
			continue
		}

		g := &gameRow{
			ID:       t.str("game_id"),
			PFRID:    t.str("pfr"),
			Season:   d.year,
			Type:     t.str("game_type"),
			Week:     int(t.num("week")),
			Away:     t.str("away_team"),
			Home:     t.str("home_team"),
			Overtime: t.str("overtime") == "1",
			Roof:     t.str("roof"),
			Surface:  t.str("surface"),
			Temp:     t.str("temp"),
			Wind:     t.str("wind"),
		}

		clock := t.str("gametime")
		if clock == "" {
			clock = "00:00"
		}
		g.Date, _ = time.Parse("2006-01-02 15:04", t.str("gameday")+" "+clock)

		if t.str("home_score") != "" && t.str("away_score") != "" {
			g.Played = true
			g.HomeScore = int(t.num("home_score"))
			g.AwayScore = int(t.num("away_score"))
		}
		if v := t.str("spread_line"); v != "" {
			g.Spread, g.HasSpread = t.num("spread_line"), true
		}
		if v := t.str("total_line"); v != "" {
			g.Total, g.HasTotal = t.num("total_line"), true
		}

		d.games = append(d.games, g)
		if g.PFRID != "" {
			d.byPFR[g.PFRID] = g
		}
	}
}

func (d *seasonData) loadPlayers() error {
	t, err := openCSV(playersURL())
	if err != nil {
		return err
	}
	defer t.close()

	for {
		ok, err := t.next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if g, p := t.str("gsis_id"), t.str("pfr_id"); g != "" && p != "" {
			d.pfrIDs[g] = p
		}
	}
}

func (d *seasonData) loadStats() error {
	t, err := openCSV(statsURL(d.year))
	if err != nil {
		return err
	}
	defer t.close()

	for {
		ok, err := t.next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		name := t.str("player_display_name")
		if name == "" {
			name = t.str("player_name")
		}

		r := statRow{
			GSIS:     t.str("player_id"),
			Name:     name,
			Position: t.str("position"),
			Team:     t.str("team"),

			PassCompletions: t.num("completions"),
			PassAttempts:    t.num("attempts"),
			PassYards:       t.num("passing_yards"),
			PassTDs:         t.num("passing_tds"),
			PassInts:        t.num("passing_interceptions"),
			Sacked:          t.num("sacks_suffered"),
			SackedYards:     math.Abs(t.num("sack_yards_lost")),

			RushAttempts: t.num("carries"),
			RushYards:    t.num("rushing_yards"),
			RushTDs:      t.num("rushing_tds"),

			Targets:    t.num("targets"),
			Receptions: t.num("receptions"),
			RecYards:   t.num("receiving_yards"),
			RecTDs:     t.num("receiving_tds"),

			Fumbles:      t.num("rushing_fumbles") + t.num("receiving_fumbles") + t.num("sack_fumbles"),
			Penalties:    t.num("penalties"),
			PenaltyYards: t.num("penalty_yards"),
			FumblesLost:  t.num("rushing_fumbles_lost") + t.num("receiving_fumbles_lost") + t.num("sack_fumbles_lost"),
			DefInts:      t.num("def_interceptions"),
			DefIntYards:  t.num("def_interception_yards"),
			DefIntTDs:    t.num("def_tds"),
			PassDefended: t.num("def_pass_defended"),

			Sacks:            t.num("def_sacks"),
			TacklesSolo:      t.num("def_tackles_solo"),
			TackleAssists:    t.num("def_tackle_assists"),
			TFL:              t.num("def_tackles_for_loss"),
			QBHits:           t.num("def_qb_hits"),
			FumblesRecovered: t.num("fumble_recovery_own") + t.num("fumble_recovery_opp"),
			FumblesForced:    t.num("def_fumbles_forced"),
		}

		gid := t.str("game_id")
		if gid == "" || r.GSIS == "" {
			continue
		}
		d.stats[gid] = append(d.stats[gid], r)
	}
}

func (d *seasonData) loadSnaps() error {
	t, err := openCSV(snapsURL(d.year))
	if err != nil {
		return err
	}
	defer t.close()

	for {
		ok, err := t.next()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		gid := t.str("game_id")
		pfr := t.str("pfr_player_id")
		if gid == "" || pfr == "" {
			continue
		}

		d.snaps[gid] = append(d.snaps[gid], snapRow{
			PFRID:    pfr,
			Name:     t.str("player"),
			Position: t.str("position"),
			Team:     t.str("team"),
			OffSnaps: t.num("offense_snaps"),
			OffPct:   t.num("offense_pct"),
			DefSnaps: t.num("defense_snaps"),
			DefPct:   t.num("defense_pct"),
			STSnaps:  t.num("st_snaps"),
			STPct:    t.num("st_pct"),
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
	t, err := openCSV(pbpURL(d.year))
	if err != nil {
		return err
	}
	defer t.close()

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
		ok, err := t.next()
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		if t.str("play_deleted") == "1" {
			continue
		}

		gid := t.str("game_id")
		playType := t.str("play_type")
		posteam := t.str("posteam")
		yards := int(t.num("yards_gained"))

		if posteam != "" {
			p := team(gid, posteam)
			if t.str("first_down_rush") == "1" {
				p.FirstDownsRush++
			}
			if t.str("first_down_pass") == "1" {
				p.FirstDownsPass++
			}
			if t.str("first_down_penalty") == "1" {
				p.FirstDownsPenalty++
			}
		}

		// Drives: attributes repeat on every row of the drive, so read them
		// once. The possessing team comes from a real offensive play, since
		// kickoff rows can carry the kicking team.
		if fd := t.str("fixed_drive"); fd != "" && t.str("fixed_drive_result") != "" {
			key := gid + "|" + fd
			a := drives[key]
			if a == nil {
				a = &driveAcc{}
				drives[key] = a
				order = append(order, key)

				a.drive.Result = t.str("fixed_drive_result")
				a.drive.Score = t.str("drive_ended_with_score") == "1"
				a.drive.Plays = int(t.num("drive_play_count"))
				a.drive.Seconds = parseClock(t.str("drive_time_of_possession"))
				a.drive.Yards = int(t.num("drive_yards_penalized"))
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
				if sy, ok := parseYardLine(t.str("drive_start_yard_line"), posteam); ok {
					a.drive.StartOwn = sy
				}
			}

			if a.set && posteam == a.team && (playType == "pass" || playType == "run") && t.str("two_point_attempt") != "1" {
				a.drive.Yards += yards
			}
		}

		if t.str("two_point_attempt") == "1" || t.str("sack") == "1" {
			continue
		}

		switch {
		case t.str("pass_attempt") == "1" && t.str("complete_pass") == "1":
			bump(gid, t.str("passer_player_id"), func(l *longest) { l.Pass = max(l.Pass, yards) })
			bump(gid, t.str("receiver_player_id"), func(l *longest) { l.Receive = max(l.Receive, yards) })
		case t.str("rush_attempt") == "1":
			bump(gid, t.str("rusher_player_id"), func(l *longest) { l.Rush = max(l.Rush, yards) })
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
