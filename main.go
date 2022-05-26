package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

const MAX_LINE_LENGTH = 2048
const MAX_LOG_FILE_LENTH = 4000

var current_line = 0
var regex = regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\] \[Server thread/INFO\]: (<.*|[\w ]+ (joined|left) the game)$`)
var first = true

var messageCache = []string{}

var insults = []string{}

type Config struct {
	TOKEN      string
	CHATBRIDGE string
	ADMINROLE  string
}

var config = Config{}

func parse_bridge() {
	// TODO replace with cli arg?
	file, err := os.Open("/tmp/SAG-bot")
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var contents []string
	i := 0
	for scanner.Scan() {
		i++
		if i < current_line {
			continue
		}
		current_line = i
		contents = append(contents, scanner.Text())
	}
	current_line = i + 1
	if first {
		first = false
		return
	}
	file.Close()

	for _, line := range contents {
		if len(line) <= 33 {
			continue
		}
		check_line(line[33:])
		if regex.Match([]byte(line)) {
			// send to dis
			messageCache = append(messageCache, line[33:])
		}
	}

	if current_line > MAX_LINE_LENGTH {
		err := os.Remove("/tmp/SAG-bot")
		if err != nil {
			fmt.Println(err)
		}
		cmd := exec.Command("tmux", "pipe-pane", "-t", "SAG", "cat > /tmp/SAG-bot")
		cmd.Output()
	}
}

type Ban struct {
	name    string
	expires int64
}

type BanWrapper struct {
	list []Ban
}

var bans = []Ban{}

func load_bans() {
	contents, err := ioutil.ReadFile("./bans.txt")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	for _, entry := range strings.Split(string(contents), "\n") {
		if len(entry) < 8 {
			continue
		}
		data := strings.Split(entry, "|")
		num, _ := strconv.ParseInt(data[1], 10, 64)
		bans = append(bans, Ban{
			name:    data[0],
			expires: num,
		})
	}
}

func assemble_message(tokens []string, final string) bool {
	words := len(strings.Fields(final))
	if len(tokens) < words {
		return false
	}
	return strings.Trim(strings.Join(tokens[0:words], " "), "\n") == strings.Trim(final, "\n")
}

func check_line(line string) {
	t := strings.Fields(line)
	username := t[0:1]
	t = t[1:]
	deathMessages := [44]string{
		"was shot by",
		"was pummeled by",
		"was pricked to death",
		"walked into a cactus whilst trying to escape",
		"drowned",
		"experienced kinetic energy",
		"blew up",
		"was blown up by",
		"was killed by",
		"hit the ground too hard",
		"fell from a high place",
		"fell off",
		"fell while climbing",
		"was impaled on a ",
		"was squashed",
		"was skewered by a falling stalactite",
		"went up in flames",
		"walked into fire whilst fighting",
		"burned to death",
		"was burnt to a crisp whilst fighting",
		"went off with a bang",
		"tried to swim in lava",
		"was struck by lightning",
		"discovered the floor was lava",
		"walked into danger zone due to ",
		"was killed by",
		"froze to death",
		"was frozen to death by",
		"was slain by",
		"was fireballed by",
		"was stung to death",
		"was shot by a skull from",
		"starved to death",
		"suffocated in a wall",
		"was squished",
		"was poked to death by a sweet berry bush",
		"was killed trying to hurt",
		"was killed by",
		"was impaled",
		"fell out of the world",
		"didn't want to live in the same world as",
		"withered away",
		"was roasted in dragon breath",
		"died",
	}
	dt := time.Now()
	for _, m := range deathMessages {
		if assemble_message(t, m) {
			ban_person(username[0])
			messageCache = append(messageCache, fmt.Sprintf(
				"%s was banned for 24 hours, timestamp: %02d/%02d %02d:%02d:%02d",
				username[0],
				dt.Local().Month(),
				dt.Local().Day(),
				dt.Local().Hour(),
				dt.Local().Local().Minute(),
				dt.Local().Second(),
			))
			if len(insults) < 2 {
				continue
			}
			randomI := rand.Int() % len(insults)
			messageCache = append(messageCache, strings.ReplaceAll(insults[randomI], "NAME", username[0]))
		}
	}

}

func ban_person(username string) {
	bans = append(bans, Ban{
		name:    username,
		expires: time.Now().Unix() + 86400,
	})
	save_banlist()
	dt := time.Now()
	cmd := exec.Command(
		"tmux",
		"send-keys",
		"-t",
		"SAG",
		fmt.Sprintf(
			"ban %s Banned for 24 hours from %02d/%02d %02d:%02d:%02d",
			username,
			dt.Local().Month(),
			dt.Local().Day(),
			dt.Local().Hour(),
			dt.Local().Local().Minute(),
			dt.Local().Second(),
		),
		"Enter",
	)
	cmd.Output()
}

func removeBan(list []Ban, target Ban) []Ban {
	target_index := 0
	for i, e := range list {
		if e.name == target.name {
			target_index = i
		}
	}
	return append(list[:target_index], list[target_index+1:]...)
}

func remove(list []string, target string) []string {
	target_index := 0
	for i, e := range list {
		if e == target {
			target_index = i
		}
	}
	return append(list[:target_index], list[target_index+1:]...)
}

func save_banlist() {
	newBans := []string{}
	for _, ban := range bans {
		newBans = append(newBans, fmt.Sprintf("%s|%d", ban.name, ban.expires))
	}
	_ = ioutil.WriteFile("./bans.txt", []byte(strings.Join(newBans, "\n")), 0644)
}

func unban_person() {
	for _, ban := range bans {
		if ban.expires > time.Now().Unix() {
			continue
		}
		bans = removeBan(bans, ban)
		save_banlist()
		cmd := exec.Command("tmux", "send-keys", "-t", "SAG", fmt.Sprintf("pardon %s", ban.name), "Enter")
		cmd.Output()
	}
}

func clearFormatting(message string) string {
	message = strings.ReplaceAll(message, "\\", "")
	message = strings.ReplaceAll(message, "\n", "")
	message = strings.ReplaceAll(message, "\r", "")
	message = strings.ReplaceAll(message, "\"", "\\\"")
	return message
}

func clearFormattingOutbound(message string) string {
	message = strings.ReplaceAll(message, "_", "\\_")
	message = strings.ReplaceAll(message, "\"", "\\\"")
	message = strings.ReplaceAll(message, "\r", "\\r")
	return message
}

func includes(roles []string, target string) bool {
	for _, i := range roles {
		if i == target {
			return true
		}
	}
	return false
}

func formatDuration(d uint64) string {
	h := d / 3600
	d -= h * 3600
	m := d / 60
	d -= m * 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, d)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.ChannelID == config.CHATBRIDGE {
		msg := clearFormatting(m.Content)
		if len(msg) < 1 {
			return
		}
		cmd := exec.Command("tmux", "send-keys", "-t", "SAG", fmt.Sprintf("tellraw @a {\"text\":\"§3[§f%s§3]§f %s\"}", m.Author.Username, msg), "Enter")
		cmd.Output()
	}
	splits := strings.Fields(m.Content)
	if len(splits) < 1 {
		return
	}

	if splits[0] == "listbans" {
		banList := []string{}
		for _, ban := range bans {
			banList = append(
				banList,
				fmt.Sprintf(
					"IGN: %s \t %s left until unban (HR:MM:SS)",
					ban.name,
					formatDuration(uint64(ban.expires)-uint64(time.Now().Unix())),
				),
			)
		}
		s.ChannelMessageSend(m.ChannelID, strings.Join(banList, "\n"))
		if len(banList) < 1 {
			s.ChannelMessageSend(m.ChannelID, "no bans at this time")
		}
	}

	if splits[0] == "listinsult" {
		s.ChannelMessageSend(m.ChannelID, strings.Join(insults, "\n"))
		if len(insults) < 1 {
			s.ChannelMessageSend(m.ChannelID, "no insults registered")
		}
	}
	if !includes(m.Member.Roles, config.ADMINROLE) || len(splits) < 2 {
		return
	}
	if splits[0] == "insult" {
		insults = append(insults, strings.Join(splits[1:], " "))
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("registered %s", strings.Join(splits[1:], " ")))
		save_insults()
	}
	if splits[0] == "uninsult" {
		insults = remove(insults, strings.Join(splits[1:], " "))
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("removed %s", strings.Join(splits[1:], " ")))
		save_insults()
	}
}

func save_insults() {
	_ = ioutil.WriteFile("./insults.txt", []byte(strings.Join(insults, "\n")), 0644)
}

func main() {

	fmt.Println("starting bot")
	contents, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	insultsContents, ierr := ioutil.ReadFile("./insults.txt")
	if ierr == nil {
		insults = strings.Split(string(insultsContents), "\n")
		for _, i := range insults {
			fmt.Println("registered insult: ", i)
		}
	}

	uerr := yaml.Unmarshal(contents, &config)
	if uerr != nil {
		log.Fatal(uerr)
		os.Exit(1)
	}
	dg, err := discordgo.New("Bot " + config.TOKEN)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	cmd := exec.Command("tmux", "pipe-pane", "-t", "SAG", "cat > /tmp/SAG-bot")
	cmd.Output()
	_, e := os.Stat("./bans.txt")
	if errors.Is(e, os.ErrNotExist) {
		fmt.Println("creating new ban list")
		ban, err := os.Create("./bans.txt")
		if err != nil {
			log.Fatal(err)
		}
		ban.Close()
	}
	load_bans()
	save_banlist()
	for _, ban := range bans {
		fmt.Println("\t", ban.name, "\t", ban.expires)
	}
	for {
		start := time.Now()
		unban_person()
		parse_bridge()
		if len(messageCache) > 0 {
			dg.ChannelMessageSend(config.CHATBRIDGE, clearFormattingOutbound(strings.Join(messageCache, "\n")))
			messageCache = make([]string, 0)
		}
		time.Sleep(time.Duration(250000000) - time.Since(start))
	}
}
