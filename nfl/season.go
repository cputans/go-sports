package nfl

import (
	"fmt"
	"strings"
)

type Season struct {
	Year     uint
	NFCTeams []SeasonTeam
	AFCTeams []SeasonTeam

	// Err is set by Parse when the season could not be loaded.
	Err error
}

type SeasonTeam struct {
	Name              string
	ShortName         string // lower-case PFR abbreviation, accepted by NewTeam
	Year              uint
	Wins              uint
	Losses            uint
	Ties              uint
	WinLossPercentage float64
	PointsFor         uint
	PointsAgainst     uint
	PointDifferential int
	MarginOfVictory   float64

	// StrengthOfSchedule is the average rating of the team's opponents, using
	// the same simple rating system PFR uses (margin of victory adjusted for
	// opponents).
	StrengthOfSchedule float64
	URL                string
}

func NewSeason(year uint) *Season {
	return &Season{Year: year}
}

// Parse builds the season's teams and regular-season records from nflverse data.
func (s *Season) Parse() {
	data, err := loadSeason(int(s.Year))
	if err != nil {
		s.Err = err
		return
	}

	records := map[string]*SeasonTeam{}
	var results []gameResult
	for i := range nflTeams {
		t := &nflTeams[i]
		records[t.Abbr] = &SeasonTeam{
			Name:      t.Name,
			ShortName: strings.ToLower(t.PFR),
			Year:      s.Year,
			URL:       fmt.Sprintf("/teams/%s/%d.htm", strings.ToLower(t.PFR), s.Year),
		}
	}

	for _, g := range data.games {
		if !g.Played || g.Type != "REG" {
			continue
		}
		home, away := findTeam(g.Home), findTeam(g.Away)
		if home == nil || away == nil {
			continue
		}
		h, a := records[home.Abbr], records[away.Abbr]

		results = append(results, gameResult{home.Abbr, away.Abbr, g.HomeScore - g.AwayScore})

		h.PointsFor += uint(g.HomeScore)
		h.PointsAgainst += uint(g.AwayScore)
		a.PointsFor += uint(g.AwayScore)
		a.PointsAgainst += uint(g.HomeScore)

		switch {
		case g.HomeScore > g.AwayScore:
			h.Wins++
			a.Losses++
		case g.HomeScore < g.AwayScore:
			a.Wins++
			h.Losses++
		default:
			h.Ties++
			a.Ties++
		}
	}

	sos := strengthOfSchedule(results)

	s.NFCTeams, s.AFCTeams = nil, nil
	for i := range nflTeams {
		t := &nflTeams[i]
		r := records[t.Abbr]

		if played := r.Wins + r.Losses + r.Ties; played > 0 {
			r.WinLossPercentage = (float64(r.Wins) + 0.5*float64(r.Ties)) / float64(played)
			r.MarginOfVictory = float64(int(r.PointsFor)-int(r.PointsAgainst)) / float64(played)
		}
		r.PointDifferential = int(r.PointsFor) - int(r.PointsAgainst)
		r.StrengthOfSchedule = sos[t.Abbr]

		if t.Conference == "NFC" {
			s.NFCTeams = append(s.NFCTeams, *r)
		} else {
			s.AFCTeams = append(s.AFCTeams, *r)
		}
	}
}

type gameResult struct {
	home, away string
	margin     int // home score minus away score
}

// strengthOfSchedule solves the simple rating system: each team's rating is
// its average margin of victory plus the average rating of its opponents.
// The schedule component (the average opponent rating) is returned per team.
func strengthOfSchedule(results []gameResult) map[string]float64 {
	margins := map[string]float64{}
	opponents := map[string][]string{}
	for _, r := range results {
		margins[r.home] += float64(r.margin)
		margins[r.away] -= float64(r.margin)
		opponents[r.home] = append(opponents[r.home], r.away)
		opponents[r.away] = append(opponents[r.away], r.home)
	}

	mov := map[string]float64{}
	for team, m := range margins {
		mov[team] = m / float64(len(opponents[team]))
	}

	rating := map[string]float64{}
	for team := range mov {
		rating[team] = mov[team]
	}

	sos := map[string]float64{}
	for i := 0; i < 500; i++ {
		next := map[string]float64{}
		var mean float64
		for team, opps := range opponents {
			var sum float64
			for _, o := range opps {
				sum += rating[o]
			}
			sos[team] = sum / float64(len(opps))

			// Damped so that a bipartite schedule converges instead of oscillating.
			next[team] = 0.5*rating[team] + 0.5*(mov[team]+sos[team])
			mean += next[team]
		}

		// Ratings are only defined up to a constant; centre them on zero.
		mean /= float64(len(next))
		for team := range next {
			next[team] -= mean
		}
		rating = next
	}

	return sos
}
