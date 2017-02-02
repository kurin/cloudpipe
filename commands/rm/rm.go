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

package rm

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
	auth    string
	hide    bool
	hidden  bool
	all     bool
	recurse bool
	threads int
}

func (*Cmd) Name() string     { return "rm" }
func (*Cmd) Synopsis() string { return "Remove an object or bucket." }

func (*Cmd) Usage() string {
	return "rm [flags] file\n"
}

func (c *Cmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.auth, "auth", "", "path to JSON key file (gcs, b2)")
	f.BoolVar(&c.hide, "hide", false, "hide an object instead of deleting it (b2)")
	f.BoolVar(&c.hidden, "hidden", false, "operate on hidden files as well (b2)")
	f.BoolVar(&c.all, "all", false, "remove all versions of a file, not just the most recent (b2)")
	f.BoolVar(&c.recurse, "r", false, "recursively delete objects under a given path (b2, gcs)")
	f.IntVar(&c.threads, "threads", 1, "remove this many objects in parallel (b2, gcs)")
}

func (c *Cmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "%s", c.Usage())
		f.PrintDefaults()
		return subcommands.ExitUsageError
	}

	rmArg := f.Args()[0]

	rm, err := c.parseURI(ctx, rmArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", rmArg, err)
		return subcommands.ExitFailure
	}

	if err := rm.Remove(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type endpoint interface {
	Remove(context.Context) error
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
		ep.Hide = c.hide
		ep.Hidden = c.hidden
		ep.Recursive = c.recurse
		ep.Bucket = url.Path == ""
		return ep, nil
		//case "file", "":
		//	return file.Path(url.Path), nil
	}
	return nil, fmt.Errorf("%s: unknown scheme", url.Scheme)
}
