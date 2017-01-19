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

package gcs

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"google.golang.org/api/option"

	"cloud.google.com/go/storage"

	"golang.org/x/oauth2/google"
)

// Endpoint satisfies the cloudpipe.endpoint interface.
type Endpoint struct {
	// TrueNames controls whether object names will be base64-encoded or not.  If
	// false, they will be so encoded.
	TrueNames bool

	// Overwrite controls whether objects are allowed to be overwritten.  If
	// false, writes to existing objects will fail.
	Overwrite bool

	client         *storage.Client
	bucket, object string
	m              map[string]string
}

// Writer returns a writer for the given object name.  If TrueNames is false,
// the name is encoded with base64, to prevent slashes from causing weirdness
// with the GCS bucket browser.
func (e *Endpoint) Writer(ctx context.Context) (io.WriteCloser, error) {
	bucket := e.client.Bucket(e.bucket)
	name := e.object
	if !e.TrueNames {
		name = base64.StdEncoding.EncodeToString([]byte(name))
	}
	obj := bucket.Object(name)
	if !e.Overwrite {
		obj = obj.If(storage.Conditions{DoesNotExist: true})
	}
	w := obj.NewWriter(ctx)
	w.ObjectAttrs.Metadata = e.m
	return w, nil
}

func (e *Endpoint) Label(l string) {
	labels := strings.Split(l, ",")
	e.m = make(map[string]string)
	for _, l := range labels {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			continue
		}
		e.m[parts[0]] = parts[1]
	}
}

// New returns an Endpoint for the given bucket.  Auth should point to the
// project's private key in JSON format.
func New(ctx context.Context, auth string, url *url.URL) (*Endpoint, error) {
	c, err := client(ctx, auth)
	if err != nil {
		return nil, err
	}
	bucket := url.Host
	object := url.Path
	object = strings.TrimPrefix(object, "/")
	return &Endpoint{
		client: c,
		bucket: bucket,
		object: object,
	}, nil
}

func client(ctx context.Context, auth string) (*storage.Client, error) {
	if auth == "" {
		return nil, fmt.Errorf("no auth credentials supplied")
	}
	jsonKey, err := ioutil.ReadFile(auth)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(jsonKey, storage.ScopeReadWrite)
	if err != nil {
		return nil, err
	}
	return storage.NewClient(ctx, option.WithTokenSource(conf.TokenSource(ctx)))
}
