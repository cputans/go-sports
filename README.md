# go-sports

A go library to parse out sports statistical data.  Very much a work-in-progress and currently supporting only American professional football.

## Usage
```go
season := nfl.NewSeason(2024)
season.Parse()

for _, t := range season.NFCTeams {
    team := nfl.NewTeam(t.ShortName, 2024)
    team.Parse()

    for _, g := range team.Games {
        boxscore := nfl.NewBoxscore(g.BoxscoreLink)
        boxscore.Parse()
    }
}
```

## Page pull throttling
Due to restrictions pulling pages unthrottled, the environment variable `GOSPORTS_REQ_PAUSE` can be set with a number of seconds to pause between HTTP requests