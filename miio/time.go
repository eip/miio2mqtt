package miio

import (
	"time"
)

type TimeStamp uint32

func Stamp(t time.Time) TimeStamp {
	return TimeStamp(t.Unix())
}

func Now() TimeStamp {
	return Stamp(time.Now())
}

func (ts TimeStamp) String() string {
	return (time.Duration(ts) * time.Second).String()
}

func (ts TimeStamp) Time() time.Time {
	return time.Unix(int64(ts), 0).UTC()
}

func TimeStampDiff(t1, t2 TimeStamp) TimeStamp {
	if t1 > t2 {
		return t1 - t2
	}
	return t2 - t1
}

func TimeDiff(t1, t2 time.Time) time.Duration {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return time.Duration(diff)
}
