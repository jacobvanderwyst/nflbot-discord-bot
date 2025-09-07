package models

import (
	"fmt"
	"time"
)

// PlayerStats represents statistics for an NFL player
type PlayerStats struct {
	Name     string                 `json:"name"`
	Team     string                 `json:"team"`
	Position string                 `json:"position"`
	Season   int                    `json:"season"`
	Stats    map[string]interface{} `json:"stats"`
}

// GetStatsString returns a formatted string of player statistics
func (p *PlayerStats) GetStatsString() string {
	if p.Stats == nil {
		return "No stats available"
	}

	var statsStr string
	for key, value := range p.Stats {
		statsStr += fmt.Sprintf("%s: %v\n", key, value)
	}
	return statsStr
}

// TeamInfo represents information about an NFL team
type TeamInfo struct {
	Name         string   `json:"name"`
	City         string   `json:"city"`
	Conference   string   `json:"conference"`
	Division     string   `json:"division"`
	Coach        string   `json:"coach"`
	Stadium      string   `json:"stadium"`
	Founded      int      `json:"founded"`
	Championships int     `json:"championships"`
	Colors       []string `json:"colors"`
}

// Schedule represents a team's schedule
type Schedule struct {
	TeamName string `json:"team_name"`
	Season   int    `json:"season"`
	Games    []Game `json:"games"`
}

// Game represents a single NFL game
type Game struct {
	ID          string    `json:"id"`
	Week        int       `json:"week"`
	Season      int       `json:"season"`
	GameType    string    `json:"game_type"` // regular, playoff, preseason
	HomeTeam    string    `json:"home_team"`
	AwayTeam    string    `json:"away_team"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	GameTime    time.Time `json:"game_time"`
	Status      string    `json:"status"` // scheduled, in_progress, completed
	Stadium     string    `json:"stadium"`
	Weather     string    `json:"weather,omitempty"`
}

// IsLive returns true if the game is currently in progress
func (g *Game) IsLive() bool {
	return g.Status == "in_progress"
}

// IsCompleted returns true if the game has finished
func (g *Game) IsCompleted() bool {
	return g.Status == "completed"
}

// Winner returns the winning team name, or empty string if game is not completed
func (g *Game) Winner() string {
	if !g.IsCompleted() {
		return ""
	}
	
	if g.HomeScore > g.AwayScore {
		return g.HomeTeam
	} else if g.AwayScore > g.HomeScore {
		return g.AwayTeam
	}
	
	return "TIE"
}

// PlayerPosition represents different NFL positions
type PlayerPosition string

const (
	QB  PlayerPosition = "QB"  // Quarterback
	RB  PlayerPosition = "RB"  // Running Back
	WR  PlayerPosition = "WR"  // Wide Receiver
	TE  PlayerPosition = "TE"  // Tight End
	K   PlayerPosition = "K"   // Kicker
	DEF PlayerPosition = "DEF" // Defense
	// Add more positions as needed
)

// Conference represents NFL conferences
type Conference string

const (
	AFC Conference = "AFC"
	NFC Conference = "NFC"
)

// Division represents NFL divisions
type Division string

const (
	AFCEast  Division = "AFC East"
	AFCNorth Division = "AFC North"
	AFCSouth Division = "AFC South"
	AFCWest  Division = "AFC West"
	NFCEast  Division = "NFC East"
	NFCNorth Division = "NFC North"
	NFCSouth Division = "NFC South"
	NFCWest  Division = "NFC West"
)

// SeasonInfo represents current NFL season information
type SeasonInfo struct {
	Season     int    `json:"Season"`
	SeasonType string `json:"SeasonType"` // "REG", "POST", "PRE"
	Week       int    `json:"Week"`
}

// TeamStanding represents team standings information
type TeamStanding struct {
	Team       string `json:"Team"`
	Wins       int    `json:"Wins"`
	Losses     int    `json:"Losses"`
	Ties       int    `json:"Ties"`
	Percentage float64 `json:"Percentage"`
	Division   string `json:"Division"`
	Conference string `json:"Conference"`
}

// LiveScore represents a live game score
type LiveScore struct {
	GameID      string    `json:"GameID"`
	Season      int       `json:"Season"`
	Week        int       `json:"Week"`
	AwayTeam    string    `json:"AwayTeam"`
	HomeTeam    string    `json:"HomeTeam"`
	AwayScore   int       `json:"AwayScore"`
	HomeScore   int       `json:"HomeScore"`
	TimeRemaining string  `json:"TimeRemaining"`
	Quarter     string    `json:"Quarter"`
	Status      string    `json:"Status"`
	GameTime    time.Time `json:"DateTime"`
}

// IsLive returns true if the game is currently in progress
func (ls *LiveScore) IsLive() bool {
	return ls.Status == "InProgress" || ls.Status == "InProgress_Live"
}

// IsCompleted returns true if the game has finished
func (ls *LiveScore) IsCompleted() bool {
	return ls.Status == "Final" || ls.Status == "F" || ls.Status == "Completed"
}

// GetScoreString returns formatted score string
func (ls *LiveScore) GetScoreString() string {
	if ls.IsLive() {
		return fmt.Sprintf("%s %d - %d %s (%s, %s)", ls.AwayTeam, ls.AwayScore, ls.HomeScore, ls.HomeTeam, ls.Quarter, ls.TimeRemaining)
	} else if ls.IsCompleted() {
		return fmt.Sprintf("%s %d - %d %s (Final)", ls.AwayTeam, ls.AwayScore, ls.HomeScore, ls.HomeTeam)
	}
	return fmt.Sprintf("%s @ %s (Scheduled)", ls.AwayTeam, ls.HomeTeam)
}
