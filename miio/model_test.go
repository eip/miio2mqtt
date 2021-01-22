package miio

import (
	"regexp"
	"testing"

	h "github.com/eip/miio2mqtt/helpers"
)

func Test_DefaultModel(t *testing.T) {
	want := Model{
		Methods: ModelMethods{MiioInfo: defaultMiioInfoRequest, GetProp: defaultGetPropRequest},
	}
	got := DefaultModel()
	h.AssertEqual(t, got, want)
}

func TestModels_MiioInfo(t *testing.T) {
	logRe := regexp.MustCompile(`^\[WARN\]\s+unable to find`)
	tests := []struct {
		name   string
		models Models
		model  string
		want   string
		logRe  *regexp.Regexp
	}{
		{
			name:   "Empty Models",
			models: Models{},
			model:  "dummy.test.v1",
			want:   "",
			logRe:  logRe,
		},
		{
			name: "Nonexisting Model",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{MiioInfo: `{"method":"foo","id":#}`}},
			},
			model: "dummy.test.v2",
			want:  defaultMiioInfoRequest,
		},
		{
			name: "Model with undefined MiioInfo method",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{}},
			},
			model: "dummy.test.v1",
			want:  defaultMiioInfoRequest,
		},
		{
			name: "Model with empty MiioInfo method",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{MiioInfo: ""}},
			},
			model: "dummy.test.v1",
			want:  defaultMiioInfoRequest,
		},
		{
			name: "Model with defined MiioInfo method",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{MiioInfo: `{"method":"foo","id":#}`}},
			},
			model: "dummy.test.v1",
			want:  `{"method":"foo","id":#}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLog.Reset()
			got := tt.models.MiioInfo(tt.model)
			h.AssertEqual(t, got, tt.want)
			h.AssertEqual(t, testLog.Message, tt.logRe)
		})
	}
}

func TestModels_Params(t *testing.T) {
	logRe := regexp.MustCompile(`^\[WARN\]\s+unable to find`)
	tests := []struct {
		name   string
		models Models
		model  string
		want   []string
		logRe  *regexp.Regexp
	}{
		{
			name:   "Empty Models",
			models: Models{},
			model:  "dummy.test.v1",
			want:   nil,
			logRe:  logRe,
		},
		{
			name: "Nonexisting Model",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Params: []string{"foo", "bar", "baz"}},
			},
			model: "dummy.test.v2",
			want:  nil,
			logRe: logRe,
		},
		{
			name: "Model with undefined Params",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{},
			},
			model: "dummy.test.v1",
			want:  nil,
			logRe: logRe,
		},
		{
			name: "Model with empty Params",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Params: []string{}},
			},
			model: "dummy.test.v1",
			want:  nil,
			logRe: logRe,
		},
		{
			name: "Model with empty strings in Params",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Params: []string{"", "", ""}},
			},
			model: "dummy.test.v1",
			want:  nil,
			logRe: logRe,
		},
		{
			name: "Model with defined Params",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Params: []string{"foo", "bar", "baz"}},
			},
			model: "dummy.test.v1",
			want:  []string{"foo", "bar", "baz"},
		},
		{
			name: "Model with defined Params containing empty strings",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Params: []string{"", "foo", "bar", "", "baz", "", ""}},
			},
			model: "dummy.test.v1",
			want:  []string{"foo", "bar", "baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLog.Reset()
			got := tt.models.Params(tt.model)
			h.AssertEqual(t, got, tt.want)
			h.AssertEqual(t, testLog.Message, tt.logRe)
		})
	}
}

func TestModels_GetProp(t *testing.T) {
	logRe := regexp.MustCompile(`^\[WARN\]\s+unable to find`)
	tests := []struct {
		name   string
		models Models
		model  string
		want   string
		logRe  *regexp.Regexp
	}{
		{
			name:   "Empty Models",
			models: Models{},
			model:  "dummy.test.v1",
			want:   "",
			logRe:  logRe,
		},
		{
			name: "Nonexisting Model",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{GetProp: `{"method":"foo","id":#}`}},
			},
			model: "dummy.test.v2",
			want:  "",
			logRe: logRe,
		},
		{
			name: "Model with undefined GetProp method",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{}},
			},
			model: "dummy.test.v1",
			want:  "",
			logRe: logRe,
		},
		{
			name: "Model with undefined GetProp method and defined Params",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{}, Params: []string{"foo", "bar", "baz"}},
			},
			model: "dummy.test.v1",
			want:  `{"method":"get_prop","params":["foo","bar","baz"],"id":#}`,
		},
		{
			name: "Model with empty GetProp method",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{GetProp: ""}},
			},
			model: "dummy.test.v1",
			want:  "",
			logRe: logRe,
		},
		{
			name: "Model with defined GetProp method",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{GetProp: `{"method":"bar","params":#,"id":#}`}},
			},
			model: "dummy.test.v1",
			want:  "",
			logRe: logRe,
		},
		{
			name: "Model with defined GetProp method and Params",
			models: Models{
				"*":             DefaultModel(),
				"dummy.test.v1": Model{Methods: ModelMethods{GetProp: `{"method":"bar","params":#,"id":#}`}, Params: []string{"foo", "bar", "baz"}},
			},
			model: "dummy.test.v1",
			want:  `{"method":"bar","params":["foo","bar","baz"],"id":#}`,
		},
		{
			name: "Model with defined GetProp method and empty Params",
			models: Models{
				"dummy.test.v1": Model{Methods: ModelMethods{GetProp: `{"method":"bar","params":#,"id":#}`}, Params: []string{}},
			},
			model: "dummy.test.v1",
			want:  "",
			logRe: logRe,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLog.Reset()
			got := tt.models.GetProp(tt.model)
			h.AssertEqual(t, got, tt.want)
			h.AssertEqual(t, testLog.Message, tt.logRe)
		})
	}
}

func Test_ParseReply(t *testing.T) {
	logRe := regexp.MustCompile(`^\[WARN\]\s+unable to parse`)
	tests := []struct {
		name  string
		data  string
		want  Reply
		logRe *regexp.Regexp
	}{
		{
			name:  "Invalid JSON",
			data:  "foo",
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name:  "Invalid MiioInfo reply 1",
			data:  `{"result":{"life":123456,"cfg_time":0},"id":1}`,
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name:  "Invalid MiioInfo reply 2",
			data:  `{"result":{"life":123456,"cfg_time":0,"model":123.45},"id":1}`,
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name:  "Invalid MiioInfo reply 2",
			data:  `{"result":{"life":123456,"cfg_time":0,"model":""},"id":1}`,
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name: "MiioInfo reply",
			data: `{"result":{"life":123456,"cfg_time":0,"model":"dummy.test.v1"},"id":1}`,
			want: Reply{Type: MiioInfo, Model: "dummy.test.v1"},
		},
		{
			name:  "Invalid GetProp reply 1",
			data:  `{"foo":["foo","bar",123.45,true],"id":1}`,
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name:  "Invalid GetProp reply 2",
			data:  `{"result":123.45,"id":1}`,
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name:  "Invalid GetProp reply 3",
			data:  `{"result":[],"id":1}`,
			want:  Reply{Type: Unrecognized},
			logRe: logRe,
		},
		{
			name: "GetProp reply",
			data: `{"result":["foo","bar",123.45,true],"id":1}`,
			want: Reply{Type: GetProp, Props: []interface{}{"foo", "bar", 123.45, true}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLog.Reset()
			got := ParseReply([]byte(tt.data))
			h.AssertEqual(t, got, tt.want)
			h.AssertEqual(t, testLog.Message, tt.logRe)
		})
	}
}
