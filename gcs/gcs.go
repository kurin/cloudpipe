package gcs

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
)

// Endpoint satisfies the gcspipe.endpoint interface.
type Endpoint struct {
	// TrueNames controls whether object names will be base64-encoded or not.  If
	// false, they will be so encoded.
	TrueNames bool

	// Overwrite controls whether objects are allowed to be overwritten.  If
	// false, writes to existing objects will fail.
	Overwrite bool

	client *storage.Client
	bucket string
}

// Writer returns a writer for the given object name.  If TrueNames is false,
// the name is encoded with base64, to prevent slashes from causing weirdness
// with the GCS bucket browser.
func (e *Endpoint) Writer(ctx context.Context, name string) (io.WriteCloser, error) {
	bucket := e.client.Bucket(e.bucket)
	if !e.TrueNames {
		name = base64.StdEncoding.EncodeToString([]byte(name))
	}
	obj := bucket.Object(name)
	if !e.Overwrite {
		obj = obj.WithConditions(storage.IfGenerationMatch(0))
	}
	return obj.NewWriter(ctx), nil
}

// New returns an Endpoint for the given bucket.  Auth should point to the
// project's private key in JSON format.
func New(ctx context.Context, auth, bucket string) (*Endpoint, error) {
	c, err := client(ctx, auth)
	if err != nil {
		return nil, err
	}
	return &Endpoint{
		client: c,
		bucket: bucket,
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
	return storage.NewClient(ctx, cloud.WithTokenSource(conf.TokenSource(ctx)))
}
