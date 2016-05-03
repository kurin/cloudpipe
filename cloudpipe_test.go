// Copyright 2016, Google
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
