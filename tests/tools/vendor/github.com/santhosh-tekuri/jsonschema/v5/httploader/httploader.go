// Package httploader implements loader.Loader for http/https url.
//
// The package is typically only imported for the side effect of
// registering its Loaders.
//
// To use httploader, link this package into your program:
//
//	import _ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
package httploader

import (
	"fmt"
	"io"
	"net/http"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Client is the default HTTP Client used to Get the resource.
var Client = http.DefaultClient

// Load loads resource from given http(s) url.
func Load(url string) (io.ReadCloser, error) {
	resp, err := Client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

func init() {
	jsonschema.Loaders["http"] = Load
	jsonschema.Loaders["https"] = Load
}
