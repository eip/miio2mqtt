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

// go build -ldflags "-X main.version=`git describe --exact-match --tags 2> /dev/null || git rev-parse --short HEAD`" .
var version = "development"

var devices = make(miio.Devices)

func init() {
	log.Print("[DEBUG] main init()")
}

func initDevices() {
	// now := time.Now()
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
			// UpdatedAt:   now.Add(-time.Second),
			// PushedAt:    now,
		}
		token, _ := hex.DecodeString(dc.Token)
		copy(d.Token[:], token)
		d.SetStage(miio.Undiscovered)
		devices[id] = &d
	}
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
	fmt.Printf("miio2mqtt %s\n", version)
	if err := run(ctx); err != nil {
		log.Printf("[ERROR] %v", err)
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}
	log.Print("[DEBUG] miio2mqtt finished")
}

func run(ctx context.Context) error {
	startTime := time.Now()
	log.Printf("[DEBUG] starting [%v]...", startTime)
	wg := sync.WaitGroup{}
	defer wg.Wait()

	listener, err := net.StartListener(ctx, &wg)
	if err != nil {
		return err
	}
	defer listener.Stop()
	listener.Purge()

	messages := make(chan mqtt.Message, 100)
	go func() {
		client := mqtt.NewClient()
		for {
			select {
			case <-ctx.Done():
				log.Print("[DEBUG] stop processimg mqtt messages")
				return
			case msg := <-messages:
				if err := mqtt.Publish(client, msg); err != nil {
					log.Printf("[WARN] unable to publish to MQTT brocker: %v", err)
				}
			}
		}
	}()
	for {
		listener.Purge()
		now := time.Now()
		devices.SetStage(miio.Undiscovered, func(d *miio.Device) bool {
			if !miio.DeviceFound(d) {
				return false
			}
			outdated := now.After(d.UpdatedAt.Add(config.C.PollInterval * 3))
			if outdated {
				log.Printf("[INFO] outdated %s (updated %v ago)", d.Name, now.Sub(d.UpdatedAt))
			}
			return outdated
		})
		devices.SetStage(miio.Valid, miio.DeviceUpdated)
		err = net.QueryDevices(ctx, listener, devices, messages)
		if err != nil {
			log.Printf("[DEBUG] unable to update all devices: %v", err)
			// return err
		} else {
			log.Print("[DEBUG] all devices were updated successfully")
		}
		listener.Purge()
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(config.C.PollInterval):
		}
	}

	// devicesLeft, err := net.Discover(ctx, listener, devices)
	// if err != nil {
	// 	log.Printf("[DEBUG] %d devices were undiscovered", devicesLeft)
	// 	// return err
	// } else {
	// 	log.Print("[DEBUG] all devices were discovered successfully")
	// }

	// if devices.Count(miio.DeviceFound) == 0 {
	// 	log.Print("[DEBUG] no devices were discovered")
	// 	return nil
	// }

	// listener.Purge()
	// devicesLeft, err = net.UpdateInfo(ctx, listener, devices)
	// if err != nil {
	// 	log.Printf("[DEBUG] %d devices were not updated", devicesLeft)
	// 	// return err
	// } else {
	// 	log.Print("[DEBUG] all devices were updated successfully")
	// }

	// if devices.Count(miio.DeviceValid) == 0 {
	// 	log.Print("[DEBUG] no valid device exist")
	// 	return nil
	// }

	// <-ctx.Done()
	// return nil
	// startTime = time.Now()
	// next := startTime.Add(cfg.PollInterval).Truncate(cfg.PollInterval).Add(-cfg.PollAheadTime)
	// if next.Before(startTime) {
	// 	next = next.Add(cfg.PollInterval)
	// }
	// select {
	// case <-ctx.Done():
	// 	return nil
	// case <-time.After(next.Sub(startTime)):
	// }

	// ticker := time.NewTicker(time.Duration(cfg.PollInterval))
	// defer ticker.Stop()
	// startTime = time.Now()
	// log.Print("[DEBUG] polling begins")
	// wg := &sync.WaitGroup{}
	// process(ctx, wg)
	// for {
	// 	select {
	// 	case <-ticker.C:
	// 		process(ctx, wg)
	// 	case <-ctx.Done():
	// 		wg.Wait()
	//    return nil
	// 	}
	// }
}

// func process(ctx context.Context, wg *sync.WaitGroup) {
// 	ctxD, cancelD := context.WithTimeout(ctx, cfg.PollTimeout)
// 	defer cancelD()
// 	wg.Add(len(state.devices))
// 	for _, d := range state.devices {
// 		go func(d *deviceState) {
// 			defer wg.Done()
// 			pollDevice(ctxD, d)
// 		}(d)
// 	}
// 	wg.Wait()
// 	ctxQ, cancelQ := context.WithTimeout(ctx, cfg.PushTimeout)
// 	defer cancelQ()
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		pushToQuery(ctxQ)
// 	}()
// 	wg.Wait()
// }

// func pollDevice(ctx context.Context, ds *deviceState) {
// 	log.Printf("[DEBUG] polling device %s...", ds.name)
// 	select {
// 	case <-ctx.Done():
// 		log.Printf("[DEBUG] polling device %s timed out", ds.name)
// 	case <-time.After(time.Millisecond * time.Duration(500+rand.Intn(1000))):
// 		newValue := fmt.Sprintf(`{"power":%d}`, rand.Intn(2))
// 		if newValue != ds.value {
// 			ds.value = newValue
// 			ds.updatedAt = time.Now()
// 			log.Printf("[INFO] device %s state: %s", ds.name, ds.value)
// 		}
// 		log.Printf("[DEBUG] polling device %s done", ds.name)
// 	}
// }

// func pushToQuery(ctx context.Context) {
// 	log.Print("[DEBUG] pushing results...")
// 	// cd, _ := ctx.Deadline()
// 	// log.Printf("[DEBUG] context for query: %v", cd)
// 	select {
// 	case <-ctx.Done():
// 		log.Print("[DEBUG] pushing results timed out")
// 	case <-time.After(time.Millisecond * time.Duration(500+rand.Intn(1000))):
// 		for _, ds := range state.devices {
// 			if state.pushedAt.Before(ds.updatedAt) {
// 				log.Printf("[INFO] pushed %s to %s", ds.value, ds.properties.Topic)
// 			}
// 		}
// 		state.pushedAt = time.Now()
// 	}
// }

// // func devicesToComplete(deviceStates deviceStates) miio.Devices {
// // 	devices := miio.Devices{}
// // 	for _, ds := range deviceStates {
// // 		if ds.properties.TimeShift == 0 || ds.properties.ID == 0 || len(ds.properties.Address) == 0 {
// // 			devices = append(devices, &ds.properties)
// // 		}
// // 	}
// // 	return devices
// // }

// func (s appState) getDevices() miio.Devices {
// 	devices := make(miio.Devices, len(s.devices))
// 	for i, ds := range s.devices {
// 		devices[i] = &ds.properties
// 	}
// 	return devices
// }

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		// mqttDebugLog = log.New(log.Debug, log.CallerDepth(1), log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
