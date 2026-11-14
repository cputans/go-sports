package nfl

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Boxscore struct {
	ID      string
	DateStr string // e.g. "Sunday Sep 13, 2026"
	TimeStr string // e.g. "Start Time: 1:00pm"
	Date    time.Time

	// WonToss, Duration and Attendance aren't published by nflverse and are
	// always empty.
	WonToss    string
	Duration   string
	Attendance string

	Roof               string
	Surface            string
	Weather            string
	VegasLine          string
	OverUnder          string
	HomeTeam           string
	HomeTeamLink       string
	HomeTeamShortName  string
	HomeTeamScore      uint
	HomeTeamStats      BoxscoreTeamStats
	AwayTeam           string
	AwayTeamLink       string
	AwayTeamShortName  string
	AwayTeamScore      uint
	AwayTeamStats      BoxscoreTeamStats
	Players            map[string]BoxscorePlayer
	PlayerStatsOffense []BoxscorePlayerOffensiveStats
	PlayerStatsDefense []BoxscorePlayerDefensiveStats
	HomeSnapCounts     []BoxscoreSnapCounts
	AwaySnapCounts     []BoxscoreSnapCounts
	URL                string

	// Err is set by Parse when the boxscore could not be built. The other
	// fields must not be used when it is non-nil.
	Err error
}

type BoxscoreTeamStats struct {
	FirstDowns           uint
	RushingInfo          string // attempts-yards-touchdowns, e.g. "25-130-2"
	RushingAttempts      uint
	RushingYards         int
	RushingTouchdowns    uint
	PassingInfo          string // completions-attempts-yards-touchdowns-interceptions
	PassingCompletions   uint
	PassingAttempts      uint
	PassingYards         int
	PassingTouchdowns    uint
	PassingInterceptions uint
}

type BoxscorePlayer struct {
	Name          string
	ID            string
	TeamShortName string
	BoxscoreSnapCounts
	BoxscorePlayerOffensiveStats
	BoxscorePlayerDefensiveStats
	URL string
}

type BoxscoreSnapCounts struct {
	Name                       string
	ID                         string
	Position                   string
	OffSnapCount               uint
	OffSnapPercentage          string
	DefSnapCount               uint
	DefSnapPercentage          string
	SpecialTeamsSnapCount      uint
	SpecialTeamsSnapPercentage string
	URL                        string
}

type BoxscorePlayerOffensiveStats struct {
	Name                   string
	ID                     string
	TeamShortName          string
	PassingCompletions     uint
	PassingAttempts        uint
	PassingYards           int
	PassingTouchdowns      uint
	PassingInterceptions   uint
	PassingSacked          uint
	PassingSackedLostYards uint
	PassingLong            uint
	PassingRating          float64
	RushingAttempts        uint
	RushingYards           int
	RushingTouchdowns      uint
	RushingLong            uint
	ReceivingTargets       uint
	ReceivingReceptions    uint
	ReceivingYards         int
	ReceivingTouchdowns    uint
	ReceivingLong          uint
	Fumbles                uint
	FumblesLost            uint
	URL                    string
}

type BoxscorePlayerDefensiveStats struct {
	Name                   string
	ID                     string
	TeamShortName          string
	Interceptions          uint
	InterceptionYards      int
	InterceptionTouchdowns uint
	PassDeflections        uint
	Sacks                  float64
	TacklesCombined        uint
	TacklesSolo            uint
	TacklesAssists         uint
	TacklesForLoss         uint
	QuarterbackHits        uint
	FumblesRecovered       uint
	FumblesForced          uint
	URL                    string
}

var boxscoreIDRegex = regexp.MustCompile(`(?i)(?:/boxscores/)?([0-9]{9}[a-z]{3})(?:\.htm)?$`)

// NewBoxscore creates a boxscore from a PFR-style game reference, either a
// boxscore path ("/boxscores/202609130nyg.htm") or a bare ID ("202609130nyg").
func NewBoxscore(name string) *Boxscore {
	b := &Boxscore{URL: name}
	if m := boxscoreIDRegex.FindStringSubmatch(name); m != nil {
		b.ID = strings.ToLower(m[1])
	}
	return b
}

// seasonOfPFRID derives the season from a PFR game ID (yyyymmdd + digit + team).
// Games in January and February belong to the season that started the year before.
func seasonOfPFRID(id string) (int, bool) {
	if len(id) < 8 {
		return 0, false
	}
	year, err := strconv.Atoi(id[:4])
	if err != nil {
		return 0, false
	}
	month, err := strconv.Atoi(id[4:6])
	if err != nil {
		return 0, false
	}
	if month <= 2 {
		year--
	}
	return year, true
}

// Parse builds the boxscore from nflverse data. On failure it sets Err.
func (b *Boxscore) Parse() {
	b.Err = b.load()
}

func (b *Boxscore) load() error {
	if b.ID == "" {
		return fmt.Errorf("invalid boxscore reference %q", b.URL)
	}

	year, ok := seasonOfPFRID(b.ID)
	if !ok {
		return fmt.Errorf("cannot determine season from boxscore id %q", b.ID)
	}

	data, err := loadSeason(year)
	if err != nil {
		return err
	}

	game, ok := data.byPFR[b.ID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrGameNotFound, b.ID)
	}

	stats := data.stats[game.ID]
	if game.Played && len(stats) == 0 {
		return fmt.Errorf("%w: %s", ErrStatsNotAvailable, b.ID)
	}

	b.Date = game.Date
	b.DateStr = game.Date.Format("Monday Jan 2, 2006")
	b.TimeStr = "Start Time: " + game.Date.Format("3:04pm")
	b.Roof = game.Roof
	b.Surface = game.Surface
	b.HomeTeam = teamName(game.Home)
	b.HomeTeamShortName = pfrAbbr(game.Home)
	b.HomeTeamLink = teamLink(game.Home, year)
	b.HomeTeamScore = uint(game.HomeScore)
	b.AwayTeam = teamName(game.Away)
	b.AwayTeamShortName = pfrAbbr(game.Away)
	b.AwayTeamLink = teamLink(game.Away, year)
	b.AwayTeamScore = uint(game.AwayScore)
	b.Weather = weatherString(game)
	b.VegasLine = vegasLine(game)
	if game.HasTotal {
		b.OverUnder = fmt.Sprintf("%.1f", game.Total)
	}

	b.HomeTeamStats = teamStats(data, game, game.Home)
	b.AwayTeamStats = teamStats(data, game, game.Away)

	b.Players = buildPlayers(data, game, stats)
	b.fillPlayerTables(data, game)

	return nil
}

func weatherString(g *gameRow) string {
	if g.Temp == "" {
		return ""
	}
	if g.Wind == "" {
		return fmt.Sprintf("%s degrees", g.Temp)
	}
	return fmt.Sprintf("%s degrees, wind %s mph", g.Temp, g.Wind)
}

// vegasLine formats the spread like PFR does, e.g. "Philadelphia Eagles -3.0".
// nflverse's spread_line is positive when the home team is favored.
func vegasLine(g *gameRow) string {
	if !g.HasSpread {
		return ""
	}
	if g.Spread == 0 {
		return "Pick"
	}
	if g.Spread > 0 {
		return fmt.Sprintf("%s -%.1f", teamName(g.Home), g.Spread)
	}
	return fmt.Sprintf("%s -%.1f", teamName(g.Away), -g.Spread)
}

func teamLink(abbr string, year int) string {
	return fmt.Sprintf("/teams/%s/%d.htm", strings.ToLower(pfrAbbr(abbr)), year)
}

func teamStats(data *seasonData, game *gameRow, team string) BoxscoreTeamStats {
	t := data.gameSide(game, team)

	return BoxscoreTeamStats{
		FirstDowns:           uint(t.firstDowns()),
		RushingInfo:          fmt.Sprintf("%d-%d-%d", int(t.RushAttempts), int(t.RushYards), int(t.RushTDs)),
		RushingAttempts:      uint(t.RushAttempts),
		RushingYards:         int(t.RushYards),
		RushingTouchdowns:    uint(t.RushTDs),
		PassingInfo:          fmt.Sprintf("%d-%d-%d-%d-%d", int(t.PassCompletions), int(t.PassAttempts), int(t.PassYards), int(t.PassTDs), int(t.Interceptions)),
		PassingCompletions:   uint(t.PassCompletions),
		PassingAttempts:      uint(t.PassAttempts),
		PassingYards:         int(t.PassYards),
		PassingTouchdowns:    uint(t.PassTDs),
		PassingInterceptions: uint(t.Interceptions),
	}
}

func playerURL(id string) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf("/players/%s/%s.htm", strings.ToUpper(id[:1]), id)
}

// fillPlayerTables fills the per-table views of the players (offense, defense
// and each team's snap counts) and each player's URL.
func (b *Boxscore) fillPlayerTables(data *seasonData, game *gameRow) {
	ids := make([]string, 0, len(b.Players))
	for id := range b.Players {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		p := b.Players[id]

		url := playerURL(id)
		p.URL = url
		p.BoxscoreSnapCounts.URL = url
		p.BoxscorePlayerOffensiveStats.URL = url
		p.BoxscorePlayerDefensiveStats.URL = url
		b.Players[id] = p

		if p.BoxscoreSnapCounts.ID != "" {
			if p.TeamShortName == b.HomeTeamShortName {
				b.HomeSnapCounts = append(b.HomeSnapCounts, p.BoxscoreSnapCounts)
			} else {
				b.AwaySnapCounts = append(b.AwaySnapCounts, p.BoxscoreSnapCounts)
			}
		}

		if o := p.BoxscorePlayerOffensiveStats; o.ID != "" &&
			(o.PassingAttempts+o.PassingSacked+o.RushingAttempts+o.ReceivingTargets+o.Fumbles) > 0 {
			b.PlayerStatsOffense = append(b.PlayerStatsOffense, o)
		}

		if d := p.BoxscorePlayerDefensiveStats; d.ID != "" &&
			(d.Sacks > 0 || d.Interceptions+d.PassDeflections+d.TacklesCombined+d.TacklesForLoss+d.QuarterbackHits+d.FumblesRecovered+d.FumblesForced > 0) {
			b.PlayerStatsDefense = append(b.PlayerStatsDefense, d)
		}
	}
}

func pct(v float64) string {
	return fmt.Sprintf("%.0f%%", v*100)
}

func buildPlayers(data *seasonData, game *gameRow, stats []statRow) map[string]BoxscorePlayer {
	players := map[string]BoxscorePlayer{}
	longs := data.longs[game.ID]

	snapIDs := map[string]string{}

	// Snap counts are keyed on PFR player ids.
	for _, s := range data.snaps[game.ID] {
		snapIDs[s.Name+"|"+pfrAbbr(s.Team)] = s.PFRID
		p := players[s.PFRID]
		p.ID = s.PFRID
		p.Name = s.Name
		p.TeamShortName = pfrAbbr(s.Team)
		p.BoxscoreSnapCounts = BoxscoreSnapCounts{
			Name:                       s.Name,
			ID:                         s.PFRID,
			Position:                   s.Position,
			OffSnapCount:               uint(s.OffSnaps),
			OffSnapPercentage:          pct(s.OffPct),
			DefSnapCount:               uint(s.DefSnaps),
			DefSnapPercentage:          pct(s.DefPct),
			SpecialTeamsSnapCount:      uint(s.STSnaps),
			SpecialTeamsSnapPercentage: pct(s.STPct),
		}
		players[s.PFRID] = p
	}

	for _, r := range stats {
		// Stats are keyed on GSIS ids; translate to PFR ids so documents keep
		// the same ids they had when the data came from PFR.
		id, ok := data.pfrIDs[r.GSIS]
		if !ok {
			// Not in the player table (usually a new player): match the snap
			// count row by name and team, else fall back to the GSIS id.
			id, ok = snapIDs[r.Name+"|"+pfrAbbr(r.Team)]
			if !ok {
				id = r.GSIS
			}
		}

		p := players[id]
		p.ID = id
		p.Name = r.Name
		p.TeamShortName = pfrAbbr(r.Team)

		var l longest
		if v := longs[r.GSIS]; v != nil {
			l = *v
		}

		p.BoxscorePlayerOffensiveStats = BoxscorePlayerOffensiveStats{
			Name:                   r.Name,
			ID:                     id,
			TeamShortName:          p.TeamShortName,
			PassingCompletions:     uint(r.PassCompletions),
			PassingAttempts:        uint(r.PassAttempts),
			PassingYards:           int(r.PassYards),
			PassingTouchdowns:      uint(r.PassTDs),
			PassingInterceptions:   uint(r.PassInts),
			PassingSacked:          uint(r.Sacked),
			PassingSackedLostYards: uint(r.SackedYards),
			PassingLong:            uint(max(l.Pass, 0)),
			PassingRating:          passerRating(r.PassCompletions, r.PassAttempts, r.PassYards, r.PassTDs, r.PassInts),
			RushingAttempts:        uint(r.RushAttempts),
			RushingYards:           int(r.RushYards),
			RushingTouchdowns:      uint(r.RushTDs),
			RushingLong:            uint(max(l.Rush, 0)),
			ReceivingTargets:       uint(r.Targets),
			ReceivingReceptions:    uint(r.Receptions),
			ReceivingYards:         int(r.RecYards),
			ReceivingTouchdowns:    uint(r.RecTDs),
			ReceivingLong:          uint(max(l.Receive, 0)),
			Fumbles:                uint(r.Fumbles),
			FumblesLost:            uint(r.FumblesLost),
		}

		p.BoxscorePlayerDefensiveStats = BoxscorePlayerDefensiveStats{
			Name:                   r.Name,
			ID:                     id,
			TeamShortName:          p.TeamShortName,
			Interceptions:          uint(r.DefInts),
			InterceptionYards:      int(r.DefIntYards),
			InterceptionTouchdowns: uint(r.DefIntTDs),
			PassDeflections:        uint(r.PassDefended),
			Sacks:                  r.Sacks,
			TacklesCombined:        uint(r.TacklesSolo + r.TackleAssists),
			TacklesSolo:            uint(r.TacklesSolo),
			TacklesAssists:         uint(r.TackleAssists),
			TacklesForLoss:         uint(r.TFL),
			QuarterbackHits:        uint(r.QBHits),
			FumblesRecovered:       uint(r.FumblesRecovered),
			FumblesForced:          uint(r.FumblesForced),
		}

		if p.BoxscoreSnapCounts.Position == "" {
			p.BoxscoreSnapCounts.Position = r.Position
		}
		players[id] = p
	}

	return players
}
