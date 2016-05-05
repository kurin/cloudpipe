package main

import "testing"

func TestSize(t *testing.T) {
	table := []struct {
		s    size
		want string
	}{
		{
			s:    1024,
			want: "1.00kB",
		},
		{
			s:    1024 * 1024,
			want: "1.00MB",
		},
		{
			s:    42.42 * 1024 * 1024 * 1024,
			want: "42.42GB",
		},
	}

	for _, ent := range table {
		if ent.s.String() != ent.want {
			t.Errorf("got %q, want %q", ent.s, ent.want)
		}
	}
}
