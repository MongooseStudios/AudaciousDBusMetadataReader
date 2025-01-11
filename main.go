package main

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

const outputFileName = "trackinfo.txt"
const pollTime = 1 * time.Second
const artistKey = "xesam:artist"
const titleKey = "xesam:title"

type trackInfo struct {
	artist string
	title  string
}

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalf("error connecting session bus: %v", err)
	}

	defer func() {
		err = conn.Close()
		if err != nil {
			log.Fatalf("error disconnecting session bus: %v", err)
		}
	}()

	osSig := make(chan os.Signal)
	currentTrackInfo := trackInfo{}

	log.Print("now monitoring dbus for track metadata")

operationLoop:
	for {

		select {
		case sig := <-osSig:
			if sig == syscall.SIGINT || sig == syscall.SIGKILL {
				break operationLoop
			}
		default:
		}

		metadata, err := getMetadataFromDBus(conn)
		if err != nil {
			log.Fatalf("error getting metadata from DBus: %v", err)
		}
		info, err := convertDBusOutput(metadata)
		if err != nil {
			log.Fatalf("error parsing metadata: %v", err)
		}

		if info.title != currentTrackInfo.title || info.artist != currentTrackInfo.artist {
			err = writeData(info)
			if err != nil {
				log.Fatalf("error writing output file: %v", err)
			}

			currentTrackInfo = info
		}

		time.Sleep(pollTime)
	}
}

func getMetadataFromDBus(conn *dbus.Conn) (map[string]dbus.Variant, error) {
	bus := conn.Object("org.mpris.MediaPlayer2.audacious", "/org/mpris/MediaPlayer2")
	result, err := bus.GetProperty("org.mpris.MediaPlayer2.Player.Metadata")
	if err != nil {
		return nil, fmt.Errorf("error getting property from DBus: %v", err)
	}

	return result.Value().(map[string]dbus.Variant), nil
}

func convertDBusOutput(input map[string]dbus.Variant) (trackInfo, error) {
	var converted trackInfo
	if artist, ok := input[artistKey]; ok {
		artistSlice := artist.Value().([]string)
		converted.artist = strings.Join(artistSlice, ", ")
	}
	if title, ok := input[titleKey]; ok {
		converted.title = title.Value().(string)
	}

	return converted, nil
}

func writeData(info trackInfo) error {
	output := []byte(fmt.Sprintf("< Artist: %s | Track: %s >    ", info.artist, info.title))
	err := os.WriteFile(outputFileName, output, 0666)
	if err != nil {
		return err
	}
	return nil
}
