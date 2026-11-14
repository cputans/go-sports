package nfl

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cputans/go-sports/v2/internal/dataset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scheduleCSV = `game_id,season,game_type,week,gameday,gametime,away_team,away_score,home_team,home_score,overtime,pfr,spread_line,total_line,roof,surface,temp,wind
2026_01_GB_PHI,2026,REG,1,2026-09-13,13:00,GB,20,PHI,27,0,202609130phi,3.5,45.5,outdoors,grass,72,8
2026_02_LA_GB,2026,REG,2,2026-09-20,16:25,LA,,GB,,,202609200gnb,-2,47,dome,fieldturf,,
2025_01_GB_PHI,2025,REG,1,2025-09-07,13:00,GB,10,PHI,13,1,202509070phi,0,40,outdoors,grass,,
`

const playersCSV = `gsis_id,pfr_id
00-1,HurtJa00
00-2,LoveJo00
00-3,BarkSa00
`

const statsCSV = `player_id,player_display_name,position,game_id,team,completions,attempts,passing_yards,passing_tds,passing_interceptions,sacks_suffered,sack_yards_lost,carries,rushing_yards,rushing_tds,targets,receptions,receiving_yards,receiving_tds,rushing_fumbles,receiving_fumbles,sack_fumbles,def_sacks,def_tackles_solo,def_tackle_assists,fumble_recovery_own,fumble_recovery_opp
00-1,Jalen Hurts,QB,2026_01_GB_PHI,PHI,20,30,250,2,0,2,-14,5,30,1,0,0,0,0,0,0,1,0,0,0,0,0
00-2,Jordan Love,QB,2026_01_GB_PHI,GB,22,35,240,1,1,3,-20,2,5,0,0,0,0,0,0,0,0,0,0,0,0,0
00-3,Saquon Barkley,RB,2026_01_GB_PHI,PHI,0,0,0,0,0,0,0,20,100,1,4,3,25,0,0,0,0,0,0,0,0,0
00-9,New Rookie,LB,2026_01_GB_PHI,GB,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1.5,4,2,1,1
`

const snapsCSV = `game_id,pfr_game_id,season,game_type,week,player,pfr_player_id,position,team,opponent,offense_snaps,offense_pct,defense_snaps,defense_pct,st_snaps,st_pct
2026_01_GB_PHI,202609130phi,2026,REG,1,Jalen Hurts,HurtJa00,QB,PHI,GB,70,1,0,0,2,0.1
2026_01_GB_PHI,202609130phi,2026,REG,1,New Rookie,RookNe00,LB,GB,PHI,0,0,50,0.75,10,0.4
`

const pbpCSV = `game_id,play_type,yards_gained,pass_attempt,complete_pass,rush_attempt,sack,two_point_attempt,passer_player_id,receiver_player_id,rusher_player_id,posteam,play_deleted,first_down_rush,first_down_pass,first_down_penalty,fixed_drive,fixed_drive_result,drive_play_count,drive_time_of_possession,drive_ended_with_score,drive_yards_penalized,drive_start_yard_line
2026_01_GB_PHI,pass,12,1,1,0,0,0,00-1,00-3,,,,,,,,,,,,,
2026_01_GB_PHI,pass,41,1,1,0,0,0,00-1,00-3,,,,,,,,,,,,,
2026_01_GB_PHI,pass,0,1,0,0,0,0,00-1,00-3,,,,,,,,,,,,,
2026_01_GB_PHI,pass,-7,0,0,0,1,0,00-1,,,,,,,,,,,,,,
2026_01_GB_PHI,run,18,0,0,1,0,0,,,00-3,,,,,,,,,,,,
2026_01_GB_PHI,run,-2,0,0,1,0,0,,,00-3,,,,,,,,,,,,
2026_01_GB_PHI,run,2,0,0,1,0,1,,,00-3,,,,,,,,,,,,
2026_01_GB_PHI,kickoff,0,0,0,0,0,0,,,,GB,,0,0,0,1,Touchdown,6,3:30,1,0,PHI 25
2026_01_GB_PHI,run,5,0,0,1,0,0,,,00-3,PHI,,1,0,0,1,Touchdown,6,3:30,1,0,PHI 25
2026_01_GB_PHI,pass,20,1,1,0,0,0,00-1,00-3,,PHI,,0,1,0,1,Touchdown,6,3:30,1,0,PHI 25
2026_01_GB_PHI,no_play,5,0,0,0,0,0,,,,PHI,,0,0,1,1,Touchdown,6,3:30,1,0,PHI 25
2026_01_GB_PHI,run,9,0,0,1,0,0,,,00-3,PHI,1,1,0,0,1,Touchdown,6,3:30,1,0,PHI 25
2026_01_GB_PHI,pass,10,1,1,0,0,0,00-2,,,GB,,0,0,0,2,Punt,3,1:30,0,-5,GB 20
`

func gz(s string) string {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte(s))
	w.Close()
	return b.String()
}

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

// fakeNflverse replaces the downloader with fixtures. Files listed in missing 404.
func fakeNflverse(t *testing.T, missing ...string) {
	t.Helper()
	seasons.Reset()

	files := map[string]string{
		scheduleURL():  scheduleCSV,
		playersURL():   playersCSV,
		statsURL(2026): statsCSV,
		snapsURL(2026): snapsCSV,
		pbpURL(2026):   gz(pbpCSV),
	}
	for _, m := range missing {
		delete(files, m)
	}

	stubFetch(t, files)
	t.Cleanup(seasons.Reset)
}

func TestSeason(t *testing.T) {
	fakeNflverse(t)

	s := NewSeason(2026)
	s.Parse()
	require.NoError(t, s.Err)

	assert.Len(t, s.NFCTeams, 16)
	assert.Len(t, s.AFCTeams, 16)

	var phi, gb *SeasonTeam
	for i, tm := range s.NFCTeams {
		switch tm.ShortName {
		case "phi":
			phi = &s.NFCTeams[i]
		case "gnb":
			gb = &s.NFCTeams[i]
		}
	}
	require.NotNil(t, phi)
	require.NotNil(t, gb)
	assert.Equal(t, "Philadelphia Eagles", phi.Name)
	assert.EqualValues(t, 1, phi.Wins)
	assert.EqualValues(t, 27, phi.PointsFor)
	assert.EqualValues(t, 1, gb.Losses)
	assert.EqualValues(t, -7, gb.PointDifferential)
}

func TestTeam(t *testing.T) {
	fakeNflverse(t)

	// PFR abbreviations are accepted in any case.
	team := NewTeam("gnb", 2026)
	team.Parse()
	require.NoError(t, team.Err)

	assert.Equal(t, "Green Bay Packers", team.Name)
	require.Len(t, team.Games, 2)

	played := team.Games[0]
	assert.Equal(t, "202609130phi", played.ID)
	assert.Equal(t, "/boxscores/202609130phi.htm", played.BoxscoreLink)
	assert.Equal(t, "L", played.Result)
	assert.Equal(t, "away", played.Location)
	assert.Equal(t, "Philadelphia Eagles", played.HomeTeam)
	assert.Equal(t, "phi", played.HomeTeamShortName)
	assert.Equal(t, "gnb", played.AwayTeamShortName)

	future := team.Games[1]
	assert.Equal(t, "", future.Result)
	assert.Equal(t, "home", future.Location)
	assert.Equal(t, "ram", future.AwayTeamShortName)
}

func TestTeamUnknown(t *testing.T) {
	fakeNflverse(t)

	team := NewTeam("xyz", 2026)
	team.Parse()
	assert.Error(t, team.Err)
}

func TestBoxscore(t *testing.T) {
	fakeNflverse(t)

	b := NewBoxscore("/boxscores/202609130phi.htm")
	b.Parse()
	require.NoError(t, b.Err)

	assert.Equal(t, "202609130phi", b.ID)
	assert.Equal(t, "2026-09-13 13:00", b.Date.Format("2006-01-02 15:04"))
	assert.Equal(t, "Philadelphia Eagles", b.HomeTeam)
	assert.Equal(t, "PHI", b.HomeTeamShortName)
	assert.Equal(t, "Green Bay Packers", b.AwayTeam)
	assert.Equal(t, "GNB", b.AwayTeamShortName)
	assert.EqualValues(t, 27, b.HomeTeamScore)
	assert.Equal(t, "Philadelphia Eagles -3.5", b.VegasLine)
	assert.Equal(t, "72 degrees, wind 8 mph", b.Weather)

	assert.Equal(t, BoxscoreTeamStats{
		FirstDowns:  3,
		RushingInfo: "25-130-2", RushingAttempts: 25, RushingYards: 130, RushingTouchdowns: 2,
		PassingInfo: "20-30-250-2-0", PassingCompletions: 20, PassingAttempts: 30, PassingYards: 250, PassingTouchdowns: 2,
	}, b.HomeTeamStats)

	assert.Equal(t, "Sunday Sep 13, 2026", b.DateStr)
	assert.Equal(t, "Start Time: 1:00pm", b.TimeStr)
	assert.Equal(t, "/teams/phi/2026.htm", b.HomeTeamLink)
	assert.Equal(t, "/teams/gnb/2026.htm", b.AwayTeamLink)
	assert.EqualValues(t, 240, b.AwayTeamStats.PassingYards)
	assert.EqualValues(t, 1, b.AwayTeamStats.PassingInterceptions)
}

func TestBoxscorePlayers(t *testing.T) {
	fakeNflverse(t)

	b := NewBoxscore("202609130phi")
	b.Parse()
	require.NoError(t, b.Err)

	// Ids are PFR ids, so existing documents keep their ids.
	hurts, ok := b.Players["HurtJa00"]
	require.True(t, ok)
	assert.Equal(t, "Jalen Hurts", hurts.Name)
	assert.Equal(t, "PHI", hurts.TeamShortName)
	assert.EqualValues(t, 70, hurts.OffSnapCount)
	assert.Equal(t, "100%", hurts.OffSnapPercentage)
	assert.Equal(t, "10%", hurts.SpecialTeamsSnapPercentage)
	assert.EqualValues(t, 250, hurts.PassingYards)
	assert.EqualValues(t, 14, hurts.PassingSackedLostYards)
	assert.EqualValues(t, 41, hurts.PassingLong) // from play-by-play
	assert.EqualValues(t, 1, hurts.Fumbles)
	assert.InDelta(t, 114.6, hurts.PassingRating, 0.05)

	barkley := b.Players["BarkSa00"]
	assert.EqualValues(t, 18, barkley.RushingLong)
	assert.EqualValues(t, 41, barkley.ReceivingLong)
	assert.EqualValues(t, 3, barkley.ReceivingReceptions)

	// A player missing from the id table is matched to their snap row by name.
	rookie, ok := b.Players["RookNe00"]
	require.True(t, ok)
	assert.Equal(t, "GNB", rookie.TeamShortName)
	assert.EqualValues(t, 50, rookie.DefSnapCount)
	assert.Equal(t, 1.5, rookie.Sacks)
	assert.EqualValues(t, 6, rookie.TacklesCombined)
	assert.EqualValues(t, 2, rookie.FumblesRecovered)
	assert.Len(t, b.Players, 4) // Hurts, Love, Barkley, Rookie
}

func TestBoxscoreErrors(t *testing.T) {
	fakeNflverse(t)

	b := NewBoxscore("/boxscores/202609130zzz.htm")
	b.Parse()
	assert.ErrorIs(t, b.Err, ErrGameNotFound)

	b = NewBoxscore("garbage")
	b.Parse()
	assert.Error(t, b.Err)

	// Future games parse fine without stats.
	b = NewBoxscore("202609200gnb")
	b.Parse()
	require.NoError(t, b.Err)
	assert.Equal(t, "Los Angeles Rams -2.0", b.VegasLine)
}

func TestBoxscoreStatsNotYetPublished(t *testing.T) {
	fakeNflverse(t, statsURL(2026), snapsURL(2026), pbpURL(2026))

	b := NewBoxscore("202609130phi")
	b.Parse()
	assert.ErrorIs(t, b.Err, ErrStatsNotAvailable)
}

func TestSeasonOfPFRID(t *testing.T) {
	y, ok := seasonOfPFRID("202609130phi")
	assert.True(t, ok)
	assert.Equal(t, 2026, y)

	y, _ = seasonOfPFRID("202701100nyg") // January belongs to the prior season
	assert.Equal(t, 2026, y)
}

func TestPasserRating(t *testing.T) {
	assert.Equal(t, 158.3, passerRating(25, 30, 400, 5, 0))
	assert.Equal(t, 0.0, passerRating(0, 0, 0, 0, 0))
	assert.Equal(t, 2.8, passerRating(10, 30, 60, 0, 3))
}

func TestBoxscoreTables(t *testing.T) {
	fakeNflverse(t)

	b := NewBoxscore("202609130phi")
	b.Parse()
	require.NoError(t, b.Err)

	assert.Equal(t, "/players/H/HurtJa00.htm", b.Players["HurtJa00"].URL)

	// Snap tables are split by team; offense and defense tables only list
	// players with activity in that phase.
	require.Len(t, b.HomeSnapCounts, 1)
	assert.Equal(t, "HurtJa00", b.HomeSnapCounts[0].ID)
	require.Len(t, b.AwaySnapCounts, 1)
	assert.Equal(t, "RookNe00", b.AwaySnapCounts[0].ID)

	var off []string
	for _, p := range b.PlayerStatsOffense {
		off = append(off, p.ID)
	}
	assert.ElementsMatch(t, []string{"HurtJa00", "LoveJo00", "BarkSa00"}, off)

	require.Len(t, b.PlayerStatsDefense, 1)
	assert.Equal(t, "RookNe00", b.PlayerStatsDefense[0].ID)
}

func TestTeamStats(t *testing.T) {
	fakeNflverse(t)

	team := NewTeam("GNB", 2026)
	team.Parse()
	require.NoError(t, team.Err)

	assert.Equal(t, "https://www.pro-football-reference.com/teams/gnb/2026.htm", team.URL)
	assert.EqualValues(t, 20, team.PointsFor)
	assert.EqualValues(t, 27, team.PointsAgainst)
	assert.EqualValues(t, 27, team.OppOppPointsFor)

	// Packers offense: 240 pass - 20 sacked + 5 rush.
	assert.EqualValues(t, 225, team.Yards)
	assert.EqualValues(t, 40, team.OffensivePlays) // 35 att + 3 sacks + 2 rushes
	assert.InDelta(t, 5.625, team.YardsPerPlay, 1e-9)
	assert.EqualValues(t, 1, team.Turnovers)
	assert.InDelta(t, 220.0/38.0, team.PassingNetYardsPerAttempt, 1e-9)
	assert.InDelta(t, 2.5, team.RushingYardsPerAttempt, 1e-9)
	assert.EqualValues(t, 1, team.DriveCount)
	assert.Equal(t, "1:30", team.DriveAverageTime)
	assert.Equal(t, "Own 20.0", team.DriveAverageStart)
	assert.InDelta(t, 5.0, team.DriveAverageYards, 1e-9) // 10 yards less a 5 yard penalty
	assert.InDelta(t, 3.0, team.DriveAveragePlays, 1e-9)
	assert.InDelta(t, 0.0, team.DriveScoringPercentage, 1e-9)

	// Opponent (Eagles) offense.
	assert.EqualValues(t, 366, team.OppYards)
	assert.EqualValues(t, 57, team.OppOffensivePlays)
	assert.EqualValues(t, 3, team.OppFirstDowns)
	assert.EqualValues(t, 1, team.OppPassingFirstDowns)
	assert.EqualValues(t, 1, team.OppRushingFirstDowns)
	assert.EqualValues(t, 1, team.OppPenaltyFirstDowns)
	assert.EqualValues(t, 1, team.OppDriveCount)
	assert.Equal(t, "3:30", team.OppDriveAverageTime)
	assert.Equal(t, "Own 25.0", team.OppDriveAverageStart)
	assert.InDelta(t, 100.0, team.OppDriveScoringPercentage, 1e-9)
	assert.InDelta(t, 25.0, team.OppDriveAverageYards, 1e-9) // the deleted play is ignored
	assert.InDelta(t, 7.0, team.OppDriveAveragePoints, 1e-9)
}

func TestSeasonURLAndSchedule(t *testing.T) {
	fakeNflverse(t)

	s := NewSeason(2026)
	s.Parse()
	require.NoError(t, s.Err)

	for _, tm := range s.NFCTeams {
		if tm.ShortName == "phi" {
			assert.Equal(t, "/teams/phi/2026.htm", tm.URL)
			// One game, a win over Green Bay: ratings are centred on zero, so
			// each team's schedule strength is the other team's rating.
			assert.InDelta(t, -3.5, tm.StrengthOfSchedule, 1e-6)
		}
		if tm.ShortName == "gnb" {
			assert.InDelta(t, 3.5, tm.StrengthOfSchedule, 1e-6)
		}
	}
}

func TestStrengthOfSchedule(t *testing.T) {
	// A and B only play each other: ratings settle at +/-5 around zero.
	sos := strengthOfSchedule([]gameResult{{"A", "B", 10}})
	assert.InDelta(t, -5.0, sos["A"], 1e-6)
	assert.InDelta(t, 5.0, sos["B"], 1e-6)

	// Three-way round robin where everyone has the same margin: no differences.
	sos = strengthOfSchedule([]gameResult{{"A", "B", 3}, {"B", "C", 3}, {"C", "A", 3}})
	for _, v := range sos {
		assert.InDelta(t, 0.0, v, 1e-6)
	}
}
