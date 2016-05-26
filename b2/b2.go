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
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/kurin/blazer/b2"

	"golang.org/x/net/context"
)

type authTicket struct {
	ID  string `json:"accountId"`
	Key string `json:"accountKey"`
}

func readAuth(file string) (authTicket, error) {
	at := authTicket{}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return at, err
	}
	return at, json.Unmarshal(data, &at)
}

type Endpoint struct {
	TrueNames bool
	Resume    bool

	attrs *b2.Attrs
	b2    *b2.Bucket
	path  string
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
	bucket, err := client.Bucket(ctx, uri.Host)
	return &Endpoint{
		b2:   bucket,
		path: strings.TrimPrefix(uri.Path, "/"),
	}, nil
}

func (e *Endpoint) Writer(ctx context.Context) (io.WriteCloser, error) {
	name := e.path
	if !e.TrueNames {
		name = base64.StdEncoding.EncodeToString([]byte(name))
	}
	w := e.b2.Object(name).NewWriter(ctx)
	w.ConcurrentUploads = 4
	w.Resume = e.Resume
	if e.attrs != nil {
		w = w.WithAttrs(e.attrs)
	}
	return w, nil
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
