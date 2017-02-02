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

package cp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/google/subcommands"
	"github.com/kurin/cloudpipe/backends/b2"
	"github.com/kurin/cloudpipe/backends/file"
	"github.com/kurin/cloudpipe/backends/gcs"
)

type Cmd struct {
	resume bool
	conns  int
	auth   string
	labels string
}

func (*Cmd) Name() string     { return "cp" }
func (*Cmd) Synopsis() string { return "Copy a file." }

func (*Cmd) Usage() string {
	return "cp [flags] source destination\n"
}

func (c *Cmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.resume, "resume", false, "resume an upload (b2)")
	f.IntVar(&c.conns, "connections", 4, "number of concurrent connections (b2)")
	f.StringVar(&c.auth, "auth", "", "path to JSON key file (gcs, b2)")
	f.StringVar(&c.labels, "labels", "", "Comma-separated key=value pairs (gcs, b2).")
}

func (c *Cmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "%s", c.Usage())
		f.PrintDefaults()
		return subcommands.ExitUsageError
	}

	srcArg := f.Args()[0]
	dstArg := f.Args()[1]

	src, err := c.parseURI(ctx, srcArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", srcArg, err)
		return subcommands.ExitFailure
	}

	dst, err := c.parseURI(ctx, dstArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", dstArg, err)
		return subcommands.ExitFailure
	}

	if c.labels != "" {
		src.Label(c.labels)
		dst.Label(c.labels)
	}

	r, err := src.Reader(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	w, err := dst.Writer(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	if _, err := io.Copy(w, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}
	if err := w.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type endpoint interface {
	Writer(ctx context.Context) (io.WriteCloser, error)
	Reader(ctx context.Context) (io.ReadCloser, error)
	Label(string)
}

type std struct{}

func (std) Label(string)                                   {}
func (std) Writer(context.Context) (io.WriteCloser, error) { return os.Stdout, nil }
func (std) Reader(context.Context) (io.ReadCloser, error)  { return os.Stdin, nil }

func (c *Cmd) parseURI(ctx context.Context, uri string) (endpoint, error) {
	if uri == "-" {
		return std{}, nil
	}
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch url.Scheme {
	case "gcs":
		ep, err := gcs.New(ctx, c.auth, url)
		if err != nil {
			return nil, err
		}
		return ep, nil
	case "b2":
		ep, err := b2.New(ctx, url)
		if err != nil {
			return nil, err
		}
		ep.Resume = c.resume
		ep.Connections = c.conns
		return ep, nil
	case "file", "":
		return file.Path(url.Path), nil
	}
	return nil, fmt.Errorf("%s: unknown scheme", url.Scheme)
}
