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

	b2   *b2.Bucket
	path string
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
	return w, nil
}