package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

const (
	PREFIX = "https://hk.appledaily.com"
)

func handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprint(w, err)
		}
	}()
	path := r.URL.Path
	if path == "/" || regexp.MustCompile("^/video/top/?").MatchString(path) {
		handleList("index", w, r)
	} else if regexp.MustCompile("^/realtime/top/?").MatchString(path) {
		handleList("realtime-top", w, r)
	} else if regexp.MustCompile("^/(realtime|enews|daily|ETW)/?").MatchString(path) {
		handleList("list", w, r)
	} else if regexp.MustCompile("^/\\w+/[0-9]{8}/\\w+/$").MatchString(path) {
		handleArticle(w, r)
	} else if regexp.MustCompile("^/(video)/?").MatchString(path) {
		handleVideo(w, r)
	} else {
		fmt.Fprint(w, "no such page")
	}
}

func main() {
	address := flag.String("listen", "127.0.0.1:8080", "address to listen to")
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Println("listening", *address)
	log.Fatal(http.ListenAndServe(*address, nil))
}
