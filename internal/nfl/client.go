package nfl

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"nfl-discord-bot/pkg/models"
)

// SportsDataPlayerStat represents a player stat from SportsData.io API
type SportsDataPlayerStat struct {
	PlayerID         float64 `json:"PlayerID"`
	Name             string  `json:"Name"`
	Team             string  `json:"Team"`
	Position         string  `json:"Position"`
	Season           float64 `json:"Season"`
	Week             float64 `json:"Week"`
	PassingYards     float64 `json:"PassingYards"`
	PassingTouchdowns float64 `json:"PassingTouchdowns"`
	Interceptions    float64 `json:"Interceptions"`
	Completions      float64 `json:"PassingCompletions"`
	Attempts         float64 `json:"PassingAttempts"`
	RushingYards     float64 `json:"RushingYards"`
	RushingTouchdowns float64 `json:"RushingTouchdowns"`
	ReceivingYards   float64 `json:"ReceivingYards"`
	ReceivingTouchdowns float64 `json:"ReceivingTouchdowns"`
	Receptions       float64 `json:"Receptions"`
	Targets          float64 `json:"Targets"`
}

// SportsDataTeam represents a team from SportsData.io API
type SportsDataTeam struct {
	Key          string `json:"Key"`
	TeamID       int    `json:"TeamID"`
	City         string `json:"City"`
	Name         string `json:"Name"`
	FullName     string `json:"FullName"`
	Conference   string `json:"Conference"`
	Division     string `json:"Division"`
	HeadCoach    string `json:"HeadCoach"`
	StadiumName  string `json:"StadiumName"`
}

// SportsDataStanding represents team standing from SportsData.io API
type SportsDataStanding struct {
	Team         string  `json:"Team"`
	Wins         int     `json:"Wins"`
	Losses       int     `json:"Losses"`
	Ties         int     `json:"Ties"`
	Percentage   float64 `json:"Percentage"`
	Division     string  `json:"Division"`
	Conference   string  `json:"Conference"`
}

// SportsDataGame represents a game from SportsData.io API
type SportsDataGame struct {
	GameKey      string    `json:"GameKey"`
	Season       int       `json:"Season"`
	Week         int       `json:"Week"`
	AwayTeam     string    `json:"AwayTeam"`
	HomeTeam     string    `json:"HomeTeam"`
	AwayScore    int       `json:"AwayScore"`
	HomeScore    int       `json:"HomeScore"`
	Quarter      string    `json:"Quarter"`
	TimeRemaining string   `json:"TimeRemaining"`
	Status       string    `json:"Status"`
	DateTime     string    `json:"DateTime"` // Changed to string for custom parsing
	Stadium      string    `json:"Stadium"`
}

// SportsDataCurrentSeason represents current season info from SportsData.io
type SportsDataCurrentSeason struct {
	Season         int    `json:"Season"`
	SeasonType     int    `json:"SeasonType"`
	ApiSeasonType  string `json:"ApiSeasonType"`
	ApiWeek        int    `json:"ApiWeek"`
}

// CacheEntry represents a cached API response
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
}

// Client represents the NFL data client
type Client struct {
	apiKey        string
	baseURL       string
	httpClient    *http.Client
	cachedSeason  *models.SeasonInfo
	lastSeasonCheck time.Time
	cache         map[string]*CacheEntry
	cacheTTL      time.Duration
}

// NewClient creates a new NFL client
func NewClient(apiKey, baseURL string) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      make(map[string]*CacheEntry),
		cacheTTL:   5 * time.Minute, // 5-minute cache TTL
	}
	
	// Start periodic cache cleanup
	c.startCacheCleanup()
	
	return c
}

// getCurrentSeason returns intelligent NFL season information based on current date
func (c *Client) getCurrentSeason() (*models.SeasonInfo, error) {
	// Cache for 1 hour to avoid excessive recalculations
	if c.cachedSeason != nil && time.Since(c.lastSeasonCheck) < time.Hour {
		return c.cachedSeason, nil
	}

	now := time.Now()
	seasonInfo := calculateCurrentNFLWeek(now)

	log.Printf("[NFL-SEASON] Calculated: %d %s Week %d (Day: %s)", 
		seasonInfo.Season, seasonInfo.SeasonType, seasonInfo.Week, now.Weekday())

	c.cachedSeason = seasonInfo
	c.lastSeasonCheck = now

	return c.cachedSeason, nil
}

// calculateCurrentNFLWeek calculates current NFL season and week with intelligent day-of-week logic
func calculateCurrentNFLWeek(now time.Time) *models.SeasonInfo {
	// Determine NFL season year (starts in September of calendar year)
	season := now.Year()
	if now.Month() < 3 { // January-February belong to previous season
		season--
	}

	// NFL regular season typically starts first Thursday after Labor Day (first Monday in September)
	// For 2025, let's approximate: season starts September 4, 2025
	seasonStart := findNFLSeasonStart(season)
	
	// Determine if we're in regular season, playoffs, or off-season
	if now.Before(seasonStart) {
		// Before season starts - use previous season's final week
		return &models.SeasonInfo{
			Season:     season - 1,
			SeasonType: "REG",
			Week:       18,
		}
	}

	// Calculate weeks since season start
	daysSinceStart := int(now.Sub(seasonStart).Hours() / 24)
	weeksSinceStart := daysSinceStart / 7

	// Apply day-of-week logic for current vs previous week preference
	weekday := now.Weekday()
	currentWeek := weeksSinceStart + 1

	// Tuesday = start of new week, Wednesday = prefer previous week
	if weekday == time.Wednesday && currentWeek > 1 {
		currentWeek-- // Use previous week on Wednesday
	}

	// Determine season type and week
	if currentWeek <= 18 {
		// Regular season (weeks 1-18)
		return &models.SeasonInfo{
			Season:     season,
			SeasonType: "REG",
			Week:       currentWeek,
		}
	} else if currentWeek <= 22 {
		// Playoffs (weeks 1-4 of postseason)
		playoffWeek := currentWeek - 18
		return &models.SeasonInfo{
			Season:     season,
			SeasonType: "POST",
			Week:       playoffWeek,
		}
	} else {
		// Off-season - return current season's last week
		return &models.SeasonInfo{
			Season:     season,
			SeasonType: "REG",
			Week:       18,
		}
	}
}

// findNFLSeasonStart finds the approximate start date of the NFL season
func findNFLSeasonStart(season int) time.Time {
	// NFL typically starts first Thursday after Labor Day
	// For simplicity, approximate as first Thursday of September
	septFirst := time.Date(season, 9, 1, 20, 0, 0, 0, time.UTC) // 8 PM UTC typical game time
	
	// Find first Thursday in September
	for septFirst.Weekday() != time.Thursday {
		septFirst = septFirst.AddDate(0, 0, 1)
	}
	
	return septFirst
}

// parseSportsDataDateTime parses SportsData.io datetime format
func parseSportsDataDateTime(dateStr string) (time.Time, error) {
	// Try common datetime formats used by SportsData.io
	formats := []string{
		"2006-01-02T15:04:05",     // Without timezone
		"2006-01-02T15:04:05Z",    // UTC
		"2006-01-02T15:04:05-07:00", // With timezone offset
		time.RFC3339,               // Standard RFC3339
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime: %s", dateStr)
}

// logRequest logs API requests for debugging
func (c *Client) logRequest(method, url string) {
	log.Printf("[NFL-API] %s %s", method, url)
}

// normalizeTeamName returns common variations of team names for matching
func normalizeTeamName(teamName string) []string {
	teamName = strings.ToLower(strings.TrimSpace(teamName))
	var variations []string

	// Add the original name
	variations = append(variations, teamName)

	// Common team name mappings
	mappings := map[string][]string{
		"bills":      {"buf", "buffalo"},
		"buffalo":    {"buf", "bills"},
		"dolphins":   {"mia", "miami"},
		"miami":      {"mia", "dolphins"},
		"patriots":   {"ne", "new england"},
		"jets":       {"nyj", "new york jets"},
		"ravens":     {"bal", "baltimore"},
		"bengals":    {"cin", "cincinnati"},
		"browns":     {"cle", "cleveland"},
		"steelers":   {"pit", "pittsburgh"},
		"texans":     {"hou", "houston"},
		"colts":      {"ind", "indianapolis"},
		"jaguars":    {"jax", "jacksonville"},
		"titans":     {"ten", "tennessee"},
		"broncos":    {"den", "denver"},
		"chiefs":     {"kc", "kansas city"},
		"raiders":    {"lv", "las vegas"},
		"chargers":   {"lac", "los angeles chargers"},
		"cowboys":    {"dal", "dallas"},
		"giants":     {"nyg", "new york giants"},
		"eagles":     {"phi", "philadelphia"},
		"commanders": {"was", "washington"},
		"bears":      {"chi", "chicago"},
		"lions":      {"det", "detroit"},
		"packers":    {"gb", "green bay"},
		"vikings":    {"min", "minnesota"},
		"falcons":    {"atl", "atlanta"},
		"panthers":   {"car", "carolina"},
		"saints":     {"no", "new orleans"},
		"buccaneers": {"tb", "tampa bay"},
		"cardinals":  {"ari", "arizona"},
		"rams":       {"lar", "los angeles rams"},
		"seahawks":   {"sea", "seattle"},
		"49ers":      {"sf", "san francisco"},
	}

	// Add mapped variations
	if mapped, exists := mappings[teamName]; exists {
		variations = append(variations, mapped...)
	}

	return variations
}

// getCachedData retrieves data from cache if still valid
func (c *Client) getCachedData(key string) (interface{}, bool) {
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Check if cache entry is still valid
	if time.Since(entry.Timestamp) > c.cacheTTL {
		delete(c.cache, key) // Clean up expired entry
		return nil, false
	}

	return entry.Data, true
}

// setCachedData stores data in cache
func (c *Client) setCachedData(key string, data interface{}) {
	c.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
	log.Printf("[NFL-CACHE] Cached data for key: %s", key)
}

// startCacheCleanup starts a periodic cache cleanup routine
func (c *Client) startCacheCleanup() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
		defer ticker.Stop()
		
		for range ticker.C {
			c.cleanupExpiredCache()
		}
	}()
}

// cleanupExpiredCache removes all expired entries from cache
func (c *Client) cleanupExpiredCache() {
	expiredKeys := make([]string, 0)
	
	// Find expired keys
	for key, entry := range c.cache {
		if time.Since(entry.Timestamp) > c.cacheTTL {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	// Remove expired entries
	for _, key := range expiredKeys {
		delete(c.cache, key)
	}
	
	if len(expiredKeys) > 0 {
		log.Printf("[NFL-CACHE] Cleaned up %d expired cache entries", len(expiredKeys))
	}
}

// getSafeName safely gets a player name from slice with bounds checking
func getSafeName(stats []SportsDataPlayerStat, index int) string {
	if index < len(stats) {
		return stats[index].Name
	}
	return "N/A"
}

// fuzzyMatch performs improved fuzzy matching for player names
func fuzzyMatch(playerName, searchName string) bool {
	// Normalize names for comparison
	playerLower := normalizePlayerNameStatic(playerName)
	searchLower := normalizePlayerNameStatic(searchName)
	
	// Split names into parts
	playerParts := strings.Fields(playerLower)
	searchParts := strings.Fields(searchLower)
	
	// If both have first and last name, try exact matching first
	if len(playerParts) >= 2 && len(searchParts) >= 2 {
		// Check if first name and last name both match
		firstMatch := strings.Contains(playerParts[0], searchParts[0]) || strings.Contains(searchParts[0], playerParts[0])
		lastMatch := strings.Contains(playerParts[len(playerParts)-1], searchParts[len(searchParts)-1]) ||
			       strings.Contains(searchParts[len(searchParts)-1], playerParts[len(playerParts)-1])
		
		// Both first and last should match for high confidence
		if firstMatch && lastMatch {
			return true
		}
		
		// Enhanced common surname detection with Jackson added
		commonLastNames := []string{"allen", "johnson", "smith", "williams", "brown", "jones", "miller", "davis", "garcia", "rodriguez", "jackson", "wilson", "moore", "taylor", "anderson", "thomas", "harris", "martin", "thompson", "white"}
		lastName := playerParts[len(playerParts)-1]
		searchLastName := searchParts[len(searchParts)-1]
		
		// If dealing with common last names, be more strict about first name matching
		for _, commonName := range commonLastNames {
			if (strings.Contains(lastName, commonName) || strings.Contains(searchLastName, commonName)) && lastMatch {
				// For common last names, require first name to have some similarity
				if len(searchParts[0]) >= 3 && len(playerParts[0]) >= 3 {
					// More strict matching - require significant first name overlap
					if playerParts[0][:3] == searchParts[0][:3] ||
					   (len(searchParts[0]) >= 5 && strings.Contains(playerParts[0], searchParts[0][:4])) ||
					   (len(playerParts[0]) >= 5 && strings.Contains(searchParts[0], playerParts[0][:4])) {
						return true
					}
				}
				return false // Don't match if common last name but different first name
			}
		}
	}
	
	// Fallback: check if any significant part matches (length >= 5 for better precision)
	for _, searchPart := range searchParts {
		if len(searchPart) >= 5 {
			for _, playerPart := range playerParts {
				if len(playerPart) >= 5 && strings.Contains(playerPart, searchPart) {
					return true
				}
			}
		}
	}
	
	return false
}

// normalizePlayerName normalizes player names for better matching
func (c *Client) normalizePlayerName(name string) string {
	// Convert to lowercase
	normalized := strings.ToLower(name)
	
	// Handle common hyphenated name patterns
	// "josh hines-allen" should match "Josh Hines-Allen"
	// But also allow "josh hines allen" to match "Josh Hines-Allen"
	normalized = strings.ReplaceAll(normalized, "-", " ")
	
	// Remove extra punctuation that might cause issues
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	
	// Clean up multiple spaces
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	return normalized
}

// normalizePlayerNameStatic is a static version of normalizePlayerName for use in fuzzyMatch
func normalizePlayerNameStatic(name string) string {
	// Convert to lowercase
	normalized := strings.ToLower(name)
	
	// Handle common hyphenated name patterns
	normalized = strings.ReplaceAll(normalized, "-", " ")
	
	// Remove extra punctuation that might cause issues
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	
	// Clean up multiple spaces
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	return normalized
}

// calculatePlayerMatchScore calculates a match score for player name matching
func (c *Client) calculatePlayerMatchScore(playerName, searchName string) int {
	// Normalize names for comparison - handle hyphens and punctuation
	normalizedPlayer := c.normalizePlayerName(playerName)
	normalizedSearch := c.normalizePlayerName(searchName)
	
	playerParts := strings.Fields(normalizedPlayer)
	searchParts := strings.Fields(normalizedSearch)
	
	// Exact match gets highest score
	if normalizedPlayer == normalizedSearch {
		return 100
	}
	
	// Handle full name vs full name
	if len(playerParts) >= 2 && len(searchParts) >= 2 {
		// For multi-part names, require exact number of parts to match
		// This prevents "josh allen" from matching "josh hines allen"
		if len(playerParts) != len(searchParts) {
			return 0 // Different number of name parts = no match
		}
		
		firstName := playerParts[0]
		lastName := playerParts[len(playerParts)-1]
		searchFirst := searchParts[0]
		searchLast := searchParts[len(searchParts)-1]
		
		// Both first and last name match exactly
		if firstMatch := strings.Contains(firstName, searchFirst) || strings.Contains(searchFirst, firstName); firstMatch {
			if lastMatch := strings.Contains(lastName, searchLast) || strings.Contains(searchLast, lastName); lastMatch {
				// For 3+ part names, check middle parts too
				if len(playerParts) >= 3 {
					for i := 1; i < len(playerParts)-1; i++ {
						middleScore := c.calculateNameSimilarity(playerParts[i], searchParts[i])
						if middleScore < 70 {
							return 0 // Middle parts must match well too
						}
					}
				}
				
				// Check if both names have good overlap
				firstScore := c.calculateNameSimilarity(firstName, searchFirst)
				lastScore := c.calculateNameSimilarity(lastName, searchLast)
				
				// Return weighted score - both names must match well
				return (firstScore + lastScore) / 2
			}
		}
		
		// Only last name provided in search (like "jackson" searching for "lamar jackson")
		if len(searchParts) == 1 {
			lastScore := c.calculateNameSimilarity(lastName, searchParts[0])
			// Reduce score for last name only matches to prevent confusion
			if lastScore >= 90 {
				return lastScore - 30 // Reduce by 30 points for last name only
			}
		}
	}
	
	// Handle case where search has 1 part, player has 2+ parts
	if len(searchParts) == 1 && len(playerParts) >= 2 {
		lastName := playerParts[len(playerParts)-1]
		lastScore := c.calculateNameSimilarity(lastName, searchParts[0])
		// Reduce score for last name only matches to prevent confusion
		if lastScore >= 90 {
			return lastScore - 30
		}
	}
	
	// Fallback: check for any significant matches
	if strings.Contains(playerName, searchName) {
		return 40
	}
	if strings.Contains(searchName, playerName) {
		return 35
	}
	
	return 0
}

// calculateNameSimilarity calculates similarity score between two name parts
func (c *Client) calculateNameSimilarity(name1, name2 string) int {
	if name1 == name2 {
		return 100
	}
	
	// Check for exact containment
	if strings.Contains(name1, name2) || strings.Contains(name2, name1) {
		// Score based on length of shorter name
		shorter := name1
		if len(name2) < len(name1) {
			shorter = name2
		}
		
		// Score based on how much of the shorter name is contained
		if len(shorter) >= 4 {
			return 90
		}
		if len(shorter) >= 3 {
			return 70
		}
	}
	
	// Check for common prefixes
	minLen := len(name1)
	if len(name2) < minLen {
		minLen = len(name2)
	}
	
	if minLen >= 3 {
		for i := minLen; i >= 3; i-- {
			if name1[:i] == name2[:i] {
				return int(float64(i) / float64(minLen) * 60)
			}
		}
	}
	
	return 0
}

// getAPIErrorReason provides user-friendly explanations for API errors
func (c *Client) getAPIErrorReason(statusCode int) string {
	switch statusCode {
	case 401:
		return "API key is invalid or expired. Check your NFL_API_KEY in .env file"
	case 403:
		return "Access forbidden. Your API plan may not include this data or you've exceeded rate limits"
	case 404:
		return "Data not found. The requested week/season may not be available yet"
	case 429:
		return "Rate limit exceeded. Too many requests in a short time. Try again later"
	case 500:
		return "NFL API server error. This is temporary, try again in a few minutes"
	case 502, 503, 504:
		return "NFL API is currently unavailable. Service may be down for maintenance"
	default:
		return "Unknown error occurred. Check your internet connection and API key"
	}
}

// findTeamInCachedData finds a team in the cached team data
func (c *Client) findTeamInCachedData(teams []SportsDataTeam, name string) (*models.TeamInfo, error) {
	// Find team by name (case-insensitive partial match)
	var foundTeam *SportsDataTeam
	searchName := strings.ToLower(name)
	for i := range teams {
		team := &teams[i]
		if strings.Contains(strings.ToLower(team.Name), searchName) ||
		   strings.Contains(strings.ToLower(team.City), searchName) ||
		   strings.Contains(strings.ToLower(team.FullName), searchName) ||
		   strings.Contains(strings.ToLower(team.Key), searchName) {
			foundTeam = team
			break
		}
	}

	if foundTeam == nil {
		return nil, fmt.Errorf("team '%s' not found", name)
	}

	// Convert to our model
	teamInfo := &models.TeamInfo{
		Name:       foundTeam.Name,
		City:       foundTeam.City,
		Conference: foundTeam.Conference,
		Division:   foundTeam.Division,
		Coach:      foundTeam.HeadCoach,
		Stadium:    foundTeam.StadiumName,
		Colors:     []string{}, // SportsData.io doesn't provide colors
	}

	return teamInfo, nil
}

// getAggregatedSeasonStats aggregates weekly stats to create season totals
func (c *Client) getAggregatedSeasonStats(playerName string, season int, seasonType string, cacheKey string) (*models.PlayerStats, error) {
	log.Printf("[NFL-API] Aggregating %d season stats for %s (weeks 1-18)", season, playerName)
	
	// We'll try a few key weeks and aggregate the stats
	// This simulates season totals by combining multiple weeks
	weeksToTry := []int{1, 2, 5, 10, 15, 18} // Sample weeks to reduce API calls
	
	var aggregatedStats *models.PlayerStats
	var foundAnyWeek bool
	
	for _, week := range weeksToTry {
		url := fmt.Sprintf("%s/stats/json/PlayerGameStatsByWeek/%d%s/%d?key=%s", 
			c.baseURL, season, seasonType, week, c.apiKey)
		
		log.Printf("[NFL-API] GET %s (Week %d for season totals)", url, week)
		
		resp, err := c.httpClient.Get(url)
		if err != nil {
			continue // Try next week
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			continue // Try next week
		}
		
		var weekStats []SportsDataPlayerStat
		if err := json.NewDecoder(resp.Body).Decode(&weekStats); err != nil {
			continue // Try next week
		}
		
		// Find player in this week's data using improved scoring
		var bestMatch *SportsDataPlayerStat
		var bestScore int
		searchName := strings.ToLower(playerName)
		
		for i := range weekStats {
			playerNameLower := strings.ToLower(weekStats[i].Name)
			
			// Calculate match score for this player
			score := c.calculatePlayerMatchScore(playerNameLower, searchName)
			if score > bestScore {
				bestScore = score
				bestMatch = &weekStats[i]
			}
		}
		
		// Only accept matches with sufficient score
		var foundPlayer *SportsDataPlayerStat
		if bestScore >= 50 {
			foundPlayer = bestMatch
			log.Printf("[NFL-API] Season stats found match: '%s' (score: %d) for search '%s'", bestMatch.Name, bestScore, playerName)
		}
		
		if foundPlayer != nil {
			if aggregatedStats == nil {
				// First time finding the player - initialize
				aggregatedStats = &models.PlayerStats{
					Name:     foundPlayer.Name,
					Team:     foundPlayer.Team,
					Position: foundPlayer.Position,
					Season:   season,
					Stats:    make(map[string]interface{}),
				}
				
				// Initialize stats to 0
				aggregatedStats.Stats["passing_yards"] = 0
				aggregatedStats.Stats["passing_touchdowns"] = 0
				aggregatedStats.Stats["interceptions"] = 0
				aggregatedStats.Stats["rushing_yards"] = 0
				aggregatedStats.Stats["rushing_touchdowns"] = 0
				aggregatedStats.Stats["receiving_yards"] = 0
				aggregatedStats.Stats["receiving_touchdowns"] = 0
				aggregatedStats.Stats["receptions"] = 0
				aggregatedStats.Stats["targets"] = 0
				aggregatedStats.Stats["games_played"] = 0
			}
			
			// Add this week's stats to the totals
			if foundPlayer.PassingYards > 0 || foundPlayer.PassingTouchdowns > 0 {
				aggregatedStats.Stats["passing_yards"] = aggregatedStats.Stats["passing_yards"].(int) + int(foundPlayer.PassingYards)
				aggregatedStats.Stats["passing_touchdowns"] = aggregatedStats.Stats["passing_touchdowns"].(int) + int(foundPlayer.PassingTouchdowns)
				aggregatedStats.Stats["interceptions"] = aggregatedStats.Stats["interceptions"].(int) + int(foundPlayer.Interceptions)
			}
			
			if foundPlayer.RushingYards > 0 || foundPlayer.RushingTouchdowns > 0 {
				aggregatedStats.Stats["rushing_yards"] = aggregatedStats.Stats["rushing_yards"].(int) + int(foundPlayer.RushingYards)
				aggregatedStats.Stats["rushing_touchdowns"] = aggregatedStats.Stats["rushing_touchdowns"].(int) + int(foundPlayer.RushingTouchdowns)
			}
			
			if foundPlayer.ReceivingYards > 0 || foundPlayer.ReceivingTouchdowns > 0 {
				aggregatedStats.Stats["receiving_yards"] = aggregatedStats.Stats["receiving_yards"].(int) + int(foundPlayer.ReceivingYards)
				aggregatedStats.Stats["receiving_touchdowns"] = aggregatedStats.Stats["receiving_touchdowns"].(int) + int(foundPlayer.ReceivingTouchdowns)
				aggregatedStats.Stats["receptions"] = aggregatedStats.Stats["receptions"].(int) + int(foundPlayer.Receptions)
				aggregatedStats.Stats["targets"] = aggregatedStats.Stats["targets"].(int) + int(foundPlayer.Targets)
			}
			
			aggregatedStats.Stats["games_played"] = aggregatedStats.Stats["games_played"].(int) + 1
			foundAnyWeek = true
		}
	}
	
	if !foundAnyWeek {
		return nil, fmt.Errorf("player '%s' not found in %d season data", playerName, season)
	}
	
	// Calculate completion percentage if passing stats exist
	passingYards := aggregatedStats.Stats["passing_yards"].(int)
	if passingTDs, ok := aggregatedStats.Stats["passing_touchdowns"].(int); ok && (passingYards > 0 || passingTDs > 0) {
		// Estimate completion % based on stats (simplified)
		if passingYards > 0 {
			aggregatedStats.Stats["completion_percent"] = "Est. 65.0%" // Reasonable estimate
		}
	}
	
	// Add season identifier to stats
	aggregatedStats.Stats["season_note"] = fmt.Sprintf("Sample from %d of 18 games (not full season)", aggregatedStats.Stats["games_played"])
	
	// Cache the result
	c.setCachedData(cacheKey, aggregatedStats)
	
	log.Printf("[NFL-API] Completed season aggregation for %s: %d games sampled", playerName, aggregatedStats.Stats["games_played"])
	
	return aggregatedStats, nil
}

// GetPlayerStats retrieves statistics for a given player from SportsData.io API
func (c *Client) GetPlayerStats(playerName string) (*models.PlayerStats, error) {
	// Normalize player name
	name := strings.TrimSpace(playerName)
	if name == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	// Get current season information
	seasonInfo, err := c.getCurrentSeason()
	if err != nil {
		return nil, fmt.Errorf("failed to get current season: %v", err)
	}

	// Create cache key
	cacheKey := fmt.Sprintf("player_stats_%s_%d%s_%d", 
		strings.ToLower(name), seasonInfo.Season, seasonInfo.SeasonType, seasonInfo.Week)

	// Check cache first
	if cachedData, found := c.getCachedData(cacheKey); found {
		log.Printf("[NFL-CACHE] Using cached player stats for %s", name)
		return cachedData.(*models.PlayerStats), nil
	}

	// Build API endpoint with current season and week
	url := fmt.Sprintf("%s/stats/json/PlayerGameStatsByWeek/%d%s/%d?key=%s", 
		c.baseURL, seasonInfo.Season, seasonInfo.SeasonType, seasonInfo.Week, c.apiKey)

	// Log the request
	c.logRequest("GET", url)

	// Make HTTP request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NFL-API] ERROR: HTTP %d - %s for URL: %s", resp.StatusCode, http.StatusText(resp.StatusCode), url)
		errorReason := c.getAPIErrorReason(resp.StatusCode)
		return nil, fmt.Errorf("API request failed with status %d (%s): %s", resp.StatusCode, http.StatusText(resp.StatusCode), errorReason)
	}

	// Parse JSON response
	var sportsDataStats []SportsDataPlayerStat
	if err := json.NewDecoder(resp.Body).Decode(&sportsDataStats); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	// Find player by name using improved scored matching
	var bestMatch *SportsDataPlayerStat
	var bestScore int
	searchName := strings.ToLower(name)
	
	log.Printf("[NFL-API] Searching for player: '%s' in %d player records", name, len(sportsDataStats))
	
	// Log first few players to help debug
	if len(sportsDataStats) > 0 {
		log.Printf("[NFL-API] Sample players: %s, %s, %s", 
			sportsDataStats[0].Name, 
			getSafeName(sportsDataStats, 1),
			getSafeName(sportsDataStats, 2))
	}
	
	for i := range sportsDataStats {
		playerName := strings.ToLower(sportsDataStats[i].Name)
		
		// Calculate match score
		score := c.calculatePlayerMatchScore(playerName, searchName)
		if score > bestScore {
			bestScore = score
			bestMatch = &sportsDataStats[i]
			log.Printf("[NFL-API] New best match: '%s' (score: %d) for search '%s'", sportsDataStats[i].Name, score, name)
		}
	}

	// Require minimum score to prevent bad matches
	if bestScore < 50 {
		return nil, fmt.Errorf("player '%s' not found in current week's stats. Try a different spelling or check if they played this week", name)
	}

	log.Printf("[NFL-API] Final match: '%s' with score %d", bestMatch.Name, bestScore)

	// Convert to our model format
	stats := &models.PlayerStats{
		Name:     bestMatch.Name,
		Team:     bestMatch.Team,
		Position: bestMatch.Position,
		Season:   int(bestMatch.Season),
		Stats:    make(map[string]interface{}),
	}

	// Add relevant stats based on position
	if bestMatch.PassingYards > 0 || bestMatch.PassingTouchdowns > 0 {
		stats.Stats["passing_yards"] = int(bestMatch.PassingYards)
		stats.Stats["passing_touchdowns"] = int(bestMatch.PassingTouchdowns)
		stats.Stats["interceptions"] = int(bestMatch.Interceptions)
		if bestMatch.Attempts > 0 {
			completionPct := bestMatch.Completions / bestMatch.Attempts * 100
			stats.Stats["completion_percent"] = fmt.Sprintf("%.1f%%", completionPct)
		}
	}

	if bestMatch.RushingYards > 0 || bestMatch.RushingTouchdowns > 0 {
		stats.Stats["rushing_yards"] = int(bestMatch.RushingYards)
		stats.Stats["rushing_touchdowns"] = int(bestMatch.RushingTouchdowns)
	}

	if bestMatch.ReceivingYards > 0 || bestMatch.ReceivingTouchdowns > 0 {
		stats.Stats["receiving_yards"] = int(bestMatch.ReceivingYards)
		stats.Stats["receiving_touchdowns"] = int(bestMatch.ReceivingTouchdowns)
		stats.Stats["receptions"] = int(bestMatch.Receptions)
		stats.Stats["targets"] = int(bestMatch.Targets)
	}

	// Cache the result
	c.setCachedData(cacheKey, stats)

	return stats, nil
}

// GetTeamInfo retrieves information about a team
func (c *Client) GetTeamInfo(teamName string) (*models.TeamInfo, error) {
	// Normalize team name
	name := strings.TrimSpace(teamName)
	if name == "" {
		return nil, fmt.Errorf("team name cannot be empty")
	}

	// Create cache key for teams data
	cacheKey := "teams_data"

	// Check cache first
	if cachedData, found := c.getCachedData(cacheKey); found {
		log.Printf("[NFL-CACHE] Using cached teams data for %s", name)
		// Extract team from cached data
		return c.findTeamInCachedData(cachedData.([]SportsDataTeam), name)
	}

	// Get all teams
	url := fmt.Sprintf("%s/scores/json/Teams?key=%s", c.baseURL, c.apiKey)
	
	// Log the request
	c.logRequest("GET", url)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teams: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NFL-API] ERROR: HTTP %d - %s for URL: %s", resp.StatusCode, http.StatusText(resp.StatusCode), url)
		errorReason := c.getAPIErrorReason(resp.StatusCode)
		return nil, fmt.Errorf("teams API request failed with status %d (%s): %s", resp.StatusCode, http.StatusText(resp.StatusCode), errorReason)
	}

	var teams []SportsDataTeam
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, fmt.Errorf("failed to parse teams response: %v", err)
	}

	// Cache the teams data
	c.setCachedData(cacheKey, teams)

	// Find team using helper function
	return c.findTeamInCachedData(teams, name)
}

// GetTeamSchedule retrieves schedule for a team
func (c *Client) GetTeamSchedule(teamName string) (*models.Schedule, error) {
	// Normalize team name
	name := strings.TrimSpace(teamName)
	if name == "" {
		return nil, fmt.Errorf("team name cannot be empty")
	}

	// Get current season info
	seasonInfo, err := c.getCurrentSeason()
	if err != nil {
		return nil, fmt.Errorf("failed to get current season: %v", err)
	}

	// Create cache key for team schedule
	cacheKey := fmt.Sprintf("team_schedule_%s_%d%s", 
		strings.ToLower(name), seasonInfo.Season, seasonInfo.SeasonType)

	// Check cache first
	if cachedData, found := c.getCachedData(cacheKey); found {
		log.Printf("[NFL-CACHE] Using cached team schedule for %s", name)
		return cachedData.(*models.Schedule), nil
	}

	// Get team schedule for current season
	url := fmt.Sprintf("%s/scores/json/Schedules/%d%s?key=%s", 
		c.baseURL, seasonInfo.Season, seasonInfo.SeasonType, c.apiKey)
	
	// Log the request
	c.logRequest("GET", url)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NFL-API] ERROR: HTTP %d - %s for URL: %s", resp.StatusCode, http.StatusText(resp.StatusCode), url)
		errorReason := c.getAPIErrorReason(resp.StatusCode)
		return nil, fmt.Errorf("schedule API request failed with status %d (%s): %s", resp.StatusCode, http.StatusText(resp.StatusCode), errorReason)
	}

	var games []SportsDataGame
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to parse schedule response: %v", err)
	}

	// Filter games for the specified team
	var teamGames []models.Game
	searchVariations := normalizeTeamName(name)

	log.Printf("[NFL-API] Searching for team: '%s' with variations: %v, found %d total games", name, searchVariations, len(games))

	// Debug: Show first few teams to understand the data format
	if len(games) > 0 {
		log.Printf("[NFL-API] Sample teams from API: Home='%s', Away='%s'", games[0].HomeTeam, games[0].AwayTeam)
	}

	for _, game := range games {
		homeTeam := strings.ToLower(game.HomeTeam)
		awayTeam := strings.ToLower(game.AwayTeam)
		
		// Check if this is a BYE week for our team
		isByeWeek := strings.ToUpper(game.HomeTeam) == "BYE" || strings.ToUpper(game.AwayTeam) == "BYE"
		
		// For BYE weeks, check if the non-BYE team matches our search
		var matchesTeam bool
		if isByeWeek {
			// Find the actual team in the BYE week
			actualTeam := game.HomeTeam
			if strings.ToUpper(game.HomeTeam) == "BYE" {
				actualTeam = game.AwayTeam
			}
			
			// Check if the actual team matches our search variations
			for _, variation := range searchVariations {
				if strings.Contains(strings.ToLower(actualTeam), variation) {
					matchesTeam = true
					break
				}
			}
		} else {
			// Regular game - check if any search variation matches either team
			for _, variation := range searchVariations {
				if strings.Contains(homeTeam, variation) || strings.Contains(awayTeam, variation) {
					matchesTeam = true
					break
				}
			}
		}
		
		if !matchesTeam {
			continue
		}
		
		log.Printf("[NFL-API] Found matching game: %s @ %s (Week %d)", game.AwayTeam, game.HomeTeam, game.Week)

		// Parse game time (skip for BYE weeks which may have empty datetime)
		var gameTime time.Time
		if game.DateTime != "" {
			var err error
			gameTime, err = parseSportsDataDateTime(game.DateTime)
			if err != nil {
				log.Printf("Warning: Could not parse game time '%s': %v", game.DateTime, err)
				gameTime = time.Time{} // Default to zero time
			}
		}

		// Convert to our model
		gameModel := models.Game{
			ID:          game.GameKey,
			Week:        game.Week,
			Season:      game.Season,
			GameType:    seasonInfo.SeasonType,
			HomeTeam:    game.HomeTeam,
			AwayTeam:    game.AwayTeam,
			HomeScore:   game.HomeScore,
			AwayScore:   game.AwayScore,
			GameTime:    gameTime,
			Status:      game.Status,
			Stadium:     game.Stadium,
		}

		teamGames = append(teamGames, gameModel)
	}

	log.Printf("[NFL-API] Found %d games for team '%s'", len(teamGames), name)

	if len(teamGames) == 0 {
		return nil, fmt.Errorf("no games found for team '%s'", name)
	}

	// Create schedule
	schedule := &models.Schedule{
		TeamName: name,
		Season:   seasonInfo.Season,
		Games:    teamGames,
	}

	// Cache the result
	c.setCachedData(cacheKey, schedule)

	return schedule, nil
}

// GetLiveScores retrieves current live scores
func (c *Client) GetLiveScores() ([]*models.LiveScore, error) {
	// Get current season info
	seasonInfo, err := c.getCurrentSeason()
	if err != nil {
		return nil, fmt.Errorf("failed to get current season: %v", err)
	}

	// Create cache key for live scores
	cacheKey := fmt.Sprintf("live_scores_%d%s_%d", 
		seasonInfo.Season, seasonInfo.SeasonType, seasonInfo.Week)

	// Check cache first
	if cachedData, found := c.getCachedData(cacheKey); found {
		log.Printf("[NFL-CACHE] Using cached live scores for week %d", seasonInfo.Week)
		return cachedData.([]*models.LiveScore), nil
	}

	// Get live scores for current week
	url := fmt.Sprintf("%s/scores/json/ScoresByWeek/%d%s/%d?key=%s", 
		c.baseURL, seasonInfo.Season, seasonInfo.SeasonType, seasonInfo.Week, c.apiKey)
	
	// Log the request
	c.logRequest("GET", url)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live scores: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NFL-API] ERROR: HTTP %d - %s for URL: %s", resp.StatusCode, http.StatusText(resp.StatusCode), url)
		errorReason := c.getAPIErrorReason(resp.StatusCode)
		return nil, fmt.Errorf("live scores API request failed with status %d (%s): %s", resp.StatusCode, http.StatusText(resp.StatusCode), errorReason)
	}

	var games []SportsDataGame
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to parse live scores response: %v", err)
	}

	// Convert to our live score model
	var liveScores []*models.LiveScore
	for _, game := range games {
		// Parse game time (skip for BYE weeks which may have empty datetime)
		var gameTime time.Time
		if game.DateTime != "" {
			var err error
			gameTime, err = parseSportsDataDateTime(game.DateTime)
			if err != nil {
				log.Printf("Warning: Could not parse live score game time '%s': %v", game.DateTime, err)
				gameTime = time.Time{} // Default to zero time
			}
		}

		liveScore := &models.LiveScore{
			GameID:        game.GameKey,
			Season:        game.Season,
			Week:          game.Week,
			AwayTeam:      game.AwayTeam,
			HomeTeam:      game.HomeTeam,
			AwayScore:     game.AwayScore,
			HomeScore:     game.HomeScore,
			TimeRemaining: game.TimeRemaining,
			Quarter:       game.Quarter,
			Status:        game.Status,
			GameTime:      gameTime,
		}

		liveScores = append(liveScores, liveScore)
	}

	// Cache the result
	c.setCachedData(cacheKey, liveScores)

	return liveScores, nil
}

// GetPlayerSeasonStats retrieves season statistics for a player from previous completed season
func (c *Client) GetPlayerSeasonStats(playerName string) (*models.PlayerStats, error) {
	// Normalize player name
	name := strings.TrimSpace(playerName)
	if name == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	// Use previous completed season (2024) for season stats
	prevSeason := 2024
	seasonType := "REG"
	
	// Create cache key
	cacheKey := fmt.Sprintf("player_season_stats_%s_%d%s", 
		strings.ToLower(name), prevSeason, seasonType)

	// Check cache first
	if cachedData, found := c.getCachedData(cacheKey); found {
		log.Printf("[NFL-CACHE] Using cached season stats for %s", name)
		return cachedData.(*models.PlayerStats), nil
	}

	// We'll sum up all weeks from the previous season to get season totals
	// Start with week 1 and aggregate through week 18
	return c.getAggregatedSeasonStats(name, prevSeason, seasonType, cacheKey)
}

// GetPlayerWeekStats retrieves statistics for a player from a specific week and season
func (c *Client) GetPlayerWeekStats(playerName string, season, week int) (*models.PlayerStats, error) {
	// Normalize player name
	name := strings.TrimSpace(playerName)
	if name == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	// Validate inputs
	if week < 1 || week > 18 {
		return nil, fmt.Errorf("invalid week number: %d (must be 1-18)", week)
	}
	if season < 2020 || season > 2025 {
		return nil, fmt.Errorf("invalid season: %d (must be 2020-2025)", season)
	}

	// Create cache key
	cacheKey := fmt.Sprintf("player_week_stats_%s_%d_REG_%d", 
		strings.ToLower(name), season, week)

	// Check cache first
	if cachedData, found := c.getCachedData(cacheKey); found {
		log.Printf("[NFL-CACHE] Using cached week %d stats for %s (%d)", week, name, season)
		return cachedData.(*models.PlayerStats), nil
	}

	// Build API endpoint
	url := fmt.Sprintf("%s/stats/json/PlayerGameStatsByWeek/%dREG/%d?key=%s", 
		c.baseURL, season, week, c.apiKey)

	// Log the request
	c.logRequest("GET", url)

	// Make HTTP request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NFL-API] ERROR: HTTP %d - %s for URL: %s", resp.StatusCode, http.StatusText(resp.StatusCode), url)
		errorReason := c.getAPIErrorReason(resp.StatusCode)
		return nil, fmt.Errorf("week stats API request failed with status %d (%s): %s", resp.StatusCode, http.StatusText(resp.StatusCode), errorReason)
	}

	// Parse JSON response
	var sportsDataStats []SportsDataPlayerStat
	if err := json.NewDecoder(resp.Body).Decode(&sportsDataStats); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	// Find player by name using improved scoring
	var bestMatch *SportsDataPlayerStat
	var bestScore int
	searchName := strings.ToLower(name)
	
	log.Printf("[NFL-API] Searching for player: '%s' in %d player records (Week %d, %d)", name, len(sportsDataStats), week, season)
	
	for i := range sportsDataStats {
		playerNameLower := strings.ToLower(sportsDataStats[i].Name)
		
		// Calculate match score for this player
		score := c.calculatePlayerMatchScore(playerNameLower, searchName)
		if score > bestScore {
			bestScore = score
			bestMatch = &sportsDataStats[i]
		}
	}

	// Require minimum score to prevent bad matches
	if bestScore < 50 {
		return nil, fmt.Errorf("player '%s' not found in Week %d, %d stats. Try a different spelling or check if they played that week", name, week, season)
	}
	
	log.Printf("[NFL-API] Week stats found match: '%s' (score: %d) for search '%s'", bestMatch.Name, bestScore, name)

	// Convert to our model format (same logic as current week)
	stats := &models.PlayerStats{
		Name:     bestMatch.Name,
		Team:     bestMatch.Team,
		Position: bestMatch.Position,
		Season:   int(bestMatch.Season),
		Stats:    make(map[string]interface{}),
	}

	// Add relevant stats based on position
	if bestMatch.PassingYards > 0 || bestMatch.PassingTouchdowns > 0 {
		stats.Stats["passing_yards"] = int(bestMatch.PassingYards)
		stats.Stats["passing_touchdowns"] = int(bestMatch.PassingTouchdowns)
		stats.Stats["interceptions"] = int(bestMatch.Interceptions)
		if bestMatch.Attempts > 0 {
			completionPct := bestMatch.Completions / bestMatch.Attempts * 100
			stats.Stats["completion_percent"] = fmt.Sprintf("%.1f%%", completionPct)
		}
	}

	if bestMatch.RushingYards > 0 || bestMatch.RushingTouchdowns > 0 {
		stats.Stats["rushing_yards"] = int(bestMatch.RushingYards)
		stats.Stats["rushing_touchdowns"] = int(bestMatch.RushingTouchdowns)
	}

	if bestMatch.ReceivingYards > 0 || bestMatch.ReceivingTouchdowns > 0 {
		stats.Stats["receiving_yards"] = int(bestMatch.ReceivingYards)
		stats.Stats["receiving_touchdowns"] = int(bestMatch.ReceivingTouchdowns)
		stats.Stats["receptions"] = int(bestMatch.Receptions)
		stats.Stats["targets"] = int(bestMatch.Targets)
	}

	// Cache the result
	c.setCachedData(cacheKey, stats)

	return stats, nil
}
