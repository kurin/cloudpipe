package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/kurin/gcspipe/b2"
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
	rate := speed(iw.c.Per(time.Second))
	return fmt.Sprintf("wrote %s (%s)", sent, rate)
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
		ep, err := gcs.New(ctx, *auth, url)
		if err != nil {
			return nil, err
		}
		ep.TrueNames = !*b64name
		return ep, nil
	case "b2":
		return b2.New(ctx, *auth, url)
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
			var max int
			for range time.Tick(time.Second) {
				s := w.status()
				if max > len(s) {
					max = len(s)
				}
				fmt.Printf("\r%-*s\r", max-len(s), w.status())
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

var suffixes = []string{"", "k", "M", "G", "T", "P", "E"}

type speed float64

func (s speed) String() string {
	s *= 8
	for i := 0; i <= len(suffixes); i++ {
		if s < 1024 {
			return fmt.Sprintf("%.2f%sbps", s, suffixes[i])
		}
		s /= 1024
	}
	return fmt.Sprintf("%.2fZbps", s)
}

type size float64

func (s size) String() string {
	for i := 0; i <= len(suffixes); i++ {
		if s < 1024 {
			return fmt.Sprintf("%.2f%sB", s, suffixes[i])
		}
		s /= 1024
	}
	return fmt.Sprintf("%.2fZB", s)
}
