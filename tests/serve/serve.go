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
	"net/http/cgi"
	"os"
	"os/exec"
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
				fmt.Fprintf(&builder, "<a href=%q>%s</a><br/>\n", path.Join(r.URL.Path, name.Name())+suffix, html.EscapeString(fs.FormatDirEntry(name)))
			}
			fmt.Fprint(&builder, "</html></body>")
			content = strings.NewReader(builder.String())
		}
		http.ServeContent(w, r, filename, finfo.ModTime(), content)
	}
}

func runThatCGI(baseLocation, basePath, handlerAndEnvs string) func(w http.ResponseWriter, r *http.Request) {
	handler, envList, _ := strings.Cut(handlerAndEnvs, ":")
	handlerCmd := strings.Fields(handler) // github.com/mattn/go-shellwords.Parse() would be better, but we want to not require anything outside of the standard library, so this will have to do
	if !filepath.IsAbs(handlerCmd[0]) {
		abs, err := exec.LookPath(handlerCmd[0])
		if err != nil {
			log.Fatalf("finding %q: %v", handler, err)
		}
		handlerCmd[0] = abs
	}
	envSlice := strings.Split(envList, ":")
	var envs, inherits []string
	for _, env := range envSlice {
		if strings.Contains(env, "=") {
			envs = append(envs, env)
		} else {
			inherits = append(inherits, env)
		}
	}
	cgiBackend := cgi.Handler{
		Path:                handlerCmd[0],
		Args:                handlerCmd[1:],
		Root:                baseLocation,
		Dir:                 basePath,
		Env:                 envs,
		InheritEnv:          inherits,
		Stderr:              os.Stderr,
		PathLocationHandler: http.DefaultServeMux,
	}
	log.Printf("cgi: %q -> %q started in %q %+v", baseLocation, handler, basePath, cgiBackend.Env)
	return cgiBackend.ServeHTTP
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatal("requires subdirectory path [and optional port [and optional port file name [and optional TLS cert and key file names [and optional pid file name]]]]")
	}
	spec := args[1]
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
	roots := false
	for specPart := range strings.SplitSeq(spec, ",") {
		if content, subdirectoryAndHandlerAndEnvs, ok := strings.Cut(specPart, "="); ok {
			if subdirectory, handlerAndEnvs, ok := strings.Cut(subdirectoryAndHandlerAndEnvs, ":"); ok {
				http.HandleFunc(content, runThatCGI(content, subdirectory, handlerAndEnvs))
			} else {
				http.HandleFunc(content, sendThatFile(subdirectory))
			}
		} else {
			if roots {
				log.Fatal("only one content root allowed")
			}
			http.HandleFunc("/", sendThatFile(content))
			roots = true
		}
	}
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
