package main

import (
	"context"
	"errors"
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
			if errors.Is(err, os.ErrNotExist) {
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
		log.Fatal("requires subdirectory path [and optional port [and optional port file name [and optional TLS cert and key file names [and optional pid file name]]]]")
	}
	basedir := args[1]
	port := "0"
	if len(args) > 2 {
		port = args[2]
	}
	certs, key := "", ""
	if len(args) > 5 && args[4] != "" && args[5] != "" {
		certs = args[4]
		key = args[5]
	}
	if len(args) > 6 && args[6] != "" {
		err := os.WriteFile(args[6], []byte(strconv.Itoa(os.Getpid())), 0o644)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
	http.HandleFunc("/", sendThatFile(basedir))
	server := http.Server{
		Addr: ":" + port,
		BaseContext: func(l net.Listener) context.Context {
			if tcp, ok := l.Addr().(*net.TCPAddr); ok {
				if len(args) > 3 {
					f, err := os.CreateTemp(filepath.Dir(args[3]), filepath.Base(args[3]))
					if err != nil {
						log.Fatalf("%v", err)
					}
					tempName := f.Name()
					port := strconv.Itoa(tcp.Port)
					if n, err := f.WriteString(port); err != nil || n != len(port) {
						if err != nil {
							log.Fatalf("%v", err)
						}
						log.Fatalf("short write: %d != %d", n, len(port))
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
	if certs != "" && key != "" {
		log.Fatal(server.ListenAndServeTLS(certs, key))
	}
	log.Fatal(server.ListenAndServe())
}
