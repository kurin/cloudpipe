package main

import (
	"testing"
	"time"
)

type addOp struct {
	t time.Time
	i int
}

func TestCounter(t *testing.T) {
	table := []struct {
		c *counter
		v []addOp
		t time.Time
		w int
	}{
		{
			c: &counter{
				vals: make([]int, 10),
				res:  time.Second,
			},
			v: []addOp{
				{
					t: time.Unix(0, 1),
					i: 1,
				},
			},
			t: time.Unix(4, 0),
			w: 1,
		},
		{
			c: &counter{
				vals: make([]int, 10),
				res:  time.Second,
			},
			v: []addOp{
				{
					t: time.Unix(0, 1),
					i: 1,
				},
				{
					t: time.Unix(10, 1),
					i: 1,
				},
			},
			t: time.Unix(11, 0),
			w: 1,
		},
	}

	for _, ent := range table {
		for _, op := range ent.v {
			ent.c.add(op.t, op.i)
		}
		got := ent.c.get(ent.t)
		if got != ent.w {
			t.Errorf("counter %v: got %d, want %d", ent.c, got, ent.w)
		}
	}
}

func TestPerSecond(t *testing.T) {
	table := []struct {
		c *counter
		v []addOp
		t time.Time
		w float64
	}{
		{
			c: &counter{
				vals: make([]int, 4),
				res:  time.Second,
			},
			v: []addOp{
				{
					t: time.Unix(0, 0),
				},
				{
					t: time.Unix(1, 5e8),
					i: 1024 * 1024,
				},
				{
					t: time.Unix(2, 5e8),
					i: 1024 * 1024,
				},
				{
					t: time.Unix(3, 5e8),
					i: 1024 * 1024,
				},
				{
					t: time.Unix(4, 5e8),
					i: 1024 * 1024,
				},
			},
			t: time.Unix(4, 6e8),
			w: 1024 * 1024,
		},
		{
			c: &counter{
				vals: make([]int, 2),
				res:  time.Second,
			},
			v: []addOp{
				{
					t: time.Unix(0, 0),
				},
				{
					t: time.Unix(10, 5e8),
					i: 30,
				},
				{
					t: time.Unix(11, 5e8),
					i: 10,
				},
			},
			t: time.Unix(11, 6e8),
			w: 20,
		},
		{
			c: &counter{
				vals: make([]int, 2),
				res:  time.Second,
			},
			v: []addOp{
				{
					t: time.Unix(0, 0),
				},
				{
					t: time.Unix(10, 5e8),
					i: 1024 * 1024 * 2,
				},
				{
					t: time.Unix(11, 5e8),
					i: 0,
				},
			},
			t: time.Unix(11, 6e8),
			w: 1024 * 1024,
		},
		{
			c: &counter{
				vals: make([]int, 90),
				res:  time.Second,
			},
			v: []addOp{
				{
					t: time.Unix(10, 0),
					i: 1024 * 1024 * 2,
				},
				{
					t: time.Unix(11, 0),
					i: 0,
				},
			},
			t: time.Unix(12, 0),
			w: 1024 * 1024,
		},
	}

	for _, ent := range table {
		for _, op := range ent.v {
			ent.c.add(op.t, op.i)
		}
		got := ent.c.perSecond(ent.t)
		if got != ent.w {
			t.Errorf("counter %v: got %f, want %f", ent.c, got, ent.w)
		}
	}
}

func BenchmarkCounter(b *testing.B) {
	c := &counter{
		vals: make([]int, 900),
		res:  time.Second,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.add(time.Unix(0, int64(i)*1000000), i)
	}
}
