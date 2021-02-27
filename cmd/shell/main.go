package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	prompt "github.com/c-bata/go-prompt"
	"github.com/eip/miio2mqtt/config"
	"github.com/eip/miio2mqtt/miio"
	"github.com/eip/miio2mqtt/net"
)

var suggestions = []prompt.Suggest{
	// Commands
	{Text: "quit", Description: "quit shell"},
	{Text: "info", Description: "show device info"},
	// Config
	{Text: "id", Description: "<id> set device id"},
	{Text: "ip", Description: "<address> set device ip address"},
	{Text: "token", Description: "<token> set device token"},
}

type application struct {
	deviceCfg *miio.DeviceCfg
	device    *miio.Device
	devices   miio.Devices
	transport *net.UDPTransport
	poller    *net.Poller
}

var app = &application{
	deviceCfg: &miio.DeviceCfg{},
}

func identifyDevice(dc *miio.DeviceCfg) error {
	var id uint32
	if len(dc.Address) > 0 {
		id, _ = net.IPv4StrToInt(dc.Address)
	} else if dc.ID > 0 {
		id = dc.ID
	}
	if id == 0 {
		return errors.New("Invalid device configuration")
	}

	app.device = miio.NewDevice(*dc, "Device")
	app.device.SetFinalStage(miio.Valid)
	app.devices[id] = app.device

	wg := sync.WaitGroup{}
	defer wg.Wait()
	ctx := context.Background()
	err := app.transport.Start(ctx, &wg)
	if err != nil {
		return fmt.Errorf("Unable to listen for UDP packets: %v", err)
	}
	err = app.poller.PollDevices(ctx)
	app.transport.Stop()
	if err != nil {
		return fmt.Errorf("Unable to identify device: %v", err)
	}
	return nil
}

func printDeviceInfo(d *miio.Device) {
	if d.ID == 0 && len(d.Address) == 0 {
		fmt.Println("Uninitialized device")
		return
	}
	div := " "
	fmt.Print("Device")
	if d.ID > 0 {
		fmt.Printf("%sID: %08x", div, d.ID)
		div = ", "
	}
	if len(d.Address) > 0 {
		fmt.Printf("%sAddress: %s", div, d.Address)
		div = ", "
	}
	if len(d.Model()) > 0 {
		fmt.Printf("%sModel: %s", div, d.Model())
	}
	fmt.Printf("%sToken: %x\n", div, d.Token())
}

func setDeviceID(d *miio.DeviceCfg, val string) error {
	if id, _ := strconv.ParseUint(val, 0, 32); id != 0 {
		d.ID = uint32(id)
		return nil
	}
	return fmt.Errorf("Invalid device ID: %q", val)
}

func setDeviceAddress(d *miio.DeviceCfg, val string) error {
	if _, err := net.IPv4StrToInt(val); err != nil {
		return fmt.Errorf("Invalid IP address: %q", val)
	}
	d.Address = val
	return nil
}

func setDeviceToken(d *miio.DeviceCfg, val string) error {
	if token, _ := hex.DecodeString(val); len(token) == 16 {
		d.Token = hex.EncodeToString(token)
		return nil
	}
	return fmt.Errorf("Invalid token: %q", val)
}

func livePrefix() (string, bool) {
	if app.device != nil {
		if app.device.ID > 0 {
			return fmt.Sprintf("%08x> ", app.device.ID), true
		}
		if len(app.device.Address) > 0 {
			return fmt.Sprintf("%s> ", app.device.Address), true
		}
	}
	if app.deviceCfg.ID > 0 {
		return fmt.Sprintf("%08x> ", app.deviceCfg.ID), true
	}
	if len(app.deviceCfg.Address) > 0 {
		return fmt.Sprintf("%s> ", app.deviceCfg.Address), true
	}
	return "", false
}

func executor(in string) {
	in = strings.TrimSpace(in)
	blocks := strings.Split(in, " ")
	blocks = append(blocks, "", "", "")
	switch blocks[0] {
	case "q", "quit":
		colorPrintf(prompt.Green, "Bye!\n")
		os.Exit(0)
	case "info":
		var err error
		if app.device != nil && len(app.device.Model()) > 0 {
			printDeviceInfo(app.device)
			break
		}
		if err = identifyDevice(app.deviceCfg); err != nil {
			colorPrintf(prompt.Brown, "%v\n", err)
			break
		}
		printDeviceInfo(app.device)
	case "id":
		if err := setDeviceID(app.deviceCfg, blocks[1]); err != nil {
			colorPrintf(prompt.Brown, "%v\n", err)
		}
	case "ip":
		if err := setDeviceAddress(app.deviceCfg, blocks[1]); err != nil {
			colorPrintf(prompt.Brown, "%v\n", err)
		}
	case "token":
		if err := setDeviceToken(app.deviceCfg, blocks[1]); err != nil {
			colorPrintf(prompt.Brown, "%v\n", err)
		}
	default:
		colorPrintf(prompt.Brown, "Unknown command: %s\n", blocks[0])
	}
}

func completer(in prompt.Document) []prompt.Suggest {
	w := in.GetWordBeforeCursor()
	if w == "" {
		return []prompt.Suggest{}
	}
	return prompt.FilterHasPrefix(suggestions, w, true)
}

func initPrompt() *prompt.Prompt {
	return prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionLivePrefix(livePrefix),
		prompt.OptionTitle("miio-shell"),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionSuggestionTextColor(prompt.White),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionDescriptionTextColor(prompt.LightGray),
		prompt.OptionDescriptionBGColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionTextColor(prompt.White),
		prompt.OptionSelectedSuggestionBGColor(prompt.DarkBlue),
		prompt.OptionSelectedDescriptionTextColor(prompt.LightGray),
		prompt.OptionSelectedDescriptionBGColor(prompt.DarkBlue),
		prompt.OptionPreviewSuggestionTextColor(prompt.White),
		prompt.OptionScrollbarBGColor(prompt.DarkGray),
		prompt.OptionScrollbarThumbColor(prompt.LightGray),
	)
}

func main() {
	config := config.New()
	setupLog()
	app.devices = make(miio.Devices)
	app.transport = net.NewTransport(config)
	app.poller = net.NewPoller(config, app.transport, app.devices)

	for i := 1; i < len(os.Args) && i < 4; i++ {
		arg := os.Args[i]
		if err := setDeviceID(app.deviceCfg, arg); err == nil {
			continue
		}
		if err := setDeviceAddress(app.deviceCfg, arg); err == nil {
			continue
		}
		if err := setDeviceToken(app.deviceCfg, arg); err == nil {
			continue
		}
	}
	initPrompt().Run()
}
