// The MIT License (MIT)
//
// Copyright (c) 2015 emersion
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package mpris

import (
	"fmt"
	"log"
	"strings"

	"github.com/godbus/dbus"
)

const (
	dbusObjectPath          = "/org/mpris/MediaPlayer2"
	propertiesChangedSignal = "org.freedesktop.DBus.Properties.PropertiesChanged"

	baseInterface      = "org.mpris.MediaPlayer2"
	playerInterface    = "org.mpris.MediaPlayer2.Player"
	trackListInterface = "org.mpris.MediaPlayer2.TrackList"
	playlistsInterface = "org.mpris.MediaPlayer2.Playlists"

	getPropertyMethod = "org.freedesktop.DBus.Properties.Get"
)

func getProperty(obj *dbus.Object, iface string, prop string) dbus.Variant {
	result := dbus.Variant{}
	err := obj.Call(getPropertyMethod, 0, iface, prop).Store(&result)
	if err != nil {
		log.Println("Warning: could not get dbus property:", iface, prop, err)
		return dbus.Variant{}
	}
	return result
}

func List(conn *dbus.Conn) ([]string, error) {
	var names []string
	err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return nil, err
	}

	var mprisNames []string
	for _, name := range names {
		if strings.HasPrefix(name, baseInterface) {
			mprisNames = append(mprisNames, name)
		}
	}
	return mprisNames, nil
}

type Player struct {
	*base
	*player
}

type base struct {
	obj *dbus.Object
}

func (i *base) Raise() {
	i.obj.Call(baseInterface+".Raise", 0)
}

func (i *base) Quit() {
	i.obj.Call(baseInterface+".Quit", 0)
}

func (i *base) GetIdentity() string {
	return getProperty(i.obj, baseInterface, "Identity").String()
}

type player struct {
	obj *dbus.Object
}

func (i *player) Next() {
	i.obj.Call(playerInterface+".Next", 0)
}

func (i *player) Previous() {
	i.obj.Call(playerInterface+".Previous", 0)
}

func (i *player) Pause() {
	i.obj.Call(playerInterface+".Pause", 0)
}

func (i *player) PlayPause() {
	i.obj.Call(playerInterface+".PlayPause", 0)
}

func (i *player) Stop() {
	i.obj.Call(playerInterface+".Stop", 0)
}

func (i *player) Play() {
	i.obj.Call(playerInterface+".Play", 0)
}

func (i *player) OpenURI(uri string) {
	i.obj.Call(playerInterface+".OpenUri", 0, uri)
}

type PlaybackStatus string

const (
	PlaybackPlaying PlaybackStatus = "Playing"
	PlaybackPaused                 = "Paused"
	PlaybackStopped                = "Stopped"
)

func (i *player) GetPlaybackStatus() PlaybackStatus {
	variant, err := i.obj.GetProperty(playerInterface + ".PlaybackStatus")
	if err != nil {
		return ""
	}
	return PlaybackStatus(fmt.Sprint(variant.Value()))
}

type LoopStatus string

const (
	LoopNone     LoopStatus = "None"
	LoopTrack               = "Track"
	LoopPlaylist            = "Playlist"
)

func (i *player) GetLoopStatus() LoopStatus {
	return LoopStatus(getProperty(i.obj, playerInterface, "LoopStatus").String())
}

func (i *player) GetRate() float64 {
	return getProperty(i.obj, playerInterface, "Rate").Value().(float64)
}

func (i *player) GetShuffle() bool {
	return getProperty(i.obj, playerInterface, "Shuffle").Value().(bool)
}

func (i *player) GetMetadata() map[string]string {
	variants := getProperty(i.obj, playerInterface, "Metadata").Value().(map[string]dbus.Variant)
	out := map[string]string{}
	for k, v := range variants {
		out[k] = fmt.Sprint(v.Value())
	}
	return out
}

func (i *player) GetVolume() float64 {
	return getProperty(i.obj, playerInterface, "Volume").Value().(float64)
}

func (i *player) GetPosition() int64 {
	return getProperty(i.obj, playerInterface, "Position").Value().(int64)
}

func New(conn *dbus.Conn, name string) *Player {
	obj := conn.Object(name, dbusObjectPath).(*dbus.Object)

	return &Player{
		&base{obj},
		&player{obj},
	}
}
