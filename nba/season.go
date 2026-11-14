package nba

import "sort"

type Season struct {
	// Year is the year the season ends in: 2027 is the 2026-27 season.
	Year  uint
	Teams []SeasonTeam

	// Err is set by Parse when the season could not be loaded.
	Err error
}

type SeasonTeam struct {
	Name              string
	ShortName         string // ESPN abbreviation, accepted by NewTeam
	Year              uint
	Wins              uint
	Losses            uint
	WinLossPercentage float64
	PointsFor         uint
	PointsAgainst     uint
	PointDifferential int
}

func NewSeason(year uint) *Season {
	return &Season{Year: year}
}

// Parse builds the season's teams and regular-season records.
func (s *Season) Parse() {
	data, err := loadSeason(int(s.Year))
	if err != nil {
		s.Err = err
		return
	}

	records := map[string]*SeasonTeam{}
	get := func(r teamRef) *SeasonTeam {
		t, ok := records[r.Abbr]
		if !ok {
			t = &SeasonTeam{Name: r.Name, ShortName: r.Abbr, Year: s.Year}
			records[r.Abbr] = t
		}
		return t
	}

	for _, g := range data.games {
		// Teams appear even before they've played so the list is complete from
		// the start of the season.
		h, a := get(g.Home), get(g.Away)
		if g.SeasonType != seasonRegular || !g.Played {
			continue
		}

		h.PointsFor += uint(g.Home.Score)
		h.PointsAgainst += uint(g.Away.Score)
		a.PointsFor += uint(g.Away.Score)
		a.PointsAgainst += uint(g.Home.Score)
		if g.Home.Score > g.Away.Score {
			h.Wins++
			a.Losses++
		} else {
			a.Wins++
			h.Losses++
		}
	}

	s.Teams = s.Teams[:0]
	for _, t := range records {
		if n := t.Wins + t.Losses; n > 0 {
			t.WinLossPercentage = float64(t.Wins) / float64(n)
		}
		t.PointDifferential = int(t.PointsFor) - int(t.PointsAgainst)
		s.Teams = append(s.Teams, *t)
	}
	sort.Slice(s.Teams, func(i, j int) bool { return s.Teams[i].ShortName < s.Teams[j].ShortName })
}
