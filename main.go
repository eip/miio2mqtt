package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/eip/miio2mqtt/config"
	"github.com/eip/miio2mqtt/miio"
	"github.com/eip/miio2mqtt/mqtt"
	"github.com/eip/miio2mqtt/net"
	log "github.com/go-pkgz/lgr"
)

// go build -ldflags "-s -w -X main.version=`git describe --exact-match --tags 2> /dev/null || git rev-parse --short HEAD`" .
var version = "development"

var devices = make(miio.Devices)

func init() {
	log.Print("[DEBUG] main init()")
}

func main() {
	setupLog(false)
	if err := config.Load("./config.yml"); err != nil {
		log.Printf("[ERROR] configuration: %v", err)
		return
	}
	if config.C.Debug {
		setupLog(true)
	}
	initDevices()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigChan
		fmt.Print("\r")
		log.Printf("[WARN] %v signal received", s)
		cancel()
	}()
	fmt.Printf("miio2mqtt version %s\n", version)
	if err := run(ctx); err != nil {
		log.Printf("[ERROR] %v", err)
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}
	log.Print("[DEBUG] miio2mqtt finished")
}

func run(ctx context.Context) error {
	wg := sync.WaitGroup{}
	defer wg.Wait()

	client := mqtt.NewClient()
	defer client.Disconnect()
	messages := make(chan mqtt.Message, 100)
	wg.Add(1)
	go func() { defer wg.Done(); processMessages(ctx, client, messages) }()

	for {
		next := nextTime(time.Now())
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(next.Sub(time.Now())):
		}
		now := time.Now()
		devices.SetStage(miio.Undiscovered, func(d *miio.Device) bool {
			if !miio.DeviceFound(d) {
				return false
			}
			outdated := now.After(d.UpdatedAt.Add(config.C.PollInterval * 2))
			if outdated {
				log.Printf("[INFO] outdated %s (updated %v ago)", d.Name, now.Sub(d.UpdatedAt))
			}
			return outdated
		})
		devices.SetStage(miio.Valid, miio.DeviceUpdated)
		listener, err := net.StartListener(ctx, &wg)
		if err != nil {
			log.Printf("[WARN] unable to listen for UDP packets: %v", err)
			continue
		}
		err = net.QueryDevices(ctx, listener, devices, messages)
		listener.Stop()
		if err != nil {
			log.Printf("[WARN] unable to update all devices: %v", err)
			// return err
		} else {
			log.Print("[DEBUG] all devices were updated successfully")
		}
	}
}

func initDevices() {
	idx := 0
	for n, dc := range config.C.Devices {
		idx++
		var id uint32
		if len(dc.Address) > 0 {
			id, _ = net.IPv4StrToInt(dc.Address)
		} else if dc.ID > 0 {
			id = dc.ID
		}
		if id == 0 {
			log.Printf("[WARN] invalid device configuration: %s", n)
			continue
		}
		if d, exists := devices[id]; exists {
			log.Printf("[WARN] duplicate device: %s (%08x) >>> %s", n, id, d.Name)
			continue
		}
		d := miio.Device{
			DeviceCfg: dc,
			Name:      n,
		}
		token, _ := hex.DecodeString(dc.Token)
		copy(d.Token[:], token)
		d.SetStage(miio.Undiscovered)
		devices[id] = &d
	}
}

func processMessages(ctx context.Context, client *mqtt.Client, messages <-chan mqtt.Message) {
	for {
		select {
		case <-ctx.Done():
			log.Print("[DEBUG] stop processing mqtt messages")
			return
		case msg := <-messages:
			if err := client.Publish(msg); err != nil {
				log.Printf("[WARN] unable to publish to MQTT broker: %v", err)
			}
		}
	}
}

func nextTime(now time.Time) time.Time {
	result := now.Add(config.C.PollInterval).Truncate(config.C.PollInterval).Add(-config.C.PollAheadTime)
	if result.Before(now) {
		return result.Add(config.C.PollInterval)
	}
	return result
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		// mqttDebugLog = log.New(log.Debug, log.CallerDepth(1), log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
