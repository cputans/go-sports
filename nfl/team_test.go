package nfl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTeam(t *testing.T) {
	e := NewTeam("abc", 2024)

	assert.Equal(t,
		e.URL,
		"https://www.pro-football-reference.com/teams/abc/2024.htm",
		fmt.Sprintf("expected URL = https://www.pro-football-reference.com/teams/abc/2024.htm, got %s", e.URL),
	)

	assert.Equal(t,
		e.ShortName,
		"abc",
		fmt.Sprintf("expected ShortName = abc, got %s", e.ShortName),
	)
}

func TestTeamPostProcess(t *testing.T) {
	e := Team{
		Name:      "ABC 123",
		ShortName: "abc",
		Games: []TeamGame{
			{
				BoxscoreLink: "",
			},
			{
				Opponent:     "DEF 456",
				OpponentLink: "/teams/def/2024.htm",
				BoxscoreLink: "/boxscores/abcd1234.htm",
				Location:     "@",
			},
			{
				Opponent:     "GHI 789",
				OpponentLink: "/teams/ghi/2024.htm",
				BoxscoreLink: "/boxscores/efgh5678.htm",
				Location:     "",
			},
		},
	}

	e.PostProcess()

	assert.Equal(t,
		len(e.Games),
		2,
		fmt.Sprintf("expected list of games to contain 2 elements, got %d", len(e.Games)),
	)

	/* Away Game */
	assert.Equal(t,
		e.Games[0].Location,
		"away",
		fmt.Sprintf("expected game 1 location to be away, got %s", e.Games[0].Location),
	)

	assert.Equal(t,
		e.Games[0].HomeTeam,
		"DEF 456",
		fmt.Sprintf("expected game 1 HomeTeam to be DEF 456, got %s", e.Games[0].HomeTeam),
	)

	assert.Equal(t,
		e.Games[0].HomeTeamShortName,
		"def",
		fmt.Sprintf("expected game 1 HomeTeamShortName to be def, got %s", e.Games[0].HomeTeamShortName),
	)

	assert.Equal(t,
		e.Games[0].AwayTeam,
		"ABC 123",
		fmt.Sprintf("expected game 1 AwayTeam to be ABC 123, got %s", e.Games[0].AwayTeam),
	)

	assert.Equal(t,
		e.Games[0].AwayTeamShortName,
		"abc",
		fmt.Sprintf("expected game 1 AwayTeamShortName to be abc, got %s", e.Games[0].AwayTeamShortName),
	)

	/* Home Game */
	assert.Equal(t,
		e.Games[1].Location,
		"home",
		fmt.Sprintf("expected game 2 location to be home, got %s", e.Games[1].Location),
	)

	assert.Equal(t,
		e.Games[1].HomeTeam,
		"ABC 123",
		fmt.Sprintf("expected game 2 HomeTeam to be ABC 123, got %s", e.Games[1].HomeTeam),
	)

	assert.Equal(t,
		e.Games[1].HomeTeamShortName,
		"abc",
		fmt.Sprintf("expected game 2 HomeTeamShortName to be abc, got %s", e.Games[1].HomeTeamShortName),
	)

	assert.Equal(t,
		e.Games[1].AwayTeam,
		"GHI 789",
		fmt.Sprintf("expected game 2 AwayTeam to be DEF 456, got %s", e.Games[1].AwayTeam),
	)

	assert.Equal(t,
		e.Games[1].AwayTeamShortName,
		"ghi",
		fmt.Sprintf("expected game 2 AwayTeamShortName to be def, got %s", e.Games[1].AwayTeamShortName),
	)
}
