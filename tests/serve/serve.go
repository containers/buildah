package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	if len(args) < 2 {
		log.Fatal("requires subdirectory path [and optional port [and optional port file name]]")
	}
	basedir := args[1]
	port := "0"
	if len(args) > 2 {
		port = args[2]
	}
	http.HandleFunc("/", sendThatFile(basedir))
	server := http.Server{
		Addr: ":" + port,
		BaseContext: func(l net.Listener) context.Context {
			if tcp, ok := l.Addr().(*net.TCPAddr); ok {
				if len(args) > 3 {
					f, err := ioutil.TempFile(filepath.Dir(args[3]), filepath.Base(args[3]))
					if err != nil {
						log.Fatalf("%v", err)
					}
					tempName := f.Name()
					bytes := []byte(strconv.Itoa(tcp.Port))
					if n, err := f.Write(bytes); err != nil || n != len(bytes) {
						if err != nil {
							log.Fatalf("%v", err)
						}
						log.Fatalf("short write: %d != %d", n, len(bytes))
					}
					f.Close()
					if err := os.Rename(tempName, args[3]); err != nil {
						log.Fatalf("rename: %v", err)
					}
				}
			}
			return context.Background()
		},
	}
	log.Fatal(server.ListenAndServe())
}
