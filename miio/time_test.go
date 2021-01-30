package miio

import (
	"testing"
	"time"

	h "github.com/eip/miio2mqtt/helpers"
)

const (
	sec  TimeStamp = 1
	min            = 60 * sec
	hour           = 60 * min
)

var sampleTS TimeStamp = 111*hour + 22*min + 33*sec // unix timestamp = 0x00061e39 / 1970-01-05 15:22:33

func Test_Stamp(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want TimeStamp
	}{
		{
			name: "Zero time stamp",
			time: time.Date(1970, 1, 1, 0, 0, 0, 1e9-1, time.UTC),
			want: 0,
		},
		{
			name: "1s time stamp",
			time: time.Date(1970, 1, 1, 0, 0, 0, 1e9, time.UTC),
			want: 1,
		},
		{
			name: "Sample time stamp",
			time: time.Unix(int64(sampleTS), 0),
			want: sampleTS,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Stamp(tt.time)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_Now(t *testing.T) {
	want := TimeStamp(time.Now().Unix())
	got := Now()
	if TimeStampDiff(got, want) > 1*sec {
		h.AssertEqual(t, got, want)
	}
}

func TestTimeStamp_String(t *testing.T) {
	tests := []struct {
		name string
		ts   TimeStamp
		want string
	}{
		{
			name: "Zero time stamp",
			ts:   0,
			want: "0s",
		},
		{
			name: "1s time stamp",
			ts:   1,
			want: "1s",
		},
		{
			name: "Sample time stamp",
			ts:   sampleTS,
			want: "111h22m33s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.String()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestTimeStamp_Time(t *testing.T) {
	tests := []struct {
		name string
		ts   TimeStamp
		want time.Time
	}{
		{
			name: "Zero time stamp",
			ts:   0,
			want: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "1s time stamp",
			ts:   1,
			want: time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC),
		},
		{
			name: "Sample time stamp",
			ts:   sampleTS,
			want: time.Date(1970, 1, 5, 15, 22, 33, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.Time()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_TimeStampDiff(t *testing.T) {
	tests := []struct {
		name string
		t1   TimeStamp
		t2   TimeStamp
		want TimeStamp
	}{
		{
			name: "Same time",
			t1:   1609495835,
			t2:   1609495835,
			want: 0,
		},
		{
			name: "First time before secont",
			t1:   1609495824,
			t2:   1609495835,
			want: 11 * sec,
		},
		{
			name: "First time after secont",
			t1:   1609495835,
			t2:   1609495824,
			want: 11 * sec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeStampDiff(tt.t1, tt.t2)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_TimeDiff(t *testing.T) {
	tests := []struct {
		name string
		t1   time.Time
		t2   time.Time
		want time.Duration
	}{
		{
			name: "Same time",
			t1:   time.Unix(0, 1609495835215478931),
			t2:   time.Unix(0, 1609495835215478931),
			want: time.Duration(0),
		},
		{
			name: "First time before secont",
			t1:   time.Unix(0, 1609495824215478931),
			t2:   time.Unix(0, 1609495835215478931),
			want: time.Second * 11,
		},
		{
			name: "First time after secont",
			t1:   time.Unix(0, 1609495835215478931),
			t2:   time.Unix(0, 1609495824215478931),
			want: time.Second * 11,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeDiff(tt.t1, tt.t2)
			h.AssertEqual(t, got, tt.want)
		})
	}
}
