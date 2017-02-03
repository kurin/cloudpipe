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

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/kurin/cloudpipe/commands/b2config"
	"github.com/kurin/cloudpipe/commands/cp"
	"github.com/kurin/cloudpipe/commands/ls"
	"github.com/kurin/cloudpipe/commands/rm"
	"github.com/kurin/cloudpipe/commands/stat"
)

var (
	auth        = flag.String("auth", "", "Path to JSON keyfile (gcs, b2).")
	resume      = flag.Bool("resume", false, "Resume an upload (b2).")
	connections = flag.Int("connections", 4, "Number of simultaneous connections (b2).")
	labels      = flag.String("labels", "", "Comma-separated key=value pairs (gcs, b2).")
)

func main() {
	subcommands.Register(&cp.Cmd{}, "")
	subcommands.Register(&rm.Cmd{}, "")
	subcommands.Register(&ls.Cmd{}, "")
	subcommands.Register(&stat.Cmd{}, "")
	subcommands.Register(&b2config.Cmd{}, "configuration")
	flag.Parse()

	ctx := context.Background()

	os.Exit(int(subcommands.Execute(ctx)))
}
