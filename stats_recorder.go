package tcpez

import (
	"github.com/golang/glog"
	"time"
)

type StatsRecorder interface {
	Timer(stat string, amount int64)
	DurationTimer(stat string, begin time.Time, end time.Time)
	Gauge(stat string, amount int64)
	Counter(stat string, amount int64)
	Increment(stat string)
}

type DebugStatsRecorder struct{}

func (s *DebugStatsRecorder) log(stat string, amount int64) {
	glog.V(2).Infof("debug stats %s %v", stat, amount)
}

func (s *DebugStatsRecorder) Timer(stat string, amount int64) {
	s.log(stat, amount)
}

func (s *DebugStatsRecorder) DurationTimer(stat string, begin time.Time, end time.Time) {
	amount := int64(end.Sub(begin) / time.Millisecond)
	s.log(stat, amount)
}

func (s *DebugStatsRecorder) Gauge(stat string, amount int64) {
	s.log(stat, amount)
}

func (s *DebugStatsRecorder) Counter(stat string, amount int64) {
	s.log(stat, amount)
}

func (s *DebugStatsRecorder) Increment(stat string) {
	s.log(stat, 1)
}
