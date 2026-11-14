# go-sports

A go library to parse out sports statistical data.  Very much a work-in-progress, currently supporting American professional football (`nfl`) and the NBA (`nba`).

Both packages follow the same flow: build a `Season`, then a `Team` for each team in it, then a `Boxscore` for each game on the team's schedule. Each has a `Parse()` method that fills it in and sets `Err` if it couldn't be loaded.

## NFL
Data comes from the [nflverse](https://github.com/nflverse/nflverse-data) project, which publishes NFL data as static files on GitHub releases. Teams use lower-case Pro-Football-Reference abbreviations (`gnb`, `nwe`, ...), and games are identified by PFR-style boxscore links (`/boxscores/202609130phi.htm`).

```go
season := nfl.NewSeason(2024)
season.Parse()

for _, t := range season.NFCTeams {
    team := nfl.NewTeam(t.ShortName, 2024)
    team.Parse()

    for _, g := range team.Games {
        boxscore := nfl.NewBoxscore(g.BoxscoreLink)
        boxscore.Parse() // boxscore.Err is nfl.ErrStatsNotAvailable until published
    }
}
```

## NBA
Data comes from the [sportsdataverse](https://github.com/sportsdataverse/hoopR-nba-data) project (CC BY 4.0), which publishes ESPN-derived box scores as static files on GitHub releases. Teams use ESPN abbreviations (`GS`, `NY`, `UTAH`, ...), games are identified by ESPN game IDs, and seasons are named for the year they end in (2027 is 2026-27).

```go
season := nba.NewSeason(2027)
season.Parse()

for _, t := range season.Teams {
    team := nba.NewTeam(t.ShortName, 2027)
    team.Parse()

    for _, g := range team.Games {
        boxscore := nba.NewBoxscore(g.ID, 2027)
        boxscore.Parse() // boxscore.Err is nba.ErrStatsNotAvailable until published
    }
}
```

## Errors
`ErrStatsNotAvailable` and `ErrGameNotFound` are exported by both packages and are the same errors, so `errors.Is` works with either. A game that has finished may not have its box score published yet; leave it and retry on a later run. A game that hasn't been played (`TeamGame.Result == ""`) has no stats to load.

## Page pull throttling
Due to restrictions pulling pages unthrottled, the environment variable `GOSPORTS_REQ_PAUSE` can be set with a number of seconds to pause between HTTP requests. It only applies to pages that are still scraped (NFL player pages); the nflverse and sportsdataverse downloads aren't affected.
