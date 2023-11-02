package main

import (
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
)

var shortBase = "http://localhost:8080"

type redirects struct {
	data    map[string]string
	revdata map[string]string
	lastid  int64
}

var db redirects

func init() {
	db.data = map[string]string{}
	db.revdata = map[string]string{}
}

func (r *redirects) get(id string) (string, bool) {
	url, ok := r.data[id]
	if ok {
		log.Printf("Found %s: %s\n", id, url)
	} else {
		log.Println("Not found:", id)
	}
	return url, ok
}
func (r *redirects) create(url string) string {
	if id, ok := r.revdata[url]; ok {
		log.Printf("Already created id %s: %s", id, url)
		return id
	}
	r.lastid++
	id := big.NewInt(r.lastid).Text(62)
	r.data[id] = url
	r.revdata[url] = id
	log.Printf("New URL with id %s: %s\n", id, url)
	return id
}

func apiCreate(res http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		log.Println("Bad Request: POST request with not empty url:", req.URL.Path)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("cannot read request body")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	body := string(b)
	if !isValidUrl(body) {
		log.Println("Bad Request: invalid url:", body)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	id := db.create(body)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	fmt.Fprintf(res, "%s/%s", shortBase, id)
}

func apiGet(res http.ResponseWriter, req *http.Request) {
	id := strings.TrimPrefix(req.URL.Path, "/")
	if !isValidId(id) {
		log.Println("Bad Request: invalid id:", req.URL.Path)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	if url, ok := db.get(id); ok {
		res.Header().Set("Location", url)
		log.Printf("Redirecting %s to %s\n", id, url)
		res.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		log.Println("Bad Request: id not found:", id)
		res.WriteHeader(http.StatusBadRequest)
	}
}

func isValidUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isAlphaNumeric(c rune) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}

func isValidId(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !isAlphaNumeric(c) {
			return false
		}
	}
	return true
}

type MyHandler struct{}

func (h MyHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		apiGet(res, req)
	case http.MethodPost:
		apiCreate(res, req)
	default:
		log.Println("Bad Request: bad method:", req.Method)
		res.WriteHeader(http.StatusBadRequest)
	}
}

func main() {
	log.Println("Starting...")
	var h MyHandler
	err := http.ListenAndServe(`:8080`, h)
	if err != nil {
		panic(err)
	}
}
