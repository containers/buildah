package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func sendThatFile(basepath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Join(basepath, filepath.Clean(string([]rune{filepath.Separator})+filepath.FromSlash(r.URL.Path)))
		f, err := os.Open(filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		finfo, err := f.Stat()
		if err != nil {
			http.Error(w, fmt.Sprintf("checking file info: %v", err), http.StatusInternalServerError)
			return
		}
		content := io.ReadSeeker(f)
		if finfo.IsDir() {
			names, err := f.ReadDir(-1)
			if err != nil {
				http.Error(w, fmt.Sprintf("reading directory: %v", err), http.StatusInternalServerError)
			}
			var builder strings.Builder
			builder.WriteString("<body><html>")
			for _, name := range names {
				suffix := ""
				if name.IsDir() {
					suffix = "/"
				}
				builder.WriteString(fmt.Sprintf("<a href=%q>%s</a><br/>\n", path.Join(r.URL.Path, name.Name())+suffix, html.EscapeString(fs.FormatDirEntry(name))))
			}
			builder.WriteString("</html></body>")
			content = strings.NewReader(builder.String())
		}
		http.ServeContent(w, r, filename, finfo.ModTime(), content)
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
