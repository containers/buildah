package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func sendThatFile(basepath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Join(basepath, filepath.Clean(string([]rune{filepath.Separator})+r.URL.Path))
		f, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "whoops", http.StatusInternalServerError)
			return
		}
		finfo, err := f.Stat()
		if err != nil {
			http.Error(w, "whoops", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, filename, finfo.ModTime(), f)
	}
}

func main() {
	args := os.Args
	if len(args) < 3 {
		log.Fatal("requires listening port and subdirectory path")
	}
	port := args[1]
	basedir := args[2]
	http.HandleFunc("/", sendThatFile(basedir))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
