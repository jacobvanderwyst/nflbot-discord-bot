package bot

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"nfl-discord-bot/internal/config"
	"nfl-discord-bot/internal/nfl"
	"nfl-discord-bot/pkg/models"
)

// Bot represents the Discord bot
type Bot struct {
	discord       *discordgo.Session
	nflClient     *nfl.Client
	config        *config.Config
	silenceEnd    time.Time
	allowedRole   string
	visibilityRole string
	commands      []*discordgo.ApplicationCommand
}

// New creates a new Discord bot instance
func New(cfg *config.Config) (*Bot, error) {
	// Create Discord session
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %v", err)
	}

	// Create NFL client
	nflClient := nfl.NewClient(cfg.NFLAPIKey, cfg.NFLAPIBaseURL)

	bot := &Bot{
		discord:       dg,
		config:        cfg,
		nflClient:     nflClient,
		silenceEnd:    time.Time{},
		allowedRole:   os.Getenv("BOT_ALLOWED_ROLE"),
		visibilityRole: os.Getenv("BOT_VISIBILITY_ROLE"),
	}

	// Initialize slash commands after bot creation
	bot.commands = bot.createSlashCommands()

	// Register message handler and interaction handler
	dg.AddHandler(bot.messageCreate)
	dg.AddHandler(bot.interactionCreate)

	return bot, nil
}

// Start starts the Discord bot
func (b *Bot) Start() error {
	err := b.discord.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %v", err)
	}

	// Register slash commands
	log.Println("Registering slash commands...")
	for _, cmd := range b.commands {
		_, err := b.discord.ApplicationCommandCreate(b.discord.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", cmd.Name, err)
		}
	}

	log.Println("Discord bot is now running with slash commands")
	return nil
}

// Stop stops the Discord bot
func (b *Bot) Stop() {
	b.discord.Close()
}

// createSlashCommands defines the slash commands for the bot
func (b *Bot) createSlashCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Show comprehensive command documentation",
		},
		{
			Name:        "stats",
			Description: "Get player statistics",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "player",
					Description: "Player name",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "Stats type",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Current Week", Value: "current"},
						{Name: "Season", Value: "season"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "week",
					Description: "Specific week number (1-18)",
					Required:    false,
					MinValue:    &[]float64{1}[0],
					MaxValue:    18,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "year",
					Description: "Year (defaults to current season)",
					Required:    false,
				},
			},
		},
		{
			Name:        "compare",
			Description: "Compare two players",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "player1",
					Description: "First player name",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "player2",
					Description: "Second player name",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "Comparison type",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Current Week", Value: "current"},
						{Name: "Season", Value: "season"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "week",
					Description: "Specific week number (1-18)",
					Required:    false,
					MinValue:    &[]float64{1}[0],
					MaxValue:    18,
				},
			},
		},
		{
			Name:        "team",
			Description: "Get team information",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "team",
					Description: "Team name, city, or abbreviation",
					Required:    true,
				},
			},
		},
		{
			Name:        "schedule",
			Description: "Get team schedule",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "team",
					Description: "Team name, city, or abbreviation",
					Required:    true,
				},
			},
		},
		{
			Name:        "scores",
			Description: "Get current week's scores",
		},
	}
}

// interactionCreate handles slash command interactions
func (b *Bot) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if bot is silenced
	if time.Now().Before(b.silenceEnd) {
		return // Bot is silenced, ignore all interactions
	}

	// Check role permissions if configured
	if b.allowedRole != "" && !b.hasAllowedRoleForInteraction(s, i) {
		// Send ephemeral error message
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ You don't have permission to use this bot.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}
		return
	}

	// Handle slash commands
	switch i.ApplicationCommandData().Name {
	case "help":
		b.handleSlashHelp(s, i)
	case "stats":
		b.handleSlashStats(s, i)
	case "compare":
		b.handleSlashCompare(s, i)
	case "team":
		b.handleSlashTeam(s, i)
	case "schedule":
		b.handleSlashSchedule(s, i)
	case "scores":
		b.handleSlashScores(s, i)
	}
}

// messageCreate handles incoming Discord messages
func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check for silence command
	if strings.TrimSpace(m.Content) == "/s" {
		b.handleSilenceCommand(s, m)
		return
	}

	// Check if bot is silenced
	if time.Now().Before(b.silenceEnd) {
		return // Bot is silenced, ignore all commands
	}

	// Check if message starts with bot prefix
	if !strings.HasPrefix(m.Content, b.config.BotPrefix) {
		return
	}

	// Check role permissions if configured
	if b.allowedRole != "" && !b.hasAllowedRole(s, m) {
		return // User doesn't have required role
	}

	// Remove prefix and split command and arguments
	content := strings.TrimPrefix(m.Content, b.config.BotPrefix)
	args := strings.Fields(content)
	if len(args) == 0 {
		return
	}

	command := strings.ToLower(args[0])

	// Handle commands
	switch command {
	case "help":
		b.handleHelp(s, m)
	case "stats":
		b.handleStats(s, m, args[1:])
	case "compare":
		b.handleCompare(s, m, args[1:])
	case "team":
		b.handleTeam(s, m, args[1:])
	case "schedule":
		b.handleSchedule(s, m, args[1:])
	case "scores":
		b.handleScores(s, m)
	default:
		b.sendMessage(s, m.ChannelID, "Unknown command. Use `!help` to see available commands.")
	}
}

// handleHelp shows comprehensive command documentation
func (b *Bot) handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "🏈 NFL Discord Bot - Complete Command Guide",
		Description: "**Intelligent NFL data with real-time stats, schedules, and scores**\n\n" +
			"*Smart week detection: Wednesday shows previous week, Thursday-Monday shows current week*",
		Color: 0x013369,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "📊 Player Statistics",
				Value: "`!stats <player_name>` - Current week stats (2025)\n" +
					   "`!stats --season <player_name>` - 2024 sample stats (6 games)\n" +
					   "`!stats --week <#> <player_name>` - Specific week (current season)\n" +
					   "`!stats --week <#> <year> <player_name>` - Specific week & year\n" +
					   "*Examples: `!stats Josh Allen`, `!stats --week 5 Saquon Barkley`*",
				Inline: false,
			},
			{
				Name:  "⚖️ Player Comparisons",
				Value: "`!compare <player1> vs <player2>` - Compare current week stats\n" +
					   "`!compare --season <player1> vs <player2>` - Compare season stats\n" +
					   "`!compare --week <#> <player1> vs <player2>` - Compare specific week\n" +
					   "*Examples: `!compare Josh Allen vs Mahomes`, `!compare --week 5 Henry vs Barkley`*",
				Inline: false,
			},
			{
				Name:  "🏟️ Team Information",
				Value: "`!team <team_name>` - Complete team details\n" +
					   "*Shows: Conference, division, coach, stadium*\n" +
					   "*Examples: `!team Bills`, `!team Eagles`, `!team KC`*",
				Inline: false,
			},
			{
				Name:  "📅 Team Schedule",
				Value: "`!schedule <team_name>` - Full season schedule\n" +
					   "*Shows: Game dates, opponents, scores, BYE weeks*\n" +
					   "*Examples: `!schedule Cowboys`, `!schedule Patriots`*",
				Inline: false,
			},
			{
				Name:  "🔴 Live Scores",
				Value: "`!scores` - Current week's games and scores\n" +
					   "*Shows: Live games, completed games, upcoming games*\n" +
					   "*Updates automatically based on current NFL week*",
				Inline: false,
			},
			{
				Name:  "⚡ Smart Features",
				Value: "• **Auto Week Detection** - Always shows current NFL week\n" +
					   "• **5-Minute Caching** - Fast responses, reduced API calls\n" +
					   "• **Flexible Team Names** - Use full names, cities, or abbreviations\n" +
					   "• **Real-Time Data** - Live stats from SportsData.io",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "🤖 Data updates every 5 minutes | 📡 Powered by SportsData.io | 🔧 Built for Discord",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	b.sendEmbed(s, m.ChannelID, embed)
}

// handleStats handles player statistics requests
func (b *Bot) handleStats(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		b.sendMessage(s, m.ChannelID, "Please provide a player name. Usage: `!stats <player_name>` or `!stats --season <player_name>` for season totals")
		return
	}

	// Send acknowledgment notification
	var acknowledgment string
	if len(args) > 0 && args[0] == "--season" {
		acknowledgment = "⏳ Fetching season stats... (this may take a moment)"
	} else if len(args) > 0 && args[0] == "--week" {
		acknowledgment = "⏳ Fetching week-specific stats..."
	} else {
		acknowledgment = "⏳ Fetching current week stats..."
	}
	ack, _ := s.ChannelMessageSend(m.ChannelID, acknowledgment)
	
	// Delete the original command message
	go func() {
		time.Sleep(1 * time.Second) // Brief delay to ensure acknowledgment is sent
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}()

	// Check for flags
	var playerName string
	var isSeasonStats bool
	var specificWeek int
	var specificSeason int
	var useSpecificWeek bool
	
	if args[0] == "--season" {
		if len(args) < 2 {
			b.sendMessage(s, m.ChannelID, "Please provide a player name after --season flag. Usage: `!stats --season <player_name>`")
			return
		}
		isSeasonStats = true
		playerName = strings.Join(args[1:], " ")
	} else if args[0] == "--week" {
		if len(args) < 3 {
			b.sendMessage(s, m.ChannelID, "Please provide week number and player name. Usage: `!stats --week <week> <player_name>` or `!stats --week <week> <year> <player_name>`")
			return
		}
		
		// Parse week number
		weekNum, err := strconv.Atoi(args[1])
		if err != nil || weekNum < 1 || weekNum > 18 {
			b.sendMessage(s, m.ChannelID, "Invalid week number. Please use a number between 1 and 18.")
			return
		}
		specificWeek = weekNum
		
		// Check if third argument is a year or part of player name
		if len(args) >= 4 {
			if yearNum, err := strconv.Atoi(args[2]); err == nil && yearNum >= 2020 && yearNum <= 2025 {
				// Third argument is a year
				specificSeason = yearNum
				playerName = strings.Join(args[3:], " ")
			} else {
				// Third argument is part of player name, use current season
				specificSeason = 2025 // Default to current season
				playerName = strings.Join(args[2:], " ")
			}
		} else {
			// Only week and player name provided, use current season
			specificSeason = 2025
			playerName = strings.Join(args[2:], " ")
		}
		useSpecificWeek = true
	} else {
		playerName = strings.Join(args, " ")
	}
	
	// Get player stats from NFL client
	var stats *models.PlayerStats
	var err error
	
	if isSeasonStats {
		stats, err = b.nflClient.GetPlayerSeasonStats(playerName)
	} else if useSpecificWeek {
		stats, err = b.nflClient.GetPlayerWeekStats(playerName, specificSeason, specificWeek)
	} else {
		stats, err = b.nflClient.GetPlayerStats(playerName)
	}
	
	if err != nil {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		statsType := "current week"
		if isSeasonStats {
			statsType = "season sample"
		} else if useSpecificWeek {
			statsType = fmt.Sprintf("Week %d, %d", specificWeek, specificSeason)
		}
		b.sendMessage(s, m.ChannelID, fmt.Sprintf("Error getting %s stats for %s: %v", statsType, playerName, err))
		return
	}

	// Create embed with player stats
	statsTitle := "Current Week Stats (2025)"
	if isSeasonStats {
		statsTitle = "2024 Sample Stats (6 games)"
	} else if useSpecificWeek {
		statsTitle = fmt.Sprintf("Week %d, %d Stats", specificWeek, specificSeason)
	}
	
	// Delete acknowledgment message before sending results
	if ack != nil {
		s.ChannelMessageDelete(m.ChannelID, ack.ID)
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s - %s", stats.Name, statsTitle),
		Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Team",
				Value:  stats.Team,
				Inline: true,
			},
			{
				Name:   "Position",
				Value:  stats.Position,
				Inline: true,
			},
			{
				Name:   "Season Stats",
				Value:  stats.GetStatsString(),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from NFL API",
		},
	}

	b.sendEmbed(s, m.ChannelID, embed)
}

// handleTeam handles team information requests
func (b *Bot) handleTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		b.sendMessage(s, m.ChannelID, "Please provide a team name. Usage: `!team <team_name>`")
		return
	}

// Send acknowledgment notification
	ack, _ := s.ChannelMessageSend(m.ChannelID, "⏳ Fetching team information...")
	
	// Delete the original command message
	go func() {
		time.Sleep(1 * time.Second)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}()

	teamName := strings.Join(args, " ")
	
	// Get team info from NFL client
	teamInfo, err := b.nflClient.GetTeamInfo(teamName)
	if err != nil {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		b.sendMessage(s, m.ChannelID, fmt.Sprintf("Error getting team info for %s: %v", teamName, err))
		return
	}

	// Delete acknowledgment message before sending results
	if ack != nil {
		s.ChannelMessageDelete(m.ChannelID, ack.ID)
	}

	// Create embed with team info
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🏈 %s %s", teamInfo.City, teamInfo.Name),
		Color: 0xff6600,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Conference",
				Value:  teamInfo.Conference,
				Inline: true,
			},
			{
				Name:   "Division",
				Value:  teamInfo.Division,
				Inline: true,
			},
			{
				Name:   "Head Coach",
				Value:  teamInfo.Coach,
				Inline: true,
			},
			{
				Name:   "Stadium",
				Value:  teamInfo.Stadium,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Team data from NFL API",
		},
	}

	b.sendEmbed(s, m.ChannelID, embed)
}

// handleSchedule handles team schedule requests
func (b *Bot) handleSchedule(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		b.sendMessage(s, m.ChannelID, "Please provide a team name. Usage: `!schedule <team_name>`")
		return
	}

// Send acknowledgment notification
	ack, _ := s.ChannelMessageSend(m.ChannelID, "⏳ Fetching team schedule...")
	
	// Delete the original command message
	go func() {
		time.Sleep(1 * time.Second)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}()

	teamName := strings.Join(args, " ")
	
	// Get team schedule from NFL client
	schedule, err := b.nflClient.GetTeamSchedule(teamName)
	if err != nil {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		b.sendMessage(s, m.ChannelID, fmt.Sprintf("Error getting schedule for %s: %v", teamName, err))
		return
	}

	// Create embed with schedule (show first 10 games to avoid too long message)
	var scheduleText string
	gamesToShow := schedule.Games
	if len(gamesToShow) > 10 {
		gamesToShow = gamesToShow[:10]
	}

	for _, game := range gamesToShow {
		// Check if this is a BYE week
		if game.HomeTeam == "BYE" || game.AwayTeam == "BYE" {
			scheduleText += fmt.Sprintf("**Week %d**: 🛌 **BYE WEEK** - Rest and Recovery\n", game.Week)
			continue
		}
		
		gameDate := game.GameTime.Format("Jan 2, 3:04 PM")
		if game.IsCompleted() {
			scheduleText += fmt.Sprintf("**Week %d**: %s @ %s - %s %d-%d (Final)\n", 
				game.Week, game.AwayTeam, game.HomeTeam, game.Winner(), game.AwayScore, game.HomeScore)
		} else if game.IsLive() {
			scheduleText += fmt.Sprintf("**Week %d**: %s @ %s - %d-%d (LIVE)\n", 
				game.Week, game.AwayTeam, game.HomeTeam, game.AwayScore, game.HomeScore)
		} else {
			scheduleText += fmt.Sprintf("**Week %d**: %s @ %s - %s\n", 
				game.Week, game.AwayTeam, game.HomeTeam, gameDate)
		}
	}

	// Delete acknowledgment message before sending results
	if ack != nil {
		s.ChannelMessageDelete(m.ChannelID, ack.ID)
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📅 %s Schedule (%d Season)", schedule.TeamName, schedule.Season),
		Color: 0x00ff00,
		Description: scheduleText,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing %d of %d games", len(gamesToShow), len(schedule.Games)),
		},
	}

	b.sendEmbed(s, m.ChannelID, embed)
}

// handleScores handles live scores requests
func (b *Bot) handleScores(s *discordgo.Session, m *discordgo.MessageCreate) {
// Send acknowledgment notification
	ack, _ := s.ChannelMessageSend(m.ChannelID, "⏳ Fetching live scores...")
	
	// Delete the original command message
	go func() {
		time.Sleep(1 * time.Second)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}()

	// Get live scores from NFL client
	liveScores, err := b.nflClient.GetLiveScores()
	if err != nil {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		b.sendMessage(s, m.ChannelID, fmt.Sprintf("Error getting live scores: %v", err))
		return
	}

	if len(liveScores) == 0 {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		b.sendMessage(s, m.ChannelID, "No games found for this week.")
		return
	}

	// Create embed with live scores
	var scoresText string
	liveCount := 0
	completedCount := 0

	for _, score := range liveScores {
		if score.IsLive() {
			scoresText += fmt.Sprintf("🔴 **%s** - %s\n", "LIVE", score.GetScoreString())
			liveCount++
		} else if score.IsCompleted() {
			scoresText += fmt.Sprintf("✅ **FINAL** - %s\n", score.GetScoreString())
			completedCount++
		} else {
			gameTime := score.GameTime.Format("Jan 2, 3:04 PM")
			scoresText += fmt.Sprintf("📅 **%s** - %s @ %s\n", gameTime, score.AwayTeam, score.HomeTeam)
		}
	}

	// Delete acknowledgment message before sending results
	if ack != nil {
		s.ChannelMessageDelete(m.ChannelID, ack.ID)
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🏈 NFL Scores - Week %d", liveScores[0].Week),
		Color: 0x013369,
		Description: scoresText,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d live, %d completed, %d total games", liveCount, completedCount, len(liveScores)),
		},
	}

	b.sendEmbed(s, m.ChannelID, embed)
}

// handleCompare handles player comparison requests
func (b *Bot) handleCompare(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 3 {
		b.sendMessage(s, m.ChannelID, "Please provide two players to compare. Usage: `!compare Player1 vs Player2` or `!compare --week 5 Player1 vs Player2`")
		return
	}

	// Send acknowledgment notification
	var acknowledgment string
	if len(args) > 0 && args[0] == "--season" {
		acknowledgment = "⏳ Comparing season stats... (this may take a moment)"
	} else if len(args) > 0 && args[0] == "--week" {
		acknowledgment = "⏳ Comparing week-specific stats..."
	} else {
		acknowledgment = "⏳ Comparing current week stats..."
	}
	ack, _ := s.ChannelMessageSend(m.ChannelID, acknowledgment)
	
	// Delete the original command message
	go func() {
		time.Sleep(1 * time.Second)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}()

	// Parse arguments for flags and players
	var isSeasonStats bool
	var specificWeek int
	var specificSeason int
	var useSpecificWeek bool
	var argOffset int

	// Check for flags
	if args[0] == "--season" {
		isSeasonStats = true
		argOffset = 1
	} else if args[0] == "--week" {
		if len(args) < 4 {
			b.sendMessage(s, m.ChannelID, "Please provide week number and two players. Usage: `!compare --week 5 Player1 vs Player2`")
			return
		}
		
		weekNum, err := strconv.Atoi(args[1])
		if err != nil || weekNum < 1 || weekNum > 18 {
			b.sendMessage(s, m.ChannelID, "Invalid week number. Please use a number between 1 and 18.")
			return
		}
		specificWeek = weekNum
		specificSeason = 2025 // Default to current season for comparisons
		useSpecificWeek = true
		argOffset = 2
	}

	// Find "vs" separator
	vsIndex := -1
	for i := argOffset; i < len(args); i++ {
		if strings.ToLower(args[i]) == "vs" || strings.ToLower(args[i]) == "versus" {
			vsIndex = i
			break
		}
	}

	if vsIndex == -1 {
		b.sendMessage(s, m.ChannelID, "Please separate players with 'vs'. Usage: `!compare Player1 vs Player2`")
		return
	}

	// Extract player names
	player1Name := strings.Join(args[argOffset:vsIndex], " ")
	player2Name := strings.Join(args[vsIndex+1:], " ")

	if player1Name == "" || player2Name == "" {
		b.sendMessage(s, m.ChannelID, "Please provide valid player names on both sides of 'vs'.")
		return
	}

	// Get stats for both players
	var stats1, stats2 *models.PlayerStats
	var err1, err2 error

	if isSeasonStats {
		stats1, err1 = b.nflClient.GetPlayerSeasonStats(player1Name)
		stats2, err2 = b.nflClient.GetPlayerSeasonStats(player2Name)
	} else if useSpecificWeek {
		stats1, err1 = b.nflClient.GetPlayerWeekStats(player1Name, specificSeason, specificWeek)
		stats2, err2 = b.nflClient.GetPlayerWeekStats(player2Name, specificSeason, specificWeek)
	} else {
		stats1, err1 = b.nflClient.GetPlayerStats(player1Name)
		stats2, err2 = b.nflClient.GetPlayerStats(player2Name)
	}

	// Handle errors
	if err1 != nil {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		b.sendMessage(s, m.ChannelID, fmt.Sprintf("Error getting stats for %s: %v", player1Name, err1))
		return
	}
	if err2 != nil {
		// Delete acknowledgment message
		if ack != nil {
			s.ChannelMessageDelete(m.ChannelID, ack.ID)
		}
		b.sendMessage(s, m.ChannelID, fmt.Sprintf("Error getting stats for %s: %v", player2Name, err2))
		return
	}

	// Create comparison embed
	comparisonTitle := "Player Comparison"
	if isSeasonStats {
		comparisonTitle = "Season Comparison (2024 Sample)"
	} else if useSpecificWeek {
		comparisonTitle = fmt.Sprintf("Week %d, %d Comparison", specificWeek, specificSeason)
	}

	// Delete acknowledgment message before sending results
	if ack != nil {
		s.ChannelMessageDelete(m.ChannelID, ack.ID)
	}

	embed := b.createComparisonEmbed(stats1, stats2, comparisonTitle)
	b.sendEmbed(s, m.ChannelID, embed)
}

// createComparisonEmbed creates a side-by-side comparison embed
func (b *Bot) createComparisonEmbed(stats1, stats2 *models.PlayerStats, title string) *discordgo.MessageEmbed {
	// Determine if players are same position for relevant comparisons
	samePosType := b.getSamePositionType(stats1.Position, stats2.Position)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("⚖️ %s", title),
		Color: 0x9932cc, // Purple color for comparisons
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Players",
				Value:  fmt.Sprintf("🔵 **%s** (%s, %s) vs 🔴 **%s** (%s, %s)", 
					   stats1.Name, stats1.Team, stats1.Position,
					   stats2.Name, stats2.Team, stats2.Position),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add position-specific comparisons
	if samePosType == "QB" && b.hasPassingStats(stats1) && b.hasPassingStats(stats2) {
		b.addPassingComparison(embed, stats1, stats2)
	}
	if samePosType == "RB" || (b.hasRushingStats(stats1) && b.hasRushingStats(stats2)) {
		b.addRushingComparison(embed, stats1, stats2)
	}
	if samePosType == "WR" || samePosType == "TE" || (b.hasReceivingStats(stats1) && b.hasReceivingStats(stats2)) {
		b.addReceivingComparison(embed, stats1, stats2)
	}

	// Add footer
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "🔵 = " + stats1.Name + " | 🔴 = " + stats2.Name + " | ⬆️ Better performance",
	}

	return embed
}

// getSamePositionType returns standardized position type for comparison
func (b *Bot) getSamePositionType(pos1, pos2 string) string {
	pos1 = strings.ToUpper(pos1)
	pos2 = strings.ToUpper(pos2)
	
	// Group similar positions
	if pos1 == pos2 {
		return pos1
	}
	
	// Check if both are similar types
	if (pos1 == "WR" || pos1 == "WR1" || pos1 == "WR2") && (pos2 == "WR" || pos2 == "WR1" || pos2 == "WR2") {
		return "WR"
	}
	if (pos1 == "RB" || pos1 == "RB1" || pos1 == "RB2") && (pos2 == "RB" || pos2 == "RB1" || pos2 == "RB2") {
		return "RB"
	}
	if (pos1 == "QB" || pos1 == "QB1") && (pos2 == "QB" || pos2 == "QB1") {
		return "QB"
	}
	if (pos1 == "TE" || pos1 == "TE1") && (pos2 == "TE" || pos2 == "TE1") {
		return "TE"
	}
	
	return "" // Different position types
}

// hasPassingStats checks if player has meaningful passing stats
func (b *Bot) hasPassingStats(stats *models.PlayerStats) bool {
	passingYards := b.getStatFloat(stats, "PassingYards")
	passingTDs := b.getStatFloat(stats, "PassingTouchdowns")
	passingAttempts := b.getStatFloat(stats, "PassingAttempts")
	return passingYards > 0 || passingTDs > 0 || passingAttempts > 0
}

// hasRushingStats checks if player has meaningful rushing stats
func (b *Bot) hasRushingStats(stats *models.PlayerStats) bool {
	rushingYards := b.getStatFloat(stats, "RushingYards")
	rushingTDs := b.getStatFloat(stats, "RushingTouchdowns")
	rushingAttempts := b.getStatFloat(stats, "RushingAttempts")
	return rushingYards > 0 || rushingTDs > 0 || rushingAttempts > 0
}

// hasReceivingStats checks if player has meaningful receiving stats
func (b *Bot) hasReceivingStats(stats *models.PlayerStats) bool {
	receivingYards := b.getStatFloat(stats, "ReceivingYards")
	receivingTDs := b.getStatFloat(stats, "ReceivingTouchdowns")
	receptions := b.getStatFloat(stats, "Receptions")
	return receivingYards > 0 || receivingTDs > 0 || receptions > 0
}

// addPassingComparison adds passing stats comparison to embed
func (b *Bot) addPassingComparison(embed *discordgo.MessageEmbed, stats1, stats2 *models.PlayerStats) {
	passingField := &discordgo.MessageEmbedField{
		Name:   "🏈 Passing Stats",
		Inline: false,
	}
	
	// Get passing stats
	yards1 := int(b.getStatFloat(stats1, "PassingYards"))
	yards2 := int(b.getStatFloat(stats2, "PassingYards"))
	tds1 := int(b.getStatFloat(stats1, "PassingTouchdowns"))
	tds2 := int(b.getStatFloat(stats2, "PassingTouchdowns"))
	ints1 := int(b.getStatFloat(stats1, "Interceptions"))
	ints2 := int(b.getStatFloat(stats2, "Interceptions"))
	
	// Passing yards
	var yardIcon1, yardIcon2 string
	if yards1 > yards2 {
		yardIcon1 = " ⬆️"
	} else if yards2 > yards1 {
		yardIcon2 = " ⬆️"
	}
	
	// Passing TDs
	var tdIcon1, tdIcon2 string
	if tds1 > tds2 {
		tdIcon1 = " ⬆️"
	} else if tds2 > tds1 {
		tdIcon2 = " ⬆️"
	}
	
	// Completion percentage
	compPct1 := b.calculateCompletionPct(stats1)
	compPct2 := b.calculateCompletionPct(stats2)
	var pctIcon1, pctIcon2 string
	if compPct1 > compPct2 {
		pctIcon1 = " ⬆️"
	} else if compPct2 > compPct1 {
		pctIcon2 = " ⬆️"
	}
	
	passingField.Value = fmt.Sprintf(
		"▫ **Yards:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **TDs:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **Comp%%:** 🔵 %.1f%%%s | 🔴 %.1f%%%s\n"+
		"▫ **INTs:** 🔵 %d | 🔴 %d",
		yards1, yardIcon1, yards2, yardIcon2,
		tds1, tdIcon1, tds2, tdIcon2,
		compPct1, pctIcon1, compPct2, pctIcon2,
		ints1, ints2,
	)
	
	embed.Fields = append(embed.Fields, passingField)
}

// addRushingComparison adds rushing stats comparison to embed
func (b *Bot) addRushingComparison(embed *discordgo.MessageEmbed, stats1, stats2 *models.PlayerStats) {
	rushingField := &discordgo.MessageEmbedField{
		Name:   "🏃 Rushing Stats",
		Inline: false,
	}
	
	// Get rushing stats
	yards1 := int(b.getStatFloat(stats1, "RushingYards"))
	yards2 := int(b.getStatFloat(stats2, "RushingYards"))
	tds1 := int(b.getStatFloat(stats1, "RushingTouchdowns"))
	tds2 := int(b.getStatFloat(stats2, "RushingTouchdowns"))
	attempts1 := int(b.getStatFloat(stats1, "RushingAttempts"))
	attempts2 := int(b.getStatFloat(stats2, "RushingAttempts"))
	
	// Rushing yards
	var yardIcon1, yardIcon2 string
	if yards1 > yards2 {
		yardIcon1 = " ⬆️"
	} else if yards2 > yards1 {
		yardIcon2 = " ⬆️"
	}
	
	// Rushing TDs
	var tdIcon1, tdIcon2 string
	if tds1 > tds2 {
		tdIcon1 = " ⬆️"
	} else if tds2 > tds1 {
		tdIcon2 = " ⬆️"
	}
	
	// YPC calculation
	ypc1 := b.calculateYPC(yards1, attempts1)
	ypc2 := b.calculateYPC(yards2, attempts2)
	var ypcIcon1, ypcIcon2 string
	if ypc1 > ypc2 {
		ypcIcon1 = " ⬆️"
	} else if ypc2 > ypc1 {
		ypcIcon2 = " ⬆️"
	}
	
	rushingField.Value = fmt.Sprintf(
		"▫ **Yards:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **TDs:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **Attempts:** 🔵 %d | 🔴 %d\n"+
		"▫ **YPC:** 🔵 %.1f%s | 🔴 %.1f%s",
		yards1, yardIcon1, yards2, yardIcon2,
		tds1, tdIcon1, tds2, tdIcon2,
		attempts1, attempts2,
		ypc1, ypcIcon1, ypc2, ypcIcon2,
	)
	
	embed.Fields = append(embed.Fields, rushingField)
}

// addReceivingComparison adds receiving stats comparison to embed
func (b *Bot) addReceivingComparison(embed *discordgo.MessageEmbed, stats1, stats2 *models.PlayerStats) {
	receivingField := &discordgo.MessageEmbedField{
		Name:   "👋 Receiving Stats",
		Inline: false,
	}
	
	// Get receiving stats
	yards1 := int(b.getStatFloat(stats1, "ReceivingYards"))
	yards2 := int(b.getStatFloat(stats2, "ReceivingYards"))
	tds1 := int(b.getStatFloat(stats1, "ReceivingTouchdowns"))
	tds2 := int(b.getStatFloat(stats2, "ReceivingTouchdowns"))
	receptions1 := int(b.getStatFloat(stats1, "Receptions"))
	receptions2 := int(b.getStatFloat(stats2, "Receptions"))
	
	// Receiving yards
	var yardIcon1, yardIcon2 string
	if yards1 > yards2 {
		yardIcon1 = " ⬆️"
	} else if yards2 > yards1 {
		yardIcon2 = " ⬆️"
	}
	
	// Receiving TDs
	var tdIcon1, tdIcon2 string
	if tds1 > tds2 {
		tdIcon1 = " ⬆️"
	} else if tds2 > tds1 {
		tdIcon2 = " ⬆️"
	}
	
	// Receptions
	var recIcon1, recIcon2 string
	if receptions1 > receptions2 {
		recIcon1 = " ⬆️"
	} else if receptions2 > receptions1 {
		recIcon2 = " ⬆️"
	}
	
	// YPR calculation
	ypr1 := b.calculateYPR(yards1, receptions1)
	ypr2 := b.calculateYPR(yards2, receptions2)
	var yprIcon1, yprIcon2 string
	if ypr1 > ypr2 {
		yprIcon1 = " ⬆️"
	} else if ypr2 > ypr1 {
		yprIcon2 = " ⬆️"
	}
	
	receivingField.Value = fmt.Sprintf(
		"▫ **Yards:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **TDs:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **Receptions:** 🔵 %d%s | 🔴 %d%s\n"+
		"▫ **YPR:** 🔵 %.1f%s | 🔴 %.1f%s",
		yards1, yardIcon1, yards2, yardIcon2,
		tds1, tdIcon1, tds2, tdIcon2,
		receptions1, recIcon1, receptions2, recIcon2,
		ypr1, yprIcon1, ypr2, yprIcon2,
	)
	
	embed.Fields = append(embed.Fields, receivingField)
}

// calculateCompletionPct calculates completion percentage
func (b *Bot) calculateCompletionPct(stats *models.PlayerStats) float64 {
	attempts := b.getStatFloat(stats, "PassingAttempts")
	completions := b.getStatFloat(stats, "PassingCompletions")
	if attempts == 0 {
		return 0.0
	}
	return (completions / attempts) * 100
}

// calculateYPC calculates yards per carry
func (b *Bot) calculateYPC(yards, attempts int) float64 {
	if attempts == 0 {
		return 0.0
	}
	return float64(yards) / float64(attempts)
}

// calculateYPR calculates yards per reception
func (b *Bot) calculateYPR(yards, receptions int) float64 {
	if receptions == 0 {
		return 0.0
	}
	return float64(yards) / float64(receptions)
}

// getStatFloat safely retrieves a stat as float64 from the player stats map
func (b *Bot) getStatFloat(stats *models.PlayerStats, statName string) float64 {
	if stats.Stats == nil {
		return 0.0
	}
	
	// Try direct key first
	value, exists := stats.Stats[statName]
	if !exists {
		// Try alternative field names (season vs week stats may use different keys)
		altNames := map[string][]string{
			"PassingYards":         {"passing_yards", "PassingYards"},
			"PassingTouchdowns":    {"passing_touchdowns", "PassingTouchdowns"},
			"PassingCompletions":   {"passing_completions", "PassingCompletions", "Completions"},
			"PassingAttempts":      {"passing_attempts", "PassingAttempts", "Attempts"},
			"Interceptions":        {"interceptions", "Interceptions"},
			"RushingYards":         {"rushing_yards", "RushingYards"},
			"RushingTouchdowns":    {"rushing_touchdowns", "RushingTouchdowns"},
			"RushingAttempts":      {"rushing_attempts", "RushingAttempts"},
			"ReceivingYards":       {"receiving_yards", "ReceivingYards"},
			"ReceivingTouchdowns":  {"receiving_touchdowns", "ReceivingTouchdowns"},
			"Receptions":           {"receptions", "Receptions"},
		}
		
		if alternatives, hasAlts := altNames[statName]; hasAlts {
			for _, altName := range alternatives {
				if altValue, altExists := stats.Stats[altName]; altExists {
					value = altValue
					exists = true
					break
				}
			}
		}
	}
	
	if !exists {
		return 0.0
	}
	
	// Handle different types of numeric values
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0.0
	}
}

// handleSilenceCommand handles the /s silence command
func (b *Bot) handleSilenceCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	b.silenceEnd = time.Now().Add(5 * time.Minute)
	log.Printf("[BOT] Bot silenced for 5 minutes by %s", m.Author.Username)
	
	// Delete the original /s command message immediately
	go func() {
		time.Sleep(100 * time.Millisecond) // Very brief delay
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}()
	
	// Send temporary message that will be deleted after 3 seconds
	msg, err := s.ChannelMessageSend(m.ChannelID, "🔇 Bot silenced for 5 minutes")
	if err != nil {
		log.Printf("Error sending silence message: %v", err)
		return
	}

	// Delete the confirmation message after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		s.ChannelMessageDelete(m.ChannelID, msg.ID)
	}()
}

// hasAllowedRole checks if user has the required role to interact with bot
func (b *Bot) hasAllowedRole(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	return b.hasRole(s, m, b.allowedRole)
}

// hasVisibilityRole checks if user has the required role to see bot messages
func (b *Bot) hasVisibilityRole(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	return b.hasRole(s, m, b.visibilityRole)
}

// hasRole checks if user has a specific role
func (b *Bot) hasRole(s *discordgo.Session, m *discordgo.MessageCreate, roleName string) bool {
	if roleName == "" {
		return true // No role required
	}
	
	// Get guild member to check roles
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		log.Printf("Error getting guild member: %v", err)
		return false
	}
	
	// Check if user has the required role
	for _, roleID := range member.Roles {
		// Get role info
		role, err := s.State.Role(m.GuildID, roleID)
		if err != nil {
			continue
		}
		
		// Check if role name matches
		if strings.EqualFold(role.Name, roleName) {
			return true
		}
	}
	
	return false
}

// hasAllowedRoleForInteraction checks if user has the required role to interact with bot (for slash commands)
func (b *Bot) hasAllowedRoleForInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	return b.hasRoleForInteraction(s, i, b.allowedRole)
}

// hasVisibilityRoleForInteraction checks if user has the required role to see bot messages (for slash commands)
func (b *Bot) hasVisibilityRoleForInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	return b.hasRoleForInteraction(s, i, b.visibilityRole)
}

// hasRoleForInteraction checks if user has a specific role (for slash commands)
func (b *Bot) hasRoleForInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, roleName string) bool {
	if roleName == "" {
		return true // No role required
	}
	
	// Get guild member to check roles
	member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
	if err != nil {
		log.Printf("Error getting guild member: %v", err)
		return false
	}
	
	// Check if user has the required role
	for _, roleID := range member.Roles {
		// Get role info
		role, err := s.State.Role(i.GuildID, roleID)
		if err != nil {
			continue
		}
		
		// Check if role name matches
		if strings.EqualFold(role.Name, roleName) {
			return true
		}
	}
	
	return false
}

// respondInteraction sends a response to slash command interaction (always ephemeral if visibility role is configured)
func (b *Bot) respondInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) error {
	isEphemeral := b.visibilityRole != ""
	
	data := &discordgo.InteractionResponseData{
		Content: content,
	}
	
	if isEphemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

// respondInteractionEmbed sends an embed response to slash command interaction (always ephemeral if visibility role is configured)
func (b *Bot) respondInteractionEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	isEphemeral := b.visibilityRole != ""
	
	data := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	}
	
	if isEphemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

// followupInteraction sends a followup message to slash command interaction (always ephemeral if visibility role is configured)
func (b *Bot) followupInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) error {
	isEphemeral := b.visibilityRole != ""
	
	data := &discordgo.WebhookParams{
		Content: content,
	}
	
	if isEphemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	
	_, err := s.FollowupMessageCreate(i.Interaction, true, data)
	return err
}

// followupInteractionEmbed sends a followup embed to slash command interaction (always ephemeral if visibility role is configured)
func (b *Bot) followupInteractionEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	isEphemeral := b.visibilityRole != ""
	
	data := &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	}
	
	if isEphemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	
	_, err := s.FollowupMessageCreate(i.Interaction, true, data)
	return err
}

// sendMessage sends a text message to a Discord channel
func (b *Bot) sendMessage(s *discordgo.Session, channelID, message string) {
	_, err := s.ChannelMessageSend(channelID, message)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// sendEmbed sends an embed message to a Discord channel
func (b *Bot) sendEmbed(s *discordgo.Session, channelID string, embed *discordgo.MessageEmbed) {
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
}

// handleSlashHelp handles the /help slash command
func (b *Bot) handleSlashHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "🏈 NFL Discord Bot - Slash Commands Guide",
		Description: "**Intelligent NFL data with real-time stats, schedules, and scores**\n\n" +
			"*Smart week detection: Wednesday shows previous week, Thursday-Monday shows current week*",
		Color: 0x013369,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "📊 Player Statistics",
				Value: "`/stats player:<name>` - Current week stats\n" +
					   "`/stats player:<name> type:Season` - Season totals\n" +
					   "`/stats player:<name> week:<#>` - Specific week\n" +
					   "*Examples: `/stats player:Josh Allen`, `/stats player:Saquon Barkley week:5`*",
				Inline: false,
			},
			{
				Name:  "⚖️ Player Comparisons",
				Value: "`/compare player1:<name> player2:<name>` - Compare current week\n" +
					   "`/compare player1:<name> player2:<name> type:Season` - Compare season\n" +
					   "`/compare player1:<name> player2:<name> week:<#>` - Compare specific week\n" +
					   "*Examples: `/compare player1:Josh Allen player2:Mahomes`*",
				Inline: false,
			},
			{
				Name:  "🏟️ Team Information",
				Value: "`/team team:<name>` - Complete team details\n" +
					   "*Shows: Conference, division, coach, stadium*\n" +
					   "*Examples: `/team team:Bills`, `/team team:Eagles`*",
				Inline: false,
			},
			{
				Name:  "📅 Team Schedule",
				Value: "`/schedule team:<name>` - Full season schedule\n" +
					   "*Shows: Game dates, opponents, scores, BYE weeks*\n" +
					   "*Examples: `/schedule team:Cowboys`, `/schedule team:Patriots`*",
				Inline: false,
			},
			{
				Name:  "🔴 Live Scores",
				Value: "`/scores` - Current week's games and scores\n" +
					   "*Shows: Live games, completed games, upcoming games*",
				Inline: false,
			},
			{
				Name:  "⚡ Smart Features",
				Value: "• **Ephemeral Responses** - Only you can see responses (if configured)\n" +
					   "• **Auto Week Detection** - Always shows current NFL week\n" +
					   "• **5-Minute Caching** - Fast responses, reduced API calls\n" +
					   "• **Real-Time Data** - Live stats from SportsData.io",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "🤖 Data updates every 5 minutes | 📡 Powered by SportsData.io | ⚡ Slash Commands",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := b.respondInteractionEmbed(s, i, embed)
	if err != nil {
		log.Printf("Error responding to help slash command: %v", err)
	}
}

// handleSlashStats handles the /stats slash command
func (b *Bot) handleSlashStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		err := b.respondInteraction(s, i, "Please provide a player name.")
		if err != nil {
			log.Printf("Error responding to stats slash command: %v", err)
		}
		return
	}

	// Parse options
	var playerName string
	var statsType string = "current"
	var week *int64
	var year *int64

	for _, option := range options {
		switch option.Name {
		case "player":
			playerName = option.StringValue()
		case "type":
			statsType = option.StringValue()
		case "week":
			weekVal := option.IntValue()
			week = &weekVal
		case "year":
			yearVal := option.IntValue()
			year = &yearVal
		}
	}

	// Send initial response
	var responseMsg string
	if statsType == "season" {
		responseMsg = "⏳ Fetching season stats... (this may take a moment)"
	} else if week != nil {
		responseMsg = "⏳ Fetching week-specific stats..."
	} else {
		responseMsg = "⏳ Fetching current week stats..."
	}

	err := b.respondInteraction(s, i, responseMsg)
	if err != nil {
		log.Printf("Error sending initial stats response: %v", err)
		return
	}

	// Process stats request asynchronously
	go b.processSlashStatsRequest(s, i, playerName, statsType, week, year)
}

// handleSlashCompare handles the /compare slash command
func (b *Bot) handleSlashCompare(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) < 2 {
		err := b.respondInteraction(s, i, "Please provide both player names for comparison.")
		if err != nil {
			log.Printf("Error responding to compare slash command: %v", err)
		}
		return
	}

	// Parse options
	var player1, player2 string
	var statsType string = "current"
	var week *int64

	for _, option := range options {
		switch option.Name {
		case "player1":
			player1 = option.StringValue()
		case "player2":
			player2 = option.StringValue()
		case "type":
			statsType = option.StringValue()
		case "week":
			weekVal := option.IntValue()
			week = &weekVal
		}
	}

	err := b.respondInteraction(s, i, "⏳ Fetching player comparison...")
	if err != nil {
		log.Printf("Error sending initial compare response: %v", err)
		return
	}

	// Process compare request asynchronously
	go b.processSlashCompareRequest(s, i, player1, player2, statsType, week)
}

// handleSlashTeam handles the /team slash command
func (b *Bot) handleSlashTeam(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		err := b.respondInteraction(s, i, "Please provide a team name.")
		if err != nil {
			log.Printf("Error responding to team slash command: %v", err)
		}
		return
	}

	teamName := options[0].StringValue()

	err := b.respondInteraction(s, i, "⏳ Fetching team information...")
	if err != nil {
		log.Printf("Error sending initial team response: %v", err)
		return
	}

	// Process team request asynchronously
	go b.processSlashTeamRequest(s, i, teamName)
}

// handleSlashSchedule handles the /schedule slash command
func (b *Bot) handleSlashSchedule(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		err := b.respondInteraction(s, i, "Please provide a team name.")
		if err != nil {
			log.Printf("Error responding to schedule slash command: %v", err)
		}
		return
	}

	teamName := options[0].StringValue()

	err := b.respondInteraction(s, i, "⏳ Fetching team schedule...")
	if err != nil {
		log.Printf("Error sending initial schedule response: %v", err)
		return
	}

	// Process schedule request asynchronously
	go b.processSlashScheduleRequest(s, i, teamName)
}

// handleSlashScores handles the /scores slash command
func (b *Bot) handleSlashScores(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := b.respondInteraction(s, i, "⏳ Fetching current week scores...")
	if err != nil {
		log.Printf("Error sending initial scores response: %v", err)
		return
	}

	// Process scores request asynchronously
	go b.processSlashScoresRequest(s, i)
}

// processSlashStatsRequest processes the stats request and sends a followup message
func (b *Bot) processSlashStatsRequest(s *discordgo.Session, i *discordgo.InteractionCreate, playerName, statsType string, week, year *int64) {
	// Determine what type of stats to fetch
	var isSeasonStats bool
	var specificWeek int
	var specificSeason int
	var useSpecificWeek bool
	
	if statsType == "season" {
		isSeasonStats = true
	} else if week != nil {
		useSpecificWeek = true
		specificWeek = int(*week)
		if year != nil {
			specificSeason = int(*year)
		} else {
			specificSeason = 2025 // Default to current season
		}
	}
	
	// Get player stats from NFL client
	var stats *models.PlayerStats
	var err error
	
	if isSeasonStats {
		stats, err = b.nflClient.GetPlayerSeasonStats(playerName)
	} else if useSpecificWeek {
		stats, err = b.nflClient.GetPlayerWeekStats(playerName, specificSeason, specificWeek)
	} else {
		stats, err = b.nflClient.GetPlayerStats(playerName)
	}
	
	if err != nil {
		statsType := "current week"
		if isSeasonStats {
			statsType = "season sample"
		} else if useSpecificWeek {
			statsType = fmt.Sprintf("Week %d, %d", specificWeek, specificSeason)
		}
		errorMsg := fmt.Sprintf("Error getting %s stats for %s: %v", statsType, playerName, err)
		b.followupInteraction(s, i, errorMsg)
		return
	}
	
	// Create embed with player stats
	statsTitle := "Current Week Stats (2025)"
	if isSeasonStats {
		statsTitle = "2024 Sample Stats (6 games)"
	} else if useSpecificWeek {
		statsTitle = fmt.Sprintf("Week %d, %d Stats", specificWeek, specificSeason)
	}
	
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s - %s", stats.Name, statsTitle),
		Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Team",
				Value:  stats.Team,
				Inline: true,
			},
			{
				Name:   "Position",
				Value:  stats.Position,
				Inline: true,
			},
			{
				Name:   "Season Stats",
				Value:  stats.GetStatsString(),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from NFL API",
		},
	}
	
	err = b.followupInteractionEmbed(s, i, embed)
	if err != nil {
		log.Printf("Error sending stats embed followup: %v", err)
	}
}

// processSlashCompareRequest processes the compare request and sends a followup message
func (b *Bot) processSlashCompareRequest(s *discordgo.Session, i *discordgo.InteractionCreate, player1, player2, statsType string, week *int64) {
	// Determine what type of stats to fetch
	var isSeasonStats bool
	var specificWeek int
	var specificSeason int
	var useSpecificWeek bool
	
	if statsType == "season" {
		isSeasonStats = true
	} else if week != nil {
		useSpecificWeek = true
		specificWeek = int(*week)
		specificSeason = 2025 // Default to current season for comparisons
	}
	
	// Get stats for both players
	var stats1, stats2 *models.PlayerStats
	var err1, err2 error
	
	if isSeasonStats {
		stats1, err1 = b.nflClient.GetPlayerSeasonStats(player1)
		stats2, err2 = b.nflClient.GetPlayerSeasonStats(player2)
	} else if useSpecificWeek {
		stats1, err1 = b.nflClient.GetPlayerWeekStats(player1, specificSeason, specificWeek)
		stats2, err2 = b.nflClient.GetPlayerWeekStats(player2, specificSeason, specificWeek)
	} else {
		stats1, err1 = b.nflClient.GetPlayerStats(player1)
		stats2, err2 = b.nflClient.GetPlayerStats(player2)
	}
	
	// Handle errors
	if err1 != nil {
		errorMsg := fmt.Sprintf("Error getting stats for %s: %v", player1, err1)
		b.followupInteraction(s, i, errorMsg)
		return
	}
	if err2 != nil {
		errorMsg := fmt.Sprintf("Error getting stats for %s: %v", player2, err2)
		b.followupInteraction(s, i, errorMsg)
		return
	}
	
	// Create comparison embed
	comparisonTitle := "Player Comparison"
	if isSeasonStats {
		comparisonTitle = "Season Comparison (2024 Sample)"
	} else if useSpecificWeek {
		comparisonTitle = fmt.Sprintf("Week %d, %d Comparison", specificWeek, specificSeason)
	}
	
	embed := b.createComparisonEmbed(stats1, stats2, comparisonTitle)
	err := b.followupInteractionEmbed(s, i, embed)
	if err != nil {
		log.Printf("Error sending compare embed followup: %v", err)
	}
}

// processSlashTeamRequest processes the team request and sends a followup message
func (b *Bot) processSlashTeamRequest(s *discordgo.Session, i *discordgo.InteractionCreate, teamName string) {
	// Get team info from NFL client
	teamInfo, err := b.nflClient.GetTeamInfo(teamName)
	if err != nil {
		errorMsg := fmt.Sprintf("Error getting team info for %s: %v", teamName, err)
		b.followupInteraction(s, i, errorMsg)
		return
	}
	
	// Create embed with team info
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🏈 %s %s", teamInfo.City, teamInfo.Name),
		Color: 0xff6600,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Conference",
				Value:  teamInfo.Conference,
				Inline: true,
			},
			{
				Name:   "Division",
				Value:  teamInfo.Division,
				Inline: true,
			},
			{
				Name:   "Head Coach",
				Value:  teamInfo.Coach,
				Inline: true,
			},
			{
				Name:   "Stadium",
				Value:  teamInfo.Stadium,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Team data from NFL API",
		},
	}
	
	err = b.followupInteractionEmbed(s, i, embed)
	if err != nil {
		log.Printf("Error sending team embed followup: %v", err)
	}
}

// processSlashScheduleRequest processes the schedule request and sends a followup message
func (b *Bot) processSlashScheduleRequest(s *discordgo.Session, i *discordgo.InteractionCreate, teamName string) {
	// Get team schedule from NFL client
	schedule, err := b.nflClient.GetTeamSchedule(teamName)
	if err != nil {
		errorMsg := fmt.Sprintf("Error getting schedule for %s: %v", teamName, err)
		b.followupInteraction(s, i, errorMsg)
		return
	}
	
	// Create embed with schedule (show first 10 games to avoid too long message)
	var scheduleText string
	gamesToShow := schedule.Games
	if len(gamesToShow) > 10 {
		gamesToShow = gamesToShow[:10]
	}
	
	for _, game := range gamesToShow {
		// Check if this is a BYE week
		if game.HomeTeam == "BYE" || game.AwayTeam == "BYE" {
			scheduleText += fmt.Sprintf("**Week %d**: 🛌 **BYE WEEK** - Rest and Recovery\n", game.Week)
			continue
		}
		
		gameDate := game.GameTime.Format("Jan 2, 3:04 PM")
		if game.IsCompleted() {
			scheduleText += fmt.Sprintf("**Week %d**: %s @ %s - %s %d-%d (Final)\n", 
				game.Week, game.AwayTeam, game.HomeTeam, game.Winner(), game.AwayScore, game.HomeScore)
		} else if game.IsLive() {
			scheduleText += fmt.Sprintf("**Week %d**: %s @ %s - %d-%d (LIVE)\n", 
				game.Week, game.AwayTeam, game.HomeTeam, game.AwayScore, game.HomeScore)
		} else {
			scheduleText += fmt.Sprintf("**Week %d**: %s @ %s - %s\n", 
				game.Week, game.AwayTeam, game.HomeTeam, gameDate)
		}
	}
	
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📅 %s Schedule (%d Season)", schedule.TeamName, schedule.Season),
		Color: 0x00ff00,
		Description: scheduleText,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing %d of %d games", len(gamesToShow), len(schedule.Games)),
		},
	}
	
	err = b.followupInteractionEmbed(s, i, embed)
	if err != nil {
		log.Printf("Error sending schedule embed followup: %v", err)
	}
}

// processSlashScoresRequest processes the scores request and sends a followup message
func (b *Bot) processSlashScoresRequest(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get live scores from NFL client
	liveScores, err := b.nflClient.GetLiveScores()
	if err != nil {
		errorMsg := fmt.Sprintf("Error getting live scores: %v", err)
		b.followupInteraction(s, i, errorMsg)
		return
	}
	
	if len(liveScores) == 0 {
		b.followupInteraction(s, i, "No games found for this week.")
		return
	}
	
	// Create embed with live scores
	var scoresText string
	liveCount := 0
	completedCount := 0
	
	for _, score := range liveScores {
		if score.IsLive() {
			scoresText += fmt.Sprintf("🔴 **%s** - %s\n", "LIVE", score.GetScoreString())
			liveCount++
		} else if score.IsCompleted() {
			scoresText += fmt.Sprintf("✅ **FINAL** - %s\n", score.GetScoreString())
			completedCount++
		} else {
			gameTime := score.GameTime.Format("Jan 2, 3:04 PM")
			scoresText += fmt.Sprintf("📅 **%s** - %s @ %s\n", gameTime, score.AwayTeam, score.HomeTeam)
		}
	}
	
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🏈 NFL Scores - Week %d", liveScores[0].Week),
		Color: 0x013369,
		Description: scoresText,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d live, %d completed, %d total games", liveCount, completedCount, len(liveScores)),
		},
	}
	
	err = b.followupInteractionEmbed(s, i, embed)
	if err != nil {
		log.Printf("Error sending scores embed followup: %v", err)
	}
}
