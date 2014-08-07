package tcpez

import (
	"github.com/bmizerany/assert"
	"strings"
	"testing"
	"time"
)

func TestSpanCreation(t *testing.T) {
	span := NewSpan("")
	assert.T(t, span != nil)
	span.Start("test")
	assert.Equal(t, 1, len(span.SubSpans))
	dur := span.Finish("test")
	assert.Equal(t, 1, len(span.SubSpans))
	assert.T(t, dur > 0)
}

func TestSubSpan(t *testing.T) {
	n := time.Now()
	span := NewSpan("")
	assert.T(t, span != nil)
	span.SubSpan("test").Finish(n)
	assert.Equal(t, 1, len(span.SubSpans))
}

func TestMultipleSubSpans(t *testing.T) {
	span := NewSpan("")
	assert.T(t, span != nil)
	span.Start("test")
	assert.Equal(t, 1, len(span.SubSpans))
	span.Start("test2")
	dur := span.Finish("test2")
	assert.Equal(t, 2, len(span.SubSpans))
	assert.T(t, dur > 0)
	assert.T(t, span.Duration("test2") > 0)
	dur = span.Finish("test3")
	assert.Equal(t, 3, len(span.SubSpans))
	assert.T(t, dur == 0)
}

func TestSubSpanWithDuration(t *testing.T) {
	span := NewSpan("")
	assert.T(t, span != nil)
	span.SubSpanWithDuration("test", 42.4)
	assert.Equal(t, 1, len(span.SubSpans))
	assert.Equal(t, 42.4, span.MillisecondDuration("test"))
}

func TestMergeJSON(t *testing.T) {
	json := `
        {"subspans":{"preloading_fonts":4013.933000001998,"setEvent":0.6599999978789128,"createCard":3.197000001819106,"renderEvent":232.6809999976831,"renderCanvas":97.5880000005418,"updateDisplayList":97.50500000154716,"paintDisplayList":12.435000000550644,"renderText":75.1589999999851,"toDataURL":43.588000000454485},"attrs":{},"counters":{"renderComplete":1},"id":"6eede8679ad2888e"}
        `
	span := NewSpan("")
	assert.T(t, span != nil)
	err := span.MergeJSON(json)
	assert.T(t, err == nil)
	assert.Equal(t, 9, len(span.SubSpans))
	assert.Equal(t, 1, len(span.Counters))
}

func TestIncrement(t *testing.T) {
	span := NewSpan("")
	assert.T(t, span != nil)
	span.Increment("test")
	assert.Equal(t, 1, len(span.Counters))
	span.Increment("test")
	assert.Equal(t, 1, len(span.Counters))
	span.Increment("test2")
	assert.Equal(t, 2, len(span.Counters))
	assert.Equal(t, int64(3), span.Increment("test"))
}

func TestAttr(t *testing.T) {
	span := NewSpan("")
	assert.T(t, span != nil)
	span.Attr("command", "GET")
	assert.Equal(t, 1, len(span.Attrs))
	span.Attr("response", "OK")
	assert.Equal(t, 2, len(span.Attrs))
}

func TestString(t *testing.T) {
	span := NewSpan("")
	assert.T(t, span != nil)
	span.Start("duration")
	span.Start("inc")
	span.Increment("counter")
	span.Finish("inc")
	span.Start("add")
	span.Add("other_counter", 5)
	span.Finish("add")
	span.Finish("duration")
	span.Attr("command", "GET")
	span.Attr("response", "OK")
	assert.Equal(t, 2, len(span.Attrs))
	assert.Equal(t, 2, len(span.Counters))
	assert.Equal(t, 3, len(span.SubSpans))
	assert.T(t, span.SubSpans["duration"].Duration() > span.SubSpans["inc"].Duration())
	s := span.String()
	assert.T(t, strings.Contains(s, "command=GET response=OK"))
	assert.T(t, strings.Contains(s, "duration="))
	assert.T(t, strings.Contains(s, "inc="))
	assert.T(t, strings.Contains(s, "add="))
	assert.T(t, strings.Contains(s, "counter=1"))
	assert.T(t, strings.Contains(s, "other_counter=5"))
}
