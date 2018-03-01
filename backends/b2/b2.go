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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/kurin/blazer/b2"
	"github.com/kurin/cloudpipe/internal/b2assets"
)

var (
	statusFuncMap = template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"pRange": func(i int) string {
			f := float64(i)
			min := int(math.Pow(2, f)) - 1
			max := min + int(math.Pow(2, f))
			return fmt.Sprintf("%v - %v", time.Duration(min)*time.Millisecond, time.Duration(max)*time.Millisecond)
		},
		"lookUp": func(m map[string]int, s string) int {
			return m[s]
		},
	}
	statusTemplate = template.Must(template.New("status").Funcs(statusFuncMap).Parse(string(b2assets.MustAsset("data/status.html"))))
)

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

type Config struct {
	ID  string `json:"accountId"`
	Key string `json:"accountKey"`
}

func Save(c *Config) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(u.HomeDir, ".cloudpipe_b2"))
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	return enc.Encode(c)
}

func loadAuth() (*Config, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(u.HomeDir, ".cloudpipe_b2"))
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(f)
	c := &Config{}
	if err := dec.Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}

type status struct {
	Writers    map[string]*b2.WriterStatus
	Readers    map[string]*b2.ReaderStatus
	MethodHist map[string][]int
	Calls      map[string]int
}

func New(ctx context.Context, uri *url.URL) (*Endpoint, error) {
	at, err := loadAuth()
	if err != nil {
		return nil, err
	}
	client, err := b2.NewClient(ctx, at.ID, at.Key)
	if err != nil {
		return nil, err
	}

	hf := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		st := client.Status()
		var s status
		s.Readers = st.Readers
		s.Writers = st.Writers
		s.Calls = st.MethodInfo.CountByMethod()
		s.MethodHist = st.MethodInfo.HistogramByMethod()
		statusTemplate.Execute(rw, s)
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
	w.ChunkSize = 5e6
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

func fsize(s int64) string {
	sfxs := "BkMGT"
	f := float64(s)
	for i := 0; i < 5; i++ {
		if f < 1024 {
			return fmt.Sprintf("%.2f%c", f, sfxs[i])
		}
		f /= 1024
	}
	return fmt.Sprintf("%dB", s)
}

func (e *Endpoint) Stat(ctx context.Context) (string, error) {
	bucket, err := e.b2.Bucket(ctx, e.bucket)
	if err != nil {
		return "", err
	}
	attrs, err := bucket.Object(e.path).Attrs(ctx)
	if err != nil {
		return "", err
	}
	kv := map[string]string{
		"Name":         attrs.Name,
		"Size":         fsize(attrs.Size),
		"Content-Type": attrs.ContentType,
		"Uploaded":     attrs.UploadTimestamp.Format(time.RubyDate),
	}
	order := []string{"Name", "Size", "Content-Type", "Uploaded"}
	if !attrs.LastModified.IsZero() {
		kv["Last Modified"] = attrs.LastModified.Format(time.RubyDate)
		order = append(order, "Last Modified")
	}
	if len(attrs.SHA1) == 20 {
		kv["SHA1"] = attrs.SHA1
		order = append(order, "SHA1")
	}
	for key, val := range attrs.Info {
		kv[key] = val
		order = append(order, key)
	}

	var max int

	for key := range kv {
		if len(key) > max {
			max = len(key)
		}
	}

	buf := &bytes.Buffer{}
	for _, key := range order {
		fmt.Fprintf(buf, "%*s: %s\n", max, key, kv[key])
	}
	return buf.String(), nil
}
