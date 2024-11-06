package nfl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cputans/go-sports/internal"
)

type Player struct {
	Name     string `fullSelector:"#meta div:not(.media-item) h1:nth-child(1) span"`
	ID       string
	Position string `fullSelector:"#meta div:nth-child(2) p:nth-child(3)"`
	URL      string
}

func NewPlayer(url string) *Player {
	playerUrl := fmt.Sprintf(BASE_URL, url)

	return &Player{
		URL: playerUrl,
	}
}

func (p *Player) Parse() {
	internal.Parse(p, p.URL)
}

func (p *Player) PostProcess() {
	/* Extract position */
	p.Position = strings.TrimSpace(p.Position)
	positionRegex := regexp.MustCompile(`Position: ([^\s]+)`)
	matches := positionRegex.FindStringSubmatch(p.Position)
	if matches != nil {
		p.Position = strings.TrimSpace(matches[1])
	} else {
		p.Position = ""
	}

	/* ID */
	idRegex := regexp.MustCompile("players/[A-Za-z0-9]+/([A-Za-z0-9.]+).htm")
	idMatches := idRegex.FindStringSubmatch(p.URL)
	if idMatches != nil {
		p.ID = idMatches[1]
	}
}
