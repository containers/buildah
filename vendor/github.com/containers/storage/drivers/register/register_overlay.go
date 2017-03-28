// +build !exclude_graphdriver_overlay,linux

package register

import (
	// register the overlay2 graphdriver
	_ "github.com/containers/storage/drivers/overlay2"
)
