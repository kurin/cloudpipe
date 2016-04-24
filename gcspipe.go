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
	wc    io.WriteCloser
	n     int
	start time.Time
	end   time.Time
	once  sync.Once
}

func (iw *infoWriter) Write(p []byte) (int, error) {
	iw.once.Do(func() {
		iw.start = time.Now()
	})
	n, err := iw.wc.Write(p)
	iw.n += n
	return n, err
}

func (iw *infoWriter) Close() error {
	iw.end = time.Now()
	return iw.wc.Close()
}

func (iw *infoWriter) status() string {
	d := time.Now().Sub(iw.start)
	return fmt.Sprintf("wrote %d bytes in %d seconds", iw.n, int(d.Seconds()))
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
