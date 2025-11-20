package dockerclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type CopyInfo struct {
	os.FileInfo
	Path       string
	Decompress bool // deprecated, is never set and is ignored
	FromDir    bool
}

// CalcCopyInfo identifies the source files selected by a Dockerfile ADD or COPY instruction.
func CalcCopyInfo(origPath, rootPath string, allowWildcards bool) ([]CopyInfo, error) {
	explicitDir := origPath == "." || origPath == "/" || strings.HasSuffix(origPath, "/.") || strings.HasSuffix(origPath, "/")
	// all CopyInfo resulting from this call will have FromDir set to explicitDir
	infos, err := calcCopyInfo(origPath, rootPath, allowWildcards, explicitDir)
	if err != nil {
		return nil, err
	}
	return infos, nil
}

func calcCopyInfo(origPath, rootPath string, allowWildcards, explicitDir bool) ([]CopyInfo, error) {
	origPath = trimLeadingPath(origPath)
	if !filepath.IsAbs(rootPath) {
		rootPath = trimLeadingPath(rootPath)
	}
	// Deal with wildcards
	if allowWildcards && containsWildcards(origPath) {
		matchPath := filepath.Join(rootPath, origPath)
		var copyInfos []CopyInfo
		if err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == "" {
				// Why are we doing this check?
				return nil
			}
			if match, _ := filepath.Match(matchPath, path); !match {
				return nil
			}

			// Note we set allowWildcards to false in case the name has
			// a * in it
			subInfos, err := calcCopyInfo(trimLeadingPath(strings.TrimPrefix(path, rootPath)), rootPath, false, explicitDir)
			if err != nil {
				return err
			}
			copyInfos = append(copyInfos, subInfos...)
			return nil
		}); err != nil {
			return nil, err
		}
		return copyInfos, nil
	}

	// Must be a dir or a file
	fi, err := os.Stat(filepath.Join(rootPath, origPath))
	if err != nil {
		return nil, err
	}

	// flatten the root directory so we can rebase it
	if origPath == "." {
		if !fi.IsDir() {
			// we want to mount a single file as input
			return []CopyInfo{{FileInfo: fi, Path: origPath, FromDir: false}}, nil
		}
		var copyInfos []CopyInfo
		infos, err := ioutil.ReadDir(rootPath)
		if err != nil {
			return nil, err
		}
		for _, info := range infos {
			copyInfos = append(copyInfos, CopyInfo{FileInfo: info, Path: info.Name(), FromDir: explicitDir})
		}
		return copyInfos, nil
	}

	origPath = trimTrailingDot(origPath)
	return []CopyInfo{{FileInfo: fi, Path: origPath, FromDir: explicitDir}}, nil
}

func DownloadURL(src, dst, tempDir string) ([]CopyInfo, string, error) {
	// get filename from URL
	u, err := url.Parse(src)
	if err != nil {
		return nil, "", err
	}
	base := path.Base(u.Path)
	if base == "." {
		return nil, "", fmt.Errorf("cannot determine filename from url: %s", u)
	}

	resp, err := http.Get(src)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("server returned a status code >= 400: %s", resp.Status)
	}

	tmpDir, err := ioutil.TempDir(tempDir, "dockerbuildurl-")
	if err != nil {
		return nil, "", err
	}
	tmpFileName := filepath.Join(tmpDir, base)
	tmpFile, err := os.OpenFile(tmpFileName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", err
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", err
	}
	info, err := os.Stat(tmpFileName)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", err
	}
	return []CopyInfo{{FileInfo: info, Path: base}}, tmpDir, nil
}

func trimLeadingPath(origPath string) string {
	// Work in daemon-specific OS filepath semantics
	origPath = filepath.FromSlash(origPath)
	if origPath != "" && origPath[0] == os.PathSeparator && len(origPath) > 1 {
		origPath = origPath[1:]
	}
	origPath = strings.TrimPrefix(origPath, "."+string(os.PathSeparator))
	return origPath
}

func trimTrailingSlash(origPath string) string {
	if origPath == "/" {
		return origPath
	}
	return strings.TrimSuffix(origPath, "/")
}

func trimTrailingDot(origPath string) string {
	if strings.HasSuffix(origPath, string(os.PathSeparator)+".") {
		return strings.TrimSuffix(origPath, ".")
	}
	return origPath
}

// containsWildcards checks whether the provided name has a wildcard.
func containsWildcards(name string) bool {
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '\\' {
			i++
		} else if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}

// isURL returns true if the string appears to be a URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
