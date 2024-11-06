package nfl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPlayer(t *testing.T) {
	p := NewPlayer("/abc/123")

	assert.Equal(t,
		p.URL,
		"https://www.pro-football-reference.com/abc/123",
		fmt.Sprintf("expected URL = https://www.pro-football-reference.com/abc/123, got %s", p.URL),
	)
}

func TestPlayerPostProcess(t *testing.T) {
	p := Player{
		Position: "Position: QB",
		URL:      "players/abc/Abc123.htm",
	}

	p.PostProcess()

	assert.Equal(t,
		p.Position,
		"QB",
		fmt.Sprintf("expected Position = QB, got %s", p.Position),
	)

	assert.Equal(t,
		p.ID,
		"Abc123",
		fmt.Sprintf("expected ID = Abc123, got %s", p.ID),
	)

	p.Position = "No Position"
	p.PostProcess()

	assert.Equal(t,
		p.Position,
		"",
		fmt.Sprintf("expected Position to be blank, got %s", p.Position),
	)
}
