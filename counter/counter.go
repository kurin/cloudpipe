package counter

import (
	"sync"
	"time"
)

// Counter is a type that measures a quantity of time-series events.
type Counter struct {
	// Clock is a function that is called for the current time.  If nil, time.Now
	// is used.
	Clock func() time.Time
	mu    sync.Mutex
	vals  []int
	res   time.Duration
	prev  time.Time
	start time.Time
	once  sync.Once
}

// New returns a new counter with a given span and resolution.  A counter's
// span is the total amount of time for which the counter will hold data.  The
// resolution is the smallest period of time a counter is aware of.  For
// example, a counter with a span and resolution of five minutes is equal to an
// integer that is reset to 0 every five minutes.  However, a counter with a
// span of five minutes and a resolution of one second will lose a second's
// worth of events every second.
//
// A counter's resolution must evenly divide into its span.
func New(span, resolution time.Duration) *Counter {
	return &Counter{
		vals: make([]int, int(span/resolution)),
		res:  resolution,
	}
}

func (c *Counter) now() time.Time {
	if c.Clock == nil {
		return time.Now()
	}
	return c.Clock()
}

// Span returns the amount of time for which the counter is valid.
func (c *Counter) Span() time.Duration {
	return c.span(c.now())
}

func (c *Counter) span(now time.Time) time.Duration {
	total := c.res * time.Duration(len(c.vals))
	elapsed := now.Sub(c.start)
	if elapsed < total {
		return elapsed
	}
	return total
}

// Per returns the average number of events in a given interval given the data
// currently in the counter.  For example, given a counter with a span of five
// minutes, one could find the number of events in every thirty seconds with
// c.Per(30*time.Second).
func (c *Counter) Per(interval time.Duration) float64 {
	return c.per(c.now(), interval)
}

func (c *Counter) per(now time.Time, ival time.Duration) float64 {
	count := float64(c.get(now))
	s := float64(c.span(now))

	return count * float64(ival) / s
}

func (c *Counter) bucket(t time.Time) int {
	return int(t.UnixNano()/int64(c.res)) % len(c.vals)
}

// sweep marks any buckets zero if they have not been updated since the last
// update.  c.mu must be held by the caller.
func (c *Counter) sweep(now time.Time) {
	if c.prev.IsZero() {
		return
	}
	// If every bucket is invalid, mark all zero.
	if now.UnixNano()-c.prev.UnixNano() > int64(c.res)*int64(len(c.vals)) {
		for i := range c.vals {
			c.vals[i] = 0
		}
		return
	}

	prevBucket := int64(c.bucket(c.prev))
	numBuckets := (now.UnixNano() - c.prev.UnixNano()) / int64(c.res)
	for i := prevBucket + 1; i < prevBucket+numBuckets; i++ {
		b := int(i) % len(c.vals)
		c.vals[b] = 0
	}
}

// Add adds a number of events to the counter with a timestamp of now.
func (c *Counter) Add(i int) {
	c.add(c.now(), i)
}

func (c *Counter) add(now time.Time, inc int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.once.Do(func() {
		c.start = now
	})
	c.sweep(now)

	bucket := c.bucket(now)
	if now.UnixNano()/int64(c.res) != c.prev.UnixNano()/int64(c.res) {
		c.vals[bucket] = 0
	}
	c.vals[bucket] += inc
	c.prev = now
}

// Get returns the total number of events in the counter.
func (c *Counter) Get() int {
	return c.get(c.now())
}

func (c *Counter) get(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweep(now)

	var i int
	for _, v := range c.vals {
		i += v
	}
	return i
}
