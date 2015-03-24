package tcpez

import (
	"github.com/cactus/go-statsd-client/statsd"
	"time"
)

// StatsdStatsRecorder holds the data for a statsd external instance
// that data is flushed to periodically. It fuffils the StatsRecorder
// interface so it can be passed to Tcpez Servers
type StatsdStatsRecorder struct {
	address   string
	namespace string
	client    statsd.Statter
	counter   chan *StatsdStat
	timer     chan *StatsdStat
	gauge     chan *StatsdStat
}

type StatsdStat struct {
	stat   string
	amount int64
}

// NewStatsdStatsRecorder accepts an address and a namespace as strings and returns a pointer
// to an initialized Stat instance.
func NewStatsdStatsRecorder(address, namespace string) *StatsdStatsRecorder {
	client, err := statsd.NewClient(address, namespace)

	if err != nil {
		return nil
	}

	stats := &StatsdStatsRecorder{
		address: address,
		// arbitrary buffer size just to support as much
		// non blocking as possible
		counter: make(chan *StatsdStat, 100),
		timer:   make(chan *StatsdStat, 100),
		gauge:   make(chan *StatsdStat, 100),
		client:  client,
	}

	go stats.Start()
	return stats
}

// Timer accepts a stat name and an amount and will send that stat to the
// Stat server.
func (stats *StatsdStatsRecorder) Timer(stat string, amount int64) {
	stats.timer <- &StatsdStat{stat, amount}
}

// DurationTimer accepts a stat, a begin time, and an end time, and sends
// the appropriately massaged value to the stat server.
func (stats *StatsdStatsRecorder) DurationTimer(stat string, begin time.Time, end time.Time) {
	amount := int64(end.Sub(begin) / time.Millisecond)
	stats.timer <- &StatsdStat{stat, amount}
}

// Gauge accepts the stat name and an amount and transmits the gauge stat
// to the stat server.
func (stats *StatsdStatsRecorder) Gauge(stat string, amount int64) {
	stats.gauge <- &StatsdStat{stat, amount}
}

// Counter accepts the stat name and an amount and transmits the counter
// stat to the server.
func (stats *StatsdStatsRecorder) Counter(stat string, amount int64) {
	stats.counter <- &StatsdStat{stat, amount}
}

// Increment is the same as calling Counter with the amount 1
func (stats *StatsdStatsRecorder) Increment(stat string) {
	stats.counter <- &StatsdStat{stat, 1}
}

// Start starts the stat goroutine and reads from its Timing, Gauge, and
// Counter channels, sending the data over the network.
func (stats *StatsdStatsRecorder) Start() {
	for {
		select {
		case stat := <-stats.timer:
			stats.client.Timing(stat.stat, stat.amount, 1.0)
		case stat := <-stats.gauge:
			stats.client.Gauge(stat.stat, stat.amount, 1.0)
		case stat := <-stats.counter:
			stats.client.Inc(stat.stat, stat.amount, 1.0)
		}
	}
}
