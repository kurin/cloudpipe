package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
)

var (
	auth       = flag.String("auth", "", "Path to JSON keyfile.")
	bucketName = flag.String("bucket", "", "Bucket name.")
	objectName = flag.String("object", "", "Object name.")
)

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
	obj := bucket.Object(*objectName)

	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, os.Stdin); err != nil {
		log.Fatal(err)
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
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
