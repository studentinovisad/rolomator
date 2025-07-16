package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	GuildID   string            `json:"guildID"`
	ChannelID string            `json:"channelID"`
	MessageID string            `json:"messageID"`
	Reactions map[string]string `json:"reactions"` // emoji -> roleID
}

var (
	config    Config
	roleCache = make(map[string]map[string]string) // guildID -> roleID -> roleName
)

func main() {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		panic("DISCORD_BOT_TOKEN environment variable is not set")
	}

	if err := loadConfig("config.json"); err != nil {
		panic("failed to load config: " + err.Error())
	}

	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		panic("failed to create bot: " + err.Error())
	}

	dg.AddHandler(reactionAdd)
	dg.AddHandler(reactionRemove)

	err = dg.Open()
	if err != nil {
		panic("failed to open connection: " + err.Error())
	}
	defer dg.Close()

	// Cache roles
	err = cacheRoles(dg)
	if err != nil {
		fmt.Println("Error caching roles:", err)
	}

	// Add reactions to message on startup
	err = addStartupReactions(dg)
	if err != nil {
		fmt.Println("Error adding startup reactions:", err)
	}

	fmt.Println("Bot is running. Press CTRL+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}

func reactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.MessageID != config.MessageID || r.UserID == s.State.User.ID {
		return
	}
	if roleID, ok := config.Reactions[r.Emoji.Name]; ok {
		err := s.GuildMemberRoleAdd(r.GuildID, r.UserID, roleID)
		if err != nil {
			fmt.Println("Error adding role:", err)
			return
		}
		roleName := roleCache[r.GuildID][roleID]
		sendDM(s, r.UserID, fmt.Sprintf("✅ You were given the role: **%s** (%s)", roleName, r.Emoji.Name))
	}
}

func reactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.MessageID != config.MessageID || r.UserID == s.State.User.ID {
		return
	}
	if roleID, ok := config.Reactions[r.Emoji.Name]; ok {
		err := s.GuildMemberRoleRemove(r.GuildID, r.UserID, roleID)
		if err != nil {
			fmt.Println("Error removing role:", err)
			return
		}
		roleName := roleCache[r.GuildID][roleID]
		sendDM(s, r.UserID, fmt.Sprintf("❌ The role **%s** (%s) was removed from you", roleName, r.Emoji.Name))
	}
}

func sendDM(s *discordgo.Session, userID string, message string) {
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		fmt.Println("Failed to create DM channel:", err)
		return
	}
	_, err = s.ChannelMessageSend(channel.ID, message)
	if err != nil {
		fmt.Println("Failed to send DM:", err)
	}
}

func loadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&config)
}

func cacheRoles(s *discordgo.Session) error {
	roles, err := s.GuildRoles(config.GuildID)
	if err != nil {
		return err
	}

	roleCache[config.GuildID] = make(map[string]string)
	for _, role := range roles {
		roleCache[config.GuildID][role.ID] = role.Name
	}
	fmt.Printf("Cached %d roles for guild %s\n", len(roles), config.GuildID)
	return nil
}

func addStartupReactions(s *discordgo.Session) error {
	for emoji := range config.Reactions {
		err := s.MessageReactionAdd(config.ChannelID, config.MessageID, emoji)
		if err != nil {
			fmt.Printf("Failed to add reaction %s: %v\n", emoji, err)
		} else {
			fmt.Printf("Added reaction: %s\n", emoji)
		}
	}
	return nil
}
