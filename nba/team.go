package nba

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Team struct {
	Name      string
	ShortName string // ESPN abbreviation, e.g. "GS"; accepted in any case
	Year      uint

	// Regular season to date, excluding unplayed games.
	Wins          uint
	Losses        uint
	PointsFor     uint
	PointsAgainst uint

	// Games is the full schedule (preseason, regular season, play-in and
	// postseason) in date order.
	Games []TeamGame

	// Err is set by Parse when the team could not be loaded.
	Err error
}

type TeamGame struct {
	ID                string // ESPN game ID, accepted by NewBoxscore
	Date              time.Time
	SeasonType        string // "preseason", "regular", "postseason" or "play-in"
	Result            string // "W" or "L"; empty until the game is played
	Location          string // "home" or "away"
	Opponent          string
	OpponentShortName string
	TeamScore         uint
	OpponentScore     uint
	HomeTeam          string
	HomeTeamShortName string
	AwayTeam          string
	AwayTeamShortName string
}

func NewTeam(shortName string, year uint) *Team {
	return &Team{ShortName: shortName, Year: year}
}

// Parse builds the team's schedule and regular-season record.
func (t *Team) Parse() {
	data, err := loadSeason(int(t.Year))
	if err != nil {
		t.Err = err
		return
	}

	t.Games = nil
	for _, g := range data.games {
		var me, opp teamRef
		var loc string
		switch {
		case strings.EqualFold(g.Home.Abbr, t.ShortName):
			me, opp, loc = g.Home, g.Away, "home"
		case strings.EqualFold(g.Away.Abbr, t.ShortName):
			me, opp, loc = g.Away, g.Home, "away"
		default:
			continue
		}

		t.Name = me.Name
		t.ShortName = me.Abbr

		tg := TeamGame{
			ID:                g.ID,
			Date:              g.Date,
			SeasonType:        seasonTypeName(g.SeasonType),
			Location:          loc,
			Opponent:          opp.Name,
			OpponentShortName: opp.Abbr,
			HomeTeam:          g.Home.Name,
			HomeTeamShortName: g.Home.Abbr,
			AwayTeam:          g.Away.Name,
			AwayTeamShortName: g.Away.Abbr,
		}
		if g.Played {
			tg.TeamScore, tg.OpponentScore = uint(me.Score), uint(opp.Score)
			tg.Result = "L"
			if me.Score > opp.Score {
				tg.Result = "W"
			}

			if g.SeasonType == seasonRegular {
				if tg.Result == "W" {
					t.Wins++
				} else {
					t.Losses++
				}
				t.PointsFor += tg.TeamScore
				t.PointsAgainst += tg.OpponentScore
			}
		}
		t.Games = append(t.Games, tg)
	}

	if len(t.Games) == 0 {
		t.Err = fmt.Errorf("unknown team %q", t.ShortName)
		return
	}

	sort.SliceStable(t.Games, func(i, j int) bool { return t.Games[i].Date.Before(t.Games[j].Date) })
}
