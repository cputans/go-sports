package nfl

import (
	"fmt"
	"strings"

	"github.com/cputans/go-sports/internal"
)

type Season struct {
	URL      string
	NFCTeams []SeasonTeam `tableId:"NFC" rowSelector:"tr:not(.thead)" cellSelector:"td,th"`
	AFCTeams []SeasonTeam `tableId:"AFC" rowSelector:"tr:not(.thead)" cellSelector:"td,th"`
}

type SeasonTeam struct {
	Name               string `cell:"0"`
	ShortName          string
	Year               uint
	Wins               uint    `cell:"1"`
	Losses             uint    `cell:"2"`
	WinLossPercentage  float64 `cell:"3"`
	PointsFor          uint    `cell:"4"`
	PointsAgainst      uint    `cell:"5"`
	PointDifferential  uint    `cell:"6"`
	MarginOfVictory    float64 `cell:"7"`
	StrengthOfSchedule float64 `cell:"8"`
	URL                string  `cell:"0" dataSelector:"a" attr:"href"`
}

func NewSeason(year uint) *Season {
	url := fmt.Sprintf(SEASON_URL, year)
	return &Season{
		URL: url,
	}
}

func (s *Season) Parse() {
	internal.Parse(s, s.URL)
}

func (s *Season) PostProcess() {
	for i, t := range s.NFCTeams {
		urlParts := strings.Split(t.URL, "/")
		t.Name = strings.Replace(t.Name, "*", "", -1)
		t.Name = strings.Replace(t.Name, "+", "", -1)
		t.ShortName = urlParts[2]
		s.NFCTeams[i] = t
	}

	for i, t := range s.AFCTeams {
		urlParts := strings.Split(t.URL, "/")
		t.Name = strings.Replace(t.Name, "*", "", -1)
		t.Name = strings.Replace(t.Name, "+", "", -1)
		t.ShortName = urlParts[2]
		s.AFCTeams[i] = t
	}
}
