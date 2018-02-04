package staticweb

import (
	"fmt"
	"io/ioutil"
	"net/http"
 	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

type Handler struct {
	bucketName string
	pathPrefix string
}

func NewHandler(bucketName string, pathPrefix string) *Handler {
	return &Handler{bucketName, pathPrefix}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	p := r.URL.Path
	if !strings.HasPrefix(p, h.pathPrefix) {
		log.Errorf(ctx, "nope. %s -> %s failed.", p, h.pathPrefix)
		http.NotFound(w, r)
		return
	}
	rel := strings.TrimPrefix(p, h.pathPrefix)
	if rel == "" || strings.HasSuffix(rel, "/") {
		rel += "index.html"
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf(ctx, "failed to create client: %v", err)
		return
	}
	defer client.Close()
	bucket := client.Bucket(h.bucketName)
	obj := bucket.Object(rel)
	attr, err := obj.Attrs(ctx)
	if err != nil {
		log.Errorf(ctx, "obj.Attrs (%s, %s): %v", h.bucketName, rel, err)
		http.NotFound(w, r)
		return
	}

	if attr.ContentType != "" {
		w.Header().Set("Content-Type", attr.ContentType)
	}
	if attr.CacheControl != "" {
		w.Header().Set("Cache-Control", attr.CacheControl)
	}
	w.Header().Set("Last-Modified", attr.Updated.String())

	rdr, err := obj.NewReader(ctx)
	if err != nil {
		log.Errorf(ctx, "obj.NewReader: %v", err)
		http.NotFound(w, r)
		return
	}
	defer rdr.Close()

	slurp, err := ioutil.ReadAll(rdr)
	if err != nil {
		log.Errorf(ctx, "ReadAll: %v", err)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	w.Write(slurp)
}
