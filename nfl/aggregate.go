package nfl

import "fmt"

// sideTotals is one team's offensive totals, summed over one or more games.
// A team's "opponent" stats are the same totals for the team it played.
type sideTotals struct {
	Points int

	PassCompletions, PassAttempts, PassYards, PassTDs, Interceptions float64
	Sacks, SackYards                                                 float64
	RushAttempts, RushYards, RushTDs                                 float64
	FumblesLost, Penalties, PenaltyYards                             float64

	FirstDownsRush, FirstDownsPass, FirstDownsPenalty int

	Drives, DriveScores, DriveTurnovers                         int
	DrivePlays, DriveSeconds, DriveYards, DrivePoints, DriveOwn float64
}

func (s *sideTotals) add(o sideTotals) {
	s.Points += o.Points
	s.PassCompletions += o.PassCompletions
	s.PassAttempts += o.PassAttempts
	s.PassYards += o.PassYards
	s.PassTDs += o.PassTDs
	s.Interceptions += o.Interceptions
	s.Sacks += o.Sacks
	s.SackYards += o.SackYards
	s.RushAttempts += o.RushAttempts
	s.RushYards += o.RushYards
	s.RushTDs += o.RushTDs
	s.FumblesLost += o.FumblesLost
	s.Penalties += o.Penalties
	s.PenaltyYards += o.PenaltyYards
	s.FirstDownsRush += o.FirstDownsRush
	s.FirstDownsPass += o.FirstDownsPass
	s.FirstDownsPenalty += o.FirstDownsPenalty
	s.Drives += o.Drives
	s.DriveScores += o.DriveScores
	s.DriveTurnovers += o.DriveTurnovers
	s.DrivePlays += o.DrivePlays
	s.DriveSeconds += o.DriveSeconds
	s.DriveYards += o.DriveYards
	s.DrivePoints += o.DrivePoints
	s.DriveOwn += o.DriveOwn
}

// gameSide computes a team's totals for one game.
func (d *seasonData) gameSide(g *gameRow, team string) sideTotals {
	var s sideTotals

	info := findTeam(team)
	if info == nil {
		return s
	}

	if findTeam(g.Home) == info {
		s.Points = g.HomeScore
	} else {
		s.Points = g.AwayScore
	}

	for _, r := range d.stats[g.ID] {
		if findTeam(r.Team) != info {
			continue
		}
		s.PassCompletions += r.PassCompletions
		s.PassAttempts += r.PassAttempts
		s.PassYards += r.PassYards
		s.PassTDs += r.PassTDs
		s.Interceptions += r.PassInts
		s.Sacks += r.Sacked
		s.SackYards += r.SackedYards
		s.RushAttempts += r.RushAttempts
		s.RushYards += r.RushYards
		s.RushTDs += r.RushTDs
		s.FumblesLost += r.FumblesLost
		s.Penalties += r.Penalties
		s.PenaltyYards += r.PenaltyYards
	}

	if p := d.pbp[g.ID][info.PFR]; p != nil {
		s.FirstDownsRush = p.FirstDownsRush
		s.FirstDownsPass = p.FirstDownsPass
		s.FirstDownsPenalty = p.FirstDownsPenalty

		for _, dr := range p.Drives {
			s.Drives++
			if dr.Score {
				s.DriveScores++
			}
			if dr.Result == "Turnover" || dr.Result == "Opp touchdown" {
				s.DriveTurnovers++
			}
			s.DrivePlays += float64(dr.Plays)
			s.DriveSeconds += float64(dr.Seconds)
			s.DriveYards += float64(dr.Yards)
			s.DrivePoints += float64(dr.Points)
			s.DriveOwn += dr.StartOwn
		}
	}

	return s
}

func (s sideTotals) firstDowns() int {
	return s.FirstDownsRush + s.FirstDownsPass + s.FirstDownsPenalty
}

// plays follows PFR: pass attempts, sacks and rush attempts.
func (s sideTotals) plays() float64 { return s.PassAttempts + s.Sacks + s.RushAttempts }

// yards is net yards: passing yards less sack yards, plus rushing yards.
func (s sideTotals) yards() float64 { return s.PassYards - s.SackYards + s.RushYards }

func ratio(n, d float64) float64 {
	if d == 0 {
		return 0
	}
	return n / d
}

func (s sideTotals) yardsPerPlay() float64 { return ratio(s.yards(), s.plays()) }
func (s sideTotals) netYardsPerAttempt() float64 {
	return ratio(s.PassYards-s.SackYards, s.PassAttempts+s.Sacks)
}
func (s sideTotals) rushYardsPerAttempt() float64 { return ratio(s.RushYards, s.RushAttempts) }
func (s sideTotals) turnovers() float64           { return s.Interceptions + s.FumblesLost }

func (s sideTotals) driveScoringPct() float64 {
	return ratio(float64(s.DriveScores), float64(s.Drives)) * 100
}
func (s sideTotals) driveTurnoverPct() float64 {
	return ratio(float64(s.DriveTurnovers), float64(s.Drives)) * 100
}
func (s sideTotals) driveAvgPlays() float64  { return ratio(s.DrivePlays, float64(s.Drives)) }
func (s sideTotals) driveAvgYards() float64  { return ratio(s.DriveYards, float64(s.Drives)) }
func (s sideTotals) driveAvgPoints() float64 { return ratio(s.DrivePoints, float64(s.Drives)) }

// driveAvgTime formats the average time of possession per drive as m:ss.
func (s sideTotals) driveAvgTime() string {
	if s.Drives == 0 {
		return ""
	}
	secs := int(ratio(s.DriveSeconds, float64(s.Drives)) + 0.5)
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

// driveAvgStart formats the average starting field position, e.g. "Own 28.3"
// or "Opp 47.0".
func (s sideTotals) driveAvgStart() string {
	if s.Drives == 0 {
		return ""
	}
	own := ratio(s.DriveOwn, float64(s.Drives))
	if own <= 50 {
		return fmt.Sprintf("Own %.1f", own)
	}
	return fmt.Sprintf("Opp %.1f", 100-own)
}
