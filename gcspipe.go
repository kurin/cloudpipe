package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/kurin/gcspipe/counter"
	"github.com/kurin/gcspipe/gcs"

	"golang.org/x/net/context"
)

var (
	auth    = flag.String("auth", "", "Path to JSON keyfile.")
	destURI = flag.String("uri", "", "Destination URI.")
	b64name = flag.Bool("b64", false, "Base64-encode the object name.")
	verbose = flag.Bool("verbose", false, "Print progress every 10 seconds.")
)

type infoWriter struct {
	wc io.WriteCloser
	n  int
	c  *counter.Counter
}

func (iw *infoWriter) Write(p []byte) (int, error) {
	n, err := iw.wc.Write(p)
	iw.c.Add(n)
	iw.n += n
	return n, err
}

func (iw *infoWriter) Close() error {
	return iw.wc.Close()
}

func (iw *infoWriter) status() string {
	sent := size(iw.n)
	speed := size(iw.c.Per(time.Second))
	return fmt.Sprintf("wrote %s (%s/s)", sent, speed)
}

type endpoint interface {
	Writer(ctx context.Context) (io.WriteCloser, error)
}

func parseURI(ctx context.Context, uri string) (endpoint, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch url.Scheme {
	case "gcs":
		bucket := url.Host
		object := url.Path
		ep, err := gcs.New(ctx, *auth, bucket, object)
		if err != nil {
			return nil, err
		}
		ep.TrueNames = !*b64name
		return ep, nil
	}
	return nil, fmt.Errorf("%s: unknown scheme", url.Scheme)
}

func main() {
	flag.Parse()
	ctx := context.Background()
	ep, err := parseURI(ctx, *destURI)
	if err != nil {
		log.Fatal(err)
	}
	wc, err := ep.Writer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	w := &infoWriter{
		wc: wc,
		c:  counter.New(90*time.Second, time.Second),
	}
	if *verbose {
		go func() {
			for range time.Tick(10 * time.Second) {
				fmt.Println(w.status())
			}
		}()
	}
	if _, err := io.Copy(w, os.Stdin); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	if *verbose {
		fmt.Println(w.status())
	}
}

type size float64

func (s size) String() string {
	suffixes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	for i := 0; i <= len(suffixes); i++ {
		if s < 1024 {
			return fmt.Sprintf("%.2f%s", s, suffixes[i])
		}
		s /= 1024
	}
	return fmt.Sprintf("%.2fZB", s)
}
