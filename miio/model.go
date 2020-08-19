package miio

import (
	"encoding/json"
	"fmt"
	"regexp"

	log "github.com/go-pkgz/lgr"
)

// Device represents a miIO all device properties
type Model struct {
	Methods ModelMethods `yaml:"Methods"`
	Params  []string     `yaml:"Params"`
}

type ModelMethods struct {
	MiioInfo string `yaml:"MiioInfo"`
	GetProp  string `yaml:"GetProp"`
}

type Models map[string]Model

type ReplyType int32

const (
	Unrecognized ReplyType = iota
	MiioInfo
	GetProp
)

type Reply struct {
	Type  ReplyType
	Model string
	Props []interface{}
}

var reParams = regexp.MustCompile(`("params":\s?)#+`)

func (mm Models) MiioInfo(model string) string {
	for _, name := range []string{model, "*"} {
		if m, ok := mm[name]; ok {
			if len(m.Methods.MiioInfo) > 0 {
				return m.Methods.MiioInfo
			}
		}
	}
	log.Printf("[WARN] unable to find %s miIO.info request", model)
	return ""
}

func (mm Models) Params(model string) []string {
	for _, name := range []string{model, "*"} {
		if m, ok := mm[name]; ok {
			if len(m.Params) > 0 {
				return m.Params
			}
		}
	}
	log.Printf("[WARN] unable to find %s parameters list", model)
	return nil
}

func (mm Models) GetProp(model string) string {
	var request []byte
	for _, name := range []string{model, "*"} {
		if m, ok := mm[name]; ok {
			if len(m.Methods.GetProp) > 0 {
				request = []byte(m.Methods.GetProp)
				break
			}
		}
	}
	if len(request) == 0 {
		log.Printf("[WARN] unable to find %s get_prop request", model)
		return ""
	}
	params := mm.Params(model)
	if len(params) == 0 {
		return ""
	}
	paramsStr, err := json.Marshal(params)
	if err != nil {
		log.Printf("[WARN] invalid %s request parameters %v: %s", model, params, err)
		return ""
	}
	return string(reParams.ReplaceAll(request, []byte(fmt.Sprintf("${1}%s", paramsStr))))
}

type deviceInfoReply struct {
	ID     int `json:"id"`
	Result struct {
		Model string `json:"model"`
	} `json:"result"`
}

type devicePropReply struct {
	ID     int           `json:"id"`
	Result []interface{} `json:"result"`
}

func ParseReply(data []byte) Reply {
	result := Reply{Type: Unrecognized}
	info := deviceInfoReply{}
	if err := json.Unmarshal(data, &info); err == nil && info.Result.Model != "" {
		result.Model = info.Result.Model
		result.Type = MiioInfo
		return result
	}
	props := devicePropReply{}
	if err := json.Unmarshal(data, &props); err == nil && len(props.Result) > 0 {
		result.Props = props.Result
		result.Type = GetProp
		return result
	}
	log.Printf("[WARN] unable to parse response: %s", data)
	return result
}
