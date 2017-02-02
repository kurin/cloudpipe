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

package b2

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/kurin/blazer/b2"
	"github.com/kurin/cloudpipe/internal/b2assets"
)

type authTicket struct {
	ID  string `json:"accountId"`
	Key string `json:"accountKey"`
}

var (
	statusFuncMap = template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}
	statusTemplate = template.Must(template.New("status").Funcs(statusFuncMap).Parse(string(b2assets.MustAsset("data/status.html"))))
)

func readAuth(file string) (authTicket, error) {
	at := authTicket{}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return at, err
	}
	return at, json.Unmarshal(data, &at)
}

type Endpoint struct {
	Connections int
	Resume      bool

	Hide      bool
	Hidden    bool
	Recursive bool
	Bucket    bool

	attrs  *b2.Attrs
	b2     *b2.Client
	bucket string
	path   string
}

func New(ctx context.Context, auth string, uri *url.URL) (*Endpoint, error) {
	at, err := readAuth(auth)
	if err != nil {
		return nil, err
	}
	client, err := b2.NewClient(ctx, at.ID, at.Key)
	if err != nil {
		return nil, err
	}

	hf := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		st := client.Status()
		statusTemplate.Execute(rw, st)
	})

	http.Handle("/progress", hf)
	go func() { fmt.Println(http.ListenAndServe("0.0.0.0:8822", nil)) }()

	return &Endpoint{
		b2:     client,
		bucket: uri.Host,
		path:   strings.TrimPrefix(uri.Path, "/"),
	}, nil
}

func (e *Endpoint) Writer(ctx context.Context) (io.WriteCloser, error) {
	bucket, err := e.b2.NewBucket(ctx, e.bucket, nil)
	if err != nil {
		return nil, err
	}
	name := e.path
	w := bucket.Object(name).NewWriter(ctx)
	w.ConcurrentUploads = e.Connections
	w.Resume = e.Resume
	if e.attrs != nil {
		w = w.WithAttrs(e.attrs)
	}
	return w, nil
}

func (e *Endpoint) Reader(ctx context.Context) (io.ReadCloser, error) {
	bucket, err := e.b2.Bucket(ctx, e.bucket)
	if err != nil {
		return nil, err
	}
	r := bucket.Object(e.path).NewReader(ctx)
	r.ConcurrentDownloads = e.Connections
	return r, nil
}

func (e *Endpoint) Label(l string) {
	labels := strings.Split(l, ",")
	m := make(map[string]string)
	for _, label := range labels {
		i := strings.Index(label, "=")
		if i < 0 {
			continue
		}
		key, val := label[:i], label[i+1:]
		m[strings.Trim(key, " ")] = strings.Trim(val, " ")
	}
	e.attrs = &b2.Attrs{Info: m}
}

func (e *Endpoint) List(ctx context.Context) (chan string, chan error, error) {
	bucket, err := e.b2.Bucket(ctx, e.bucket)
	if err != nil {
		return nil, nil, err
	}

	sch := make(chan string)
	ech := make(chan error)

	go func() {
		defer close(sch)
		defer close(ech)

		lister := bucket.ListCurrentObjects
		if e.Hidden {
			lister = bucket.ListObjects
		}

		c := &b2.Cursor{Prefix: e.path, Delimiter: "/"}
		for {
			list, ncur, err := lister(ctx, 100, c)
			if err != nil && err != io.EOF {
				ech <- err
				return
			}
			c = ncur
			for _, obj := range list {
				sch <- obj.Name()
			}
			if err == io.EOF {
				return
			}
		}
	}()

	return sch, ech, nil
}

func (e *Endpoint) Remove(ctx context.Context) error {
	bucket, err := e.b2.Bucket(ctx, e.bucket)
	if err != nil {
		return err
	}
	if !e.Recursive {
		if e.Bucket {
			return bucket.Delete(ctx)
		}

		obj := bucket.Object(e.path)
		if e.Hide {
			return obj.Hide(ctx)
		}
		return obj.Delete(ctx)
	}

	lister := bucket.ListCurrentObjects
	if e.Hidden {
		lister = bucket.ListObjects
	}

	c := &b2.Cursor{Prefix: e.path}
	for {
		list, ncur, err := lister(ctx, 100, c)
		if err != nil && err != io.EOF {
			return err
		}
		c = ncur
		for _, obj := range list {
			op := obj.Delete
			if e.Hide {
				op = obj.Hide
			}
			if err := op(ctx); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
	}

	if e.Bucket {
		return bucket.Delete(ctx)
	}

	return nil
}
