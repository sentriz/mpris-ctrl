package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus"
	"go.senan.xyz/filelock"
	"go.senan.xyz/mpris-ctrl/mpris"
)

func main() {
	var command string
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	output, err := getOutput(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if output == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "%s\n", output)
}

const (
	programName              = "mpris-ctrl"
	statusChangePollInterval = 75 * time.Millisecond
)

func getOutput(command string) (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("get user cache dir: %w", err)
	}
	cacheDir := filepath.Join(userCacheDir, programName)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	indexPath := filepath.Join(cacheDir, "lock")
	idxFile, err := filelock.New(indexPath)
	if err != nil {
		return "", fmt.Errorf("create lock: %w", err)
	}
	idxFile.Lock()
	defer idxFile.Unlock()

	conn, err := dbus.SessionBus()
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}
	defer conn.Close()

	output, err := getOutputFrom(conn, idxFile, command)
	if err != nil {
		return "", fmt.Errorf("get output: %w", err)
	}
	return output, nil
}

func getOutputFrom(conn *dbus.Conn, idxFile *filelock.File, command string) (string, error) {
	players, err := getActivePlayers(conn)
	if err != nil {
		return "", fmt.Errorf("get players: %w", err)
	}
	if len(players) == 0 {
		return "", nil
	}

	idx := getInt(idxFile)
	switch command {
	case "next-player":
		idx = setInt(idxFile, positiveModulo(idx+1, len(players)))
	case "prev-player":
		idx = setInt(idxFile, positiveModulo(idx-1, len(players)))
	}

	player := players[idx]

	switch command {
	case "play-pause":
		prev := player.GetPlaybackStatus()
		player.PlayPause()
		waitUntil(func() bool {
			return player.GetPlaybackStatus() != prev
		})
	case "stop":
		prev := player.GetPlaybackStatus()
		player.Stop()
		waitUntil(func() bool {
			status := player.GetPlaybackStatus()
			return status != prev || status == mpris.PlaybackStopped
		})
		players, err = getActivePlayers(conn)
		if err != nil {
			return "", fmt.Errorf("get players: %w", err)
		}
		if len(players) == 0 {
			return "", nil
		}
		idx = setInt(idxFile, positiveModulo(idx, len(players)))
		player = players[idx]
	}

	var parts []string
	parts = append(parts, strings.ToLower(string(player.GetPlaybackStatus())))
	if len(players) > 1 {
		parts = append(parts, fmt.Sprintf("%d/%d", idx+1, len(players)))
	}

	metadata := player.GetMetadata()
	title := metadata["xesam:title"]
	parts = append(parts, fmt.Sprintf(`‘%s’`, trunc(title, 40, "…")))

	return fmt.Sprintf("%v", strings.Join(parts, " ")), nil
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

func setInt(w io.WriteSeeker, i int) int {
	w.Seek(0, 0)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))
	w.Write(b)
	return i
}

func getInt(r io.ReadSeeker) int {
	r.Seek(0, 0)
	b := make([]byte, 8)
	r.Read(b)
	return int(binary.LittleEndian.Uint64(b))
}

func trunc(in string, max int, ellip string) string {
	if len(in) <= max {
		return in
	}
	return in[:max] + ellip
}

func positiveModulo(a, b int) int {
	return ((a % b) + b) % b
}

func waitUntil(cb func() bool) {
	for i := 0; i < 20; i++ {
		if cb() {
			break
		}
		time.Sleep(statusChangePollInterval)
	}
}
