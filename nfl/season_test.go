package nfl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSeason(t *testing.T) {
	s := NewSeason(2024)

	assert.Equal(t,
		s.URL,
		"https://www.pro-football-reference.com/years/2024/",
		fmt.Sprintf("expected URL = https://www.pro-football-reference.com/years/2024/, got %s", s.URL),
	)
}

func TestSeasonPostProcess(t *testing.T) {
	s := Season{
		NFCTeams: []SeasonTeam{
			{
				Name: "ABC 123*+",
				URL:  "/teams/abc/2024.htm",
			},
			{
				Name: "DEF 456",
				URL:  "/teams/def/2024.html",
			},
		},
		AFCTeams: []SeasonTeam{
			{
				Name: "ABC 123*+",
				URL:  "/teams/abc/2024.htm",
			},
			{
				Name: "DEF 456",
				URL:  "/teams/def/2024.html",
			},
		},
	}

	s.PostProcess()

	/* NFCTeams */
	assert.Equal(t,
		s.NFCTeams[0].Name,
		"ABC 123",
		fmt.Sprintf("expected Name = ABC 123, got %s", s.NFCTeams[0].Name),
	)

	assert.Equal(t,
		s.NFCTeams[0].ShortName,
		"abc",
		fmt.Sprintf("expected ShortName = abc, got %s", s.NFCTeams[0].Name),
	)

	assert.Equal(t,
		s.NFCTeams[1].Name,
		"DEF 456",
		fmt.Sprintf("expected Name = DEF 456, got %s", s.NFCTeams[1].Name),
	)

	assert.Equal(t,
		s.NFCTeams[1].ShortName,
		"def",
		fmt.Sprintf("expected ShortName = def, got %s", s.NFCTeams[1].ShortName),
	)

	/* AFCTeams */
	assert.Equal(t,
		s.AFCTeams[0].Name,
		"ABC 123",
		fmt.Sprintf("expected Name = ABC 123, got %s", s.AFCTeams[0].Name),
	)

	assert.Equal(t,
		s.AFCTeams[0].ShortName,
		"abc",
		fmt.Sprintf("expected ShortName = abc, got %s", s.AFCTeams[0].Name),
	)

	assert.Equal(t,
		s.AFCTeams[1].Name,
		"DEF 456",
		fmt.Sprintf("expected Name = DEF 456, got %s", s.AFCTeams[1].Name),
	)

	assert.Equal(t,
		s.AFCTeams[1].ShortName,
		"def",
		fmt.Sprintf("expected ShortName = def, got %s", s.AFCTeams[1].ShortName),
	)
}
