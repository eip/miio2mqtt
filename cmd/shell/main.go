package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	prompt "github.com/c-bata/go-prompt"
	"github.com/eip/miio2mqtt/miio"
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

type context struct {
	device *miio.Device
}

var ctx = &context{
	device: &miio.Device{},
}

var console = prompt.NewStdoutWriter()

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
	if len(d.Model) > 0 {
		fmt.Printf("%sAddress: %s", div, d.Model)
	}
	fmt.Printf("%sToken: %x\n", div, d.Token)
}

func setDeviceID(d *miio.Device, val string) error {
	if id, _ := strconv.ParseUint(val, 0, 32); id != 0 {
		d.ID = uint32(id)
		return nil
	}
	return fmt.Errorf("Invalid device ID: %q", val)
}

func setDeviceAddress(d *miio.Device, val string) error {
	if ip := net.ParseIP(val); ip != nil {
		d.Address = ip.String()
		return nil
	}
	return fmt.Errorf("Invalid IP address: %q", val)
}

func setDeviceToken(d *miio.Device, val string) error {
	if token, _ := hex.DecodeString(val); len(token) == 16 {
		copy(d.Token[:], token)
		return nil
	}
	return fmt.Errorf("Invalid token: %q", val)
}

func livePrefix() (string, bool) {
	if ctx.device.ID > 0 {
		return fmt.Sprintf("%08x> ", ctx.device.ID), true
	}
	if len(ctx.device.Address) > 0 {
		return fmt.Sprintf("%s> ", ctx.device.Address), true
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
		printDeviceInfo(ctx.device)
	case "id":
		if err := setDeviceID(ctx.device, blocks[1]); err != nil {
			colorPrintf(prompt.Brown, "%v\n", err)
		}
	case "ip":
		if err := setDeviceAddress(ctx.device, blocks[1]); err != nil {
			colorPrintf(prompt.Brown, "%v\n", err)
		}
	case "token":
		if err := setDeviceToken(ctx.device, blocks[1]); err != nil {
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

func colorPrintf(fg prompt.Color, format string, a ...interface{}) {
	console.SetColor(fg, prompt.DefaultColor, false)
	console.WriteStr(fmt.Sprintf(format, a...))
	console.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	console.Flush()
}

func main() {
	for i := 1; i < len(os.Args) && i < 4; i++ {
		arg := os.Args[i]
		if err := setDeviceID(ctx.device, arg); err == nil {
			continue
		}
		if err := setDeviceAddress(ctx.device, arg); err == nil {
			continue
		}
		if err := setDeviceToken(ctx.device, arg); err == nil {
			continue
		}
	}
	p := prompt.New(
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
	p.Run()
}
