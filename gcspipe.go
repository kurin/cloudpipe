package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
)

var (
	auth       = flag.String("auth", "", "Path to JSON keyfile.")
	bucketName = flag.String("bucket", "", "Bucket name.")
	objectName = flag.String("object", "", "Object name.")
	b64name    = flag.Bool("b64", false, "Base64-encode the object name.")
	verbose    = flag.Bool("verbose", false, "Print progress every 10 seconds.")
)

type infoWriter struct {
	wc io.WriteCloser
	n  int
	c  *counter
}

func (iw *infoWriter) Write(p []byte) (int, error) {
	n, err := iw.wc.Write(p)
	iw.c.add(time.Now(), n)
	iw.n += n
	return n, err
}

func (iw *infoWriter) Close() error {
	return iw.wc.Close()
}

func (iw *infoWriter) status() string {
	sent := size(iw.n)
	speed := size(iw.c.perSecond(time.Now()))
	return fmt.Sprintf("wrote %s (%s/s)", sent, speed)
}

func main() {
	flag.Parse()
	ctx := context.Background()
	client, err := storageClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if *bucketName == "" {
		log.Fatal("bucket name cannot be empty")
	}
	bucket := client.Bucket(*bucketName)

	if *objectName == "" {
		log.Fatal("object name cannot be empty")
	}
	if *b64name {
		*objectName = base64.StdEncoding.EncodeToString([]byte(*objectName))
	}
	obj := bucket.Object(*objectName)

	w := &infoWriter{
		wc: obj.NewWriter(ctx),
		c: &counter{
			res:  time.Second,
			vals: make([]int, 90),
		},
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

func storageClient(ctx context.Context) (*storage.Client, error) {
	if *auth == "" {
		return nil, fmt.Errorf("no auth credentials supplied")
	}
	jsonKey, err := ioutil.ReadFile(*auth)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(jsonKey, storage.ScopeReadWrite)
	if err != nil {
		return nil, err
	}
	return storage.NewClient(ctx, cloud.WithTokenSource(conf.TokenSource(ctx)))
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

type counter struct {
	mu    sync.Mutex
	vals  []int
	res   time.Duration
	prev  time.Time
	start time.Time
	once  sync.Once
}

// span returns the amount of time for which the counter is valid.  It's
// basically min(now - start, total size).
func (c *counter) span(now time.Time) time.Duration {
	total := c.res * time.Duration(len(c.vals))
	elapsed := now.Sub(c.start)
	if elapsed < total {
		return elapsed
	}
	return total
}

func (c *counter) perSecond(now time.Time) float64 {
	count := float64(c.get(now))
	s := float64(c.span(now))

	return count * float64(time.Second) / s
}

func (c *counter) bucket(t time.Time) int {
	return int(t.UnixNano()/int64(c.res)) % len(c.vals)
}

// sweep marks any buckets zero if they have not been updated since the last
// update.  c.mu must be held by the caller.
func (c *counter) sweep(now time.Time) {
	if c.prev.IsZero() {
		return
	}
	// If every bucket is invalid, mark all zero.
	if now.UnixNano()-c.prev.UnixNano() > int64(c.res)*int64(len(c.vals)) {
		for i := range c.vals {
			c.vals[i] = 0
		}
		return
	}

	prevBucket := int64(c.bucket(c.prev))
	numBuckets := (now.UnixNano() - c.prev.UnixNano()) / int64(c.res)
	for i := prevBucket + 1; i < prevBucket+numBuckets; i++ {
		b := int(i) % len(c.vals)
		c.vals[b] = 0
	}
}

func (c *counter) add(now time.Time, inc int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.once.Do(func() {
		c.start = now
	})
	c.sweep(now)

	bucket := c.bucket(now)
	if now.UnixNano()/int64(c.res) != c.prev.UnixNano()/int64(c.res) {
		c.vals[bucket] = 0
	}
	c.vals[bucket] += inc
	c.prev = now
}

func (c *counter) get(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweep(now)

	var i int
	for _, v := range c.vals {
		i += v
	}
	return i
}
