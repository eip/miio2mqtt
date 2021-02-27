package main

import (
	"context"
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
	config := config.New()
	if err := config.Load("./config.yml"); err != nil {
		log.Printf("[ERROR] configuration: %v", err)
		return
	}
	if config.Debug {
		setupLog(true)
	}
	initDevices(config)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigChan
		fmt.Print("\r")
		log.Printf("[WARN] %v signal received", s)
		cancel()
	}()
	fmt.Printf("miio2mqtt version %s", version)
	if err := run(ctx, config); err != nil {
		log.Printf("[ERROR] %v", err)
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}
	log.Print("[DEBUG] miio2mqtt finished")
}

func run(ctx context.Context, config *config.Config) error {
	firstLoop := true
	wg := sync.WaitGroup{}
	defer wg.Wait()

	transport := net.NewTransport(config)
	poller := net.NewPoller(config, transport, devices)
	broker := mqtt.NewClient(config)
	defer broker.Disconnect()
	wg.Add(1)
	go func() { defer wg.Done(); publishUpdates(ctx, broker, poller.Updates()) }()

	deviceUpdateTimeout := 2 * miio.TimeStamp(config.PollInterval/time.Second)
	for {
		next := nextTime(time.Now(), config.PollInterval, config.PollAheadTime)
		if firstLoop {
			startIn := next.Sub(time.Now()) / 100 / time.Millisecond * 100 * time.Millisecond
			if startIn > 1550*time.Millisecond {
				fmt.Printf(" - starting in %v", startIn)
			}
			fmt.Println("")
			firstLoop = false
		}
		select {
		case <-ctx.Done():
			log.Printf("[INFO] max queue lengths: packets = %d, updates = %d", config.ChanStat[0], config.ChanStat[1])
			return nil
		case <-time.After(next.Sub(time.Now())):
		}
		devices.SetStage(miio.Undiscovered, miio.DeviceOutdated(deviceUpdateTimeout))
		devices.SetStage(miio.Valid, miio.DeviceUpdated)
		err := transport.Start(ctx, &wg)
		if err != nil {
			log.Printf("[WARN] unable to listen for UDP packets: %v", err)
			continue
		}
		err = poller.PollDevices(ctx)
		transport.Stop()
		if err != nil {
			log.Printf("[WARN] unable to update all devices: %v", err)
			// return err
		} else {
			log.Print("[DEBUG] all devices were updated successfully")
		}
	}
}

func initDevices(config *config.Config) {
	idx := 0
	for n, dc := range config.Devices {
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
		devices[id] = miio.NewDevice(dc, n)
	}
}

func publishUpdates(ctx context.Context, client *mqtt.Client, updates <-chan *miio.Device) {
	// defer client.Disconnect()
	for {
		select {
		case <-ctx.Done():
			log.Print("[DEBUG] stop processing mqtt messages")
			return
		case device := <-updates:
			if err := client.Publish(device); err != nil {
				log.Printf("[WARN] unable to publish to MQTT broker: %v", err)
			}
		}
	}
}

func nextTime(now time.Time, interval time.Duration, aheadTime time.Duration) time.Time {
	result := now.Add(interval).Truncate(interval).Add(-aheadTime)
	if result.Before(now) {
		return result.Add(interval)
	}
	return result
}

func setupLog(dbg bool) {
	stripDate := log.Mapper{TimeFunc: func(s string) string { return s[11:] }}
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces, log.Map(stripDate))
		return
	}
	log.Setup(log.Msec, log.LevelBraces, log.Map(stripDate))
}
