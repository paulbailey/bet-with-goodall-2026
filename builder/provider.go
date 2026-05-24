package main

import "time"

// GroupStanding holds the live standings for one group, as returned by the provider.
// Teams are sorted by standing (1st to 4th).
type GroupStanding struct {
	Group string
	Teams []TeamStats
}

type TeamStats struct {
	Name   string
	Played int
	Won    int
	Drawn  int
	Lost   int
	GF     int
	GA     int
	GD     int
	Points int
}

type Match struct {
	UtcDate  time.Time
	Status   string // SCHEDULED | IN_PLAY | PAUSED | FINISHED
	Stage    string // GROUP_STAGE | LAST_16 | QUARTER_FINALS | ...
	Group    string // GROUP_A, GROUP_B, ... (empty for knockout rounds)
	HomeTeam string
	AwayTeam string
}

type TopScorerEntry struct {
	Player string
	Team   string
	Goals  int
	Games  int
}
