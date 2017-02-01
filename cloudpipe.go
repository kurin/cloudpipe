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
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	"github.com/kurin/cloudpipe/b2"
	"github.com/kurin/cloudpipe/file"
	"github.com/kurin/cloudpipe/gcs"
)

var (
	auth        = flag.String("auth", "", "Path to JSON keyfile (gcs, b2).")
	resume      = flag.Bool("resume", false, "Resume an upload (b2).")
	connections = flag.Int("connections", 4, "Number of simultaneous connections (b2).")
	labels      = flag.String("labels", "", "Comma-separated key=value pairs (gcs, b2).")
)

type endpoint interface {
	Writer(ctx context.Context) (io.WriteCloser, error)
	Reader(ctx context.Context) (io.ReadCloser, error)
	Label(string)
}

type std struct{}

func (std) Label(string)                                   {}
func (std) Writer(context.Context) (io.WriteCloser, error) { return os.Stdout, nil }
func (std) Reader(context.Context) (io.ReadCloser, error)  { return os.Stdin, nil }

func parseURI(ctx context.Context, uri string) (endpoint, error) {
	if uri == "-" {
		return std{}, nil
	}
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch url.Scheme {
	case "gcs":
		ep, err := gcs.New(ctx, *auth, url)
		if err != nil {
			return nil, err
		}
		return ep, nil
	case "b2":
		ep, err := b2.New(ctx, *auth, url)
		if err != nil {
			return nil, err
		}
		ep.Resume = *resume
		return ep, nil
	case "file", "":
		return file.Path(url.Path), nil
	}
	return nil, fmt.Errorf("%s: unknown scheme", url.Scheme)
}

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	srcArg := flag.Args()[0]
	dstArg := flag.Args()[1]

	ctx := context.Background()

	src, err := parseURI(ctx, srcArg)
	if err != nil {
		log.Fatal(err)
	}

	dst, err := parseURI(ctx, dstArg)
	if err != nil {
		log.Fatal(err)
	}

	if *labels != "" {
		src.Label(*labels)
		dst.Label(*labels)
	}

	r, err := src.Reader(ctx)
	if err != nil {
		log.Fatal(err)
	}

	w, err := dst.Writer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(w, r); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
}
