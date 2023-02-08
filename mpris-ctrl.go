package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus"
	"go.senan.xyz/mpris-ctrl/mpris"
)

func main() {
	playerIndex := flag.Int("player-index", 0, "optional index of the player to use")
	flag.Parse()

	var command string
	if args := flag.Args(); len(args) > 0 {
		command = args[0]
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error connecting to bus: %v", err)
		return
	}
	defer conn.Close()

	output, err := run(conn, command, *playerIndex)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error getting output: %v", err)
		return
	}
	if output == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "%s\n", output)
}

func getActivePlayers(conn *dbus.Conn) ([]*mpris.Player, error) {
	names, err := mpris.List(conn)
	if err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}
	var players []*mpris.Player
	for _, name := range names {
		player := mpris.New(conn, name)
		if player.GetPlaybackStatus() == mpris.PlaybackStopped {
			continue
		}
		players = append(players, player)
	}
	return players, nil
}

func run(conn *dbus.Conn, command string, playerIndex int) (string, error) {
	players, err := getActivePlayers(conn)
	if err != nil {
		return "", fmt.Errorf("get players: %w", err)
	}
	if len(players) == 0 {
		return "", err
	}
	playerIndex = pmod(playerIndex, len(players))
	player := players[playerIndex]

	switch command {
	case "play-pause":
		prevStatus := player.GetPlaybackStatus()
		player.PlayPause()
		waitUntil(func() bool {
			return player.GetPlaybackStatus() != prevStatus
		})
	case "stop":
		prevStatus := player.GetPlaybackStatus()
		player.Stop()
		waitUntil(func() bool {
			status := player.GetPlaybackStatus()
			return status != prevStatus || status == mpris.PlaybackStopped
		})
	case "next":
		player.Next()
	case "previous":
		player.Previous()
	}

	players, err = getActivePlayers(conn)
	if err != nil {
		return "", fmt.Errorf("get players: %w", err)
	}
	if len(players) == 0 {
		return "", err
	}
	playerIndex = pmod(playerIndex, len(players))
	player = players[playerIndex]

	var parts []string
	parts = append(parts, strings.ToLower(string(player.GetPlaybackStatus())))
	if len(players) > 1 {
		parts = append(parts, fmt.Sprintf("%d/%d", playerIndex+1, len(players)))
	}

	metadata := player.GetMetadata()
	title := strings.TrimSpace(metadata["xesam:title"])
	if title == "" {
		return "", nil
	}

	parts = append(parts, fmt.Sprintf(`‘%s’`, trunc(title, 40, "…")))
	output := fmt.Sprintf("%v", strings.Join(parts, " "))

	return output, nil
}

func trunc(in string, max int, ellip string) string {
	if len(in) <= max {
		return in
	}
	return in[:max] + ellip
}

func pmod(a, b int) int {
	return ((a % b) + b) % b
}

const (
	statusChangePollInterval = 75 * time.Millisecond
)

func waitUntil(cb func() bool) {
	for i := 0; i < 20; i++ {
		if cb() {
			break
		}
		time.Sleep(statusChangePollInterval)
	}
}
