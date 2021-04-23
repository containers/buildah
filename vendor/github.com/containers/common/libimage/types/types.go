package types

import (
	"time"

	"github.com/containers/image/v5/manifest"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageData contains the inspected data of an image.
type ImageData struct {
	ID           string                        `json:"Id"`
	Digest       digest.Digest                 `json:"Digest"`
	RepoTags     []string                      `json:"RepoTags"`
	RepoDigests  []string                      `json:"RepoDigests"`
	Parent       string                        `json:"Parent"`
	Comment      string                        `json:"Comment"`
	Created      *time.Time                    `json:"Created"`
	Config       *ociv1.ImageConfig            `json:"Config"`
	Version      string                        `json:"Version"`
	Author       string                        `json:"Author"`
	Architecture string                        `json:"Architecture"`
	Os           string                        `json:"Os"`
	Size         int64                         `json:"Size"`
	VirtualSize  int64                         `json:"VirtualSize"`
	GraphDriver  *DriverData                   `json:"GraphDriver"`
	RootFS       *RootFS                       `json:"RootFS"`
	Labels       map[string]string             `json:"Labels"`
	Annotations  map[string]string             `json:"Annotations"`
	ManifestType string                        `json:"ManifestType"`
	User         string                        `json:"User"`
	History      []ociv1.History               `json:"History"`
	NamesHistory []string                      `json:"NamesHistory"`
	HealthCheck  *manifest.Schema2HealthConfig `json:"Healthcheck,omitempty"`
}

// DriverData includes data on the storage driver of the image.
type DriverData struct {
	Name string            `json:"Name"`
	Data map[string]string `json:"Data"`
}

// RootFS includes data on the root filesystem of the image.
type RootFS struct {
	Type   string          `json:"Type"`
	Layers []digest.Digest `json:"Layers"`
}

// ImageHistory contains the history information of an image.
type ImageHistory struct {
	ID        string     `json:"id"`
	Created   *time.Time `json:"created"`
	CreatedBy string     `json:"createdBy"`
	Size      int64      `json:"size"`
	Comment   string     `json:"comment"`
	Tags      []string   `json:"tags"`
}
