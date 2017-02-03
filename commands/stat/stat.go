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

package stat

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/google/subcommands"
	"github.com/kurin/cloudpipe/backends/b2"
)

type Cmd struct {
}

func (*Cmd) Name() string     { return "stat" }
func (*Cmd) Synopsis() string { return "Print information about an object." }

func (*Cmd) Usage() string {
	return "stat path\n"
}

func (c *Cmd) SetFlags(f *flag.FlagSet) {
}

func (c *Cmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "%s", c.Usage())
		f.PrintDefaults()
		return subcommands.ExitUsageError
	}

	statArg := f.Args()[0]

	stat, err := c.parseURI(ctx, statArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", statArg, err)
		return subcommands.ExitFailure
	}

	txt, err := stat.Stat(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	fmt.Print(txt)

	return subcommands.ExitSuccess
}

type endpoint interface {
	Stat(context.Context) (string, error)
}

func (c *Cmd) parseURI(ctx context.Context, uri string) (endpoint, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch url.Scheme {
	case "b2":
		ep, err := b2.New(ctx, url)
		if err != nil {
			return nil, err
		}
		return ep, nil
	}
	return nil, fmt.Errorf("%s: unknown scheme", url.Scheme)
}
