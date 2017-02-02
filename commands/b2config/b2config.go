// Copyright 2017, Google
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

package b2config

import (
	"context"
	"flag"
	"log"

	"github.com/google/subcommands"
	"github.com/kurin/cloudpipe/backends/b2"
)

type Cmd struct {
	id  string
	key string
}

func (*Cmd) Name() string     { return "b2config" }
func (*Cmd) Synopsis() string { return "Set B2 account options." }

func (*Cmd) Usage() string {
	return "b2config [flags]\n"
}

func (c *Cmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.id, "id", "", "Account ID.")
	f.StringVar(&c.key, "key", "", "Account key.")
}

func (c *Cmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	cfg := &b2.Config{
		ID:  c.id,
		Key: c.key,
	}
	if err := b2.Save(cfg); err != nil {
		log.Printf("b2config: %v", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
