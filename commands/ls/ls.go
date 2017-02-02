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

package ls

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
	auth   string
	hidden bool
}

func (*Cmd) Name() string     { return "ls" }
func (*Cmd) Synopsis() string { return "List the objects in a bucket." }

func (*Cmd) Usage() string {
	return "ls [flags] path\n"
}

func (c *Cmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.auth, "auth", "", "path to JSON key file (gcs, b2)")
	f.BoolVar(&c.hidden, "hidden", false, "list hidden files as well (b2)")
}

func (c *Cmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "%s", c.Usage())
		f.PrintDefaults()
		return subcommands.ExitUsageError
	}

	pathArg := f.Args()[0]

	path, err := c.parseURI(ctx, pathArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", pathArg, err)
		return subcommands.ExitFailure
	}

	names, errs, err := path.List(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	for name := range names {
		fmt.Println(name)
	}

	if err, ok := <-errs; ok {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type endpoint interface {
	List(context.Context) (chan string, chan error, error)
}

func (c *Cmd) parseURI(ctx context.Context, uri string) (endpoint, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch url.Scheme {
	/*case "gcs":
	ep, err := gcs.New(ctx, c.auth, url)
	if err != nil {
		return nil, err
	}
	return ep, nil
	*/
	case "b2":
		ep, err := b2.New(ctx, c.auth, url)
		if err != nil {
			return nil, err
		}
		ep.Hidden = c.hidden
		return ep, nil
		//case "file", "":
		//	return file.Path(url.Path), nil
	}
	return nil, fmt.Errorf("%s: unknown scheme", url.Scheme)
}
