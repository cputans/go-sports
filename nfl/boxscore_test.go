package nfl

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBoxscore(t *testing.T) {
	b := NewBoxscore("/boxscores/abcd1234.htm")

	assert.Equal(t,
		b.URL,
		"https://www.pro-football-reference.com/boxscores/abcd1234.htm",
		fmt.Sprintf("expected URL = https://www.pro-football-reference.com/boxscores/abcd1234.htm, got %s", b.URL),
	)
}

func TestBoxscorePostProcess(t *testing.T) {
	b := Boxscore{
		URL:          "https://www.pro-football-reference.com/boxscores/abcd1234.htm",
		TimeStr:      "Start Time: 1:00pm",
		DateStr:      "Sunday Nov 3, 2024",
		HomeTeamLink: "/teams/abc/2024.htm",
		HomeTeamStats: BoxscoreTeamStats{
			RushingInfo: "10-20-0",
			PassingInfo: "10-20-300-1-2",
		},
		AwayTeamLink:  "/teams/def/2024.htm",
		AwayTeamStats: BoxscoreTeamStats{},
		PlayerStatsOffense: []BoxscorePlayerOffensiveStats{
			{
				URL: "players/b/billdinkens.htm",
			},
		},
		PlayerStatsDefense: []BoxscorePlayerDefensiveStats{
			{
				URL: "players/p/philbertjohnson.htm",
			},
		},
	}

	b.PostProcess()

	assert.Equal(t,
		b.Date,
		time.Date(2024, 11, 3, 13, 0, 0, 0, time.Now().UTC().Location()),
		"expected Date = 2024-11-03 13:00:00 +0000 UTC, got %s", b.Date,
	)

	assert.Equal(t,
		b.ID,
		"abcd1234",
		fmt.Sprintf("expected ID = abc1234, got %s", b.ID),
	)

	/* Rushing stats */
	assert.Equal(t,
		b.HomeTeamStats.RushingAttempts,
		uint(10),
		fmt.Sprintf("expected RushingAttempts = 10, got %d", b.HomeTeamStats.RushingAttempts),
	)

	assert.Equal(t,
		b.HomeTeamStats.RushingYards,
		20,
		fmt.Sprintf("expected RushingYards = 20, got %d", b.HomeTeamStats.RushingYards),
	)

	assert.Equal(t,
		b.HomeTeamStats.RushingTouchdowns,
		uint(0),
		fmt.Sprintf("expected RushingTouchdowns = 10, got %d", b.HomeTeamStats.RushingTouchdowns),
	)

	/* Passing stats */
	assert.Equal(t,
		b.HomeTeamStats.PassingCompletions,
		uint(10),
		fmt.Sprintf("expected PassingCompletions = 10, got %d", b.HomeTeamStats.PassingCompletions),
	)

	assert.Equal(t,
		b.HomeTeamStats.PassingAttempts,
		uint(20),
		fmt.Sprintf("expected PassingAttempts = 20, got %d", b.HomeTeamStats.PassingAttempts),
	)

	assert.Equal(t,
		b.HomeTeamStats.PassingYards,
		300,
		fmt.Sprintf("expected PassingYards = 10, got %d", b.HomeTeamStats.PassingYards),
	)

	assert.Equal(t,
		b.HomeTeamStats.PassingTouchdowns,
		uint(1),
		fmt.Sprintf("expected PassingTouchdowns = 20, got %d", b.HomeTeamStats.PassingTouchdowns),
	)

	assert.Equal(t,
		b.HomeTeamStats.PassingInterceptions,
		uint(2),
		fmt.Sprintf("expected PassingInterceptions = 10, got %d", b.HomeTeamStats.PassingInterceptions),
	)

	/* Players */
	assert.Equal(t,
		b.PlayerStatsOffense[0].ID,
		"billdinkens",
		fmt.Sprintf("expected offensive player 1 ID = billdinkens, got %s", b.PlayerStatsOffense[0].ID),
	)

	assert.Equal(t,
		b.PlayerStatsDefense[0].ID,
		"philbertjohnson",
		fmt.Sprintf("expected defensive player 1 ID = philbertjohnson, got %s", b.PlayerStatsOffense[0].ID),
	)

}
