package nba

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cputans/go-sports/v2/internal/dataset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scheduleCSV = `game_id,season_type,type_abbreviation,game_date_time,home_id,home_abbreviation,home_display_name,home_score,away_id,away_abbreviation,away_display_name,away_score,status_type_completed
1001,2,STD,2025-10-21T19:30:00.000000-0400,25,OKC,Oklahoma City Thunder,125,13,LAL,Los Angeles Lakers,109,true
1002,2,STD,2025-10-23T19:30:00.000000-0400,9,GS,Golden State Warriors,0,25,OKC,Oklahoma City Thunder,0,false
1003,2,STD,2025-10-22T19:30:00.000000-0400,13,LAL,Los Angeles Lakers,100,9,GS,Golden State Warriors,110,true
1004,2,ALLSTAR,2026-02-15T19:30:00.000000-0400,132375,STRIPES,Team Stripes,21,132374,STARS,Team Stars,20,true
1005,3,STD,2026-04-25T19:30:00.000000-0400,25,OKC,Oklahoma City Thunder,100,13,LAL,Los Angeles Lakers,90,true
1006,2,STD,2025-10-24T19:30:00.000000-0400,25,OKC,Oklahoma City Thunder,100,13,LAL,Los Angeles Lakers,90,true
1007,2,STD,2025-12-16T19:30:00.000000-0400,1,TBD,TBD,0,2,TBD,TBD,0,false
`

const teamBoxCSV = `game_id,team_abbreviation,team_score,field_goals_made,field_goals_attempted,three_point_field_goals_made,three_point_field_goals_attempted,free_throws_made,free_throws_attempted,offensive_rebounds,defensive_rebounds,total_rebounds,assists,steals,blocks,turnovers,fouls,points_in_paint,fast_break_points,largest_lead
1001,OKC,125,45,90,15,40,20,25,10,35,45,28,9,6,12,18,52,14,20
1001,LAL,109,40,88,12,35,17,20,8,30,38,22,6,4,15,20,44,10,5
1003,LAL,100,36,85,10,30,18,22,9,28,37,20,5,3,14,19,40,8,2
1003,GS,110,41,86,14,38,14,16,7,33,40,27,8,5,11,17,46,12,15
`

const playerBoxCSV = `game_id,athlete_id,athlete_display_name,team_abbreviation,athlete_position_abbreviation,starter,did_not_play,reason,minutes,plus_minus,points,field_goals_made,field_goals_attempted,three_point_field_goals_made,three_point_field_goals_attempted,free_throws_made,free_throws_attempted,offensive_rebounds,defensive_rebounds,rebounds,assists,steals,blocks,turnovers,fouls
1001,1,Shai Gilgeous-Alexander,OKC,G,true,false,,34.5,+16,35,12,20,3,6,8,9,1,4,5,7,2,1,3,2
1001,2,Bench Guy,OKC,F,false,true,COACH'S DECISION,,,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0
1001,3,LeBron James,LAL,F,true,false,,36.0,-16,28,10,22,2,7,6,8,1,7,8,9,1,0,4,1
1003,3,LeBron James,LAL,F,true,false,STALE REASON,30.0,-10,20,8,18,1,5,3,4,0,5,5,6,1,0,2,2
`

// stubFetch replaces the downloader with fixtures keyed by URL. Any other URL
// behaves like a 404.
func stubFetch(t *testing.T, files map[string]string) {
	t.Helper()

	orig := dataset.Fetch
	dataset.Fetch = func(url string) (io.ReadCloser, error) {
		body, ok := files[url]
		if !ok {
			return nil, fmt.Errorf("%s: %w", url, dataset.ErrNotFound)
		}
		return io.NopCloser(strings.NewReader(body)), nil
	}
	t.Cleanup(func() { dataset.Fetch = orig })
}

// fakeData replaces the downloader with fixtures. Files listed in missing 404.
func fakeData(t *testing.T, missing ...string) {
	t.Helper()
	seasons.Reset()

	files := map[string]string{
		scheduleURL(2026):  scheduleCSV,
		teamBoxURL(2026):   teamBoxCSV,
		playerBoxURL(2026): playerBoxCSV,
	}
	for _, m := range missing {
		delete(files, m)
	}

	stubFetch(t, files)
	t.Cleanup(seasons.Reset)
}

func TestSeason(t *testing.T) {
	fakeData(t)

	s := NewSeason(2026)
	s.Parse()
	require.NoError(t, s.Err)

	// All-star teams are skipped.
	// and so are placeholder TBD teams.
	require.Len(t, s.Teams, 3)
	byAbbr := map[string]SeasonTeam{}
	for _, tm := range s.Teams {
		byAbbr[tm.ShortName] = tm
	}

	okc := byAbbr["OKC"]
	assert.Equal(t, "Oklahoma City Thunder", okc.Name)
	assert.EqualValues(t, 2, okc.Wins) // 1001, 1006; 1002 unplayed, 1005 postseason
	assert.EqualValues(t, 0, okc.Losses)
	assert.EqualValues(t, 225, okc.PointsFor)
	assert.EqualValues(t, 199, okc.PointsAgainst)
	assert.Equal(t, 26, okc.PointDifferential)

	gs := byAbbr["GS"]
	assert.EqualValues(t, 1, gs.Wins)
	assert.Equal(t, 1.0, gs.WinLossPercentage)

	lal := byAbbr["LAL"]
	assert.EqualValues(t, 0, lal.Wins)
	assert.EqualValues(t, 3, lal.Losses)
}

func TestTeam(t *testing.T) {
	fakeData(t)

	team := NewTeam("okc", 2026)
	team.Parse()
	require.NoError(t, team.Err)

	assert.Equal(t, "Oklahoma City Thunder", team.Name)
	assert.Equal(t, "OKC", team.ShortName)
	require.Len(t, team.Games, 4)
	assert.EqualValues(t, 2, team.Wins)

	first := team.Games[0]
	assert.Equal(t, "1001", first.ID)
	assert.Equal(t, "W", first.Result)
	assert.Equal(t, "home", first.Location)
	assert.Equal(t, "regular", first.SeasonType)
	assert.Equal(t, "LAL", first.OpponentShortName)
	assert.EqualValues(t, 125, first.TeamScore)

	future := team.Games[1]
	assert.Equal(t, "1002", future.ID)
	assert.Equal(t, "", future.Result)
	assert.Equal(t, "away", future.Location)

	assert.Equal(t, "postseason", team.Games[3].SeasonType)
}

func TestTeamUnknown(t *testing.T) {
	fakeData(t)

	team := NewTeam("XXX", 2026)
	team.Parse()
	assert.Error(t, team.Err)
}

func TestBoxscore(t *testing.T) {
	fakeData(t)

	b := NewBoxscore("1001", 2026)
	b.Parse()
	require.NoError(t, b.Err)

	assert.Equal(t, "regular", b.SeasonType)
	assert.Equal(t, "Oklahoma City Thunder", b.HomeTeam)
	assert.Equal(t, "OKC", b.HomeTeamShortName)
	assert.EqualValues(t, 125, b.HomeTeamScore)
	assert.Equal(t, 52, b.HomeTeamStats.PointsInPaint)
	assert.Equal(t, 28, b.HomeTeamStats.Assists)
	assert.Equal(t, 22, b.AwayTeamStats.Assists)
	assert.Equal(t, 2025, b.Date.Year())

	require.Len(t, b.Players, 3)
	sga := b.Players[0]
	assert.Equal(t, "Shai Gilgeous-Alexander", sga.Name)
	assert.True(t, sga.Starter)
	assert.False(t, sga.DidNotPlay)
	assert.Equal(t, 34.5, sga.Minutes)
	assert.Equal(t, 16, sga.PlusMinus)
	assert.Equal(t, 35, sga.Points)
	assert.Equal(t, 3, sga.ThreePointersMade)

	assert.True(t, b.Players[1].DidNotPlay)
	assert.Equal(t, "COACH'S DECISION", b.Players[1].DNPReason)
	assert.Equal(t, -16, b.Players[2].PlusMinus)
}

func TestBoxscoreStatsNotAvailable(t *testing.T) {
	fakeData(t)

	// 1006 is final in the schedule but has no box score rows yet.
	b := NewBoxscore("1006", 2026)
	b.Parse()
	assert.ErrorIs(t, b.Err, ErrStatsNotAvailable)
}

func TestBoxscoreStatsFilesMissing(t *testing.T) {
	fakeData(t, teamBoxURL(2026), playerBoxURL(2026))

	b := NewBoxscore("1001", 2026)
	b.Parse()
	assert.ErrorIs(t, b.Err, ErrStatsNotAvailable)

	// The schedule still works.
	team := NewTeam("OKC", 2026)
	team.Parse()
	require.NoError(t, team.Err)
	assert.Len(t, team.Games, 4)
}

func TestBoxscoreUnplayed(t *testing.T) {
	fakeData(t)

	b := NewBoxscore("1002", 2026)
	b.Parse()
	require.NoError(t, b.Err)
	assert.Equal(t, "GS", b.HomeTeamShortName)
	assert.Empty(t, b.Players)
}

func TestBoxscoreNotFound(t *testing.T) {
	fakeData(t)

	b := NewBoxscore("9999", 2026)
	b.Parse()
	assert.ErrorIs(t, b.Err, ErrGameNotFound)
}

func TestScheduleMissing(t *testing.T) {
	fakeData(t, scheduleURL(2026))

	s := NewSeason(2026)
	s.Parse()
	assert.ErrorIs(t, s.Err, dataset.ErrNotFound)
}

func TestDNPReasonOnlyWhenDidNotPlay(t *testing.T) {
	fakeData(t)

	b := NewBoxscore("1003", 2026)
	b.Parse()
	require.NoError(t, b.Err)
	require.Len(t, b.Players, 1)
	assert.Equal(t, "", b.Players[0].DNPReason)
}
