package manifests

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/storage/pkg/reexec"
	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	expectedInstance = digest.Digest("sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b")
	ociFixture       = "testdata/fedora.index.json"
	dockerFixture    = "testdata/fedora.list.json"
)

var (
	_ List = &list{}
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	list := Create()
	if list == nil {
		t.Fatalf("error creating an empty list")
	}
}

func TestFromBlob(t *testing.T) {
	for _, version := range []string{
		ociFixture,
		dockerFixture,
	} {
		bytes, err := ioutil.ReadFile(version)
		if err != nil {
			t.Fatalf("error loading %s: %v", version, err)
		}
		list, err := FromBlob(bytes)
		if err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		if len(list.Docker().Manifests) != len(list.OCIv1().Manifests) {
			t.Fatalf("%s: expected the same number of manifests, but %d != %d", version, len(list.Docker().Manifests), len(list.OCIv1().Manifests))
		}
		for i := range list.Docker().Manifests {
			d := list.Docker().Manifests[i]
			o := list.OCIv1().Manifests[i]
			if d.Platform.OS != o.Platform.OS {
				t.Fatalf("%s: expected the same OS", version)
			}
			if d.Platform.Architecture != o.Platform.Architecture {
				t.Fatalf("%s: expected the same Architecture", version)
			}
		}
	}
}

func TestAddInstance(t *testing.T) {
	manifestBytes, err := ioutil.ReadFile("testdata/fedora-minimal.schema2.json")
	if err != nil {
		t.Fatalf("error loading testdata/fedora-minimal.schema2.json: %v", err)
	}
	manifestType := manifest.GuessMIMEType(manifestBytes)
	manifestDigest, err := manifest.Digest(manifestBytes)
	if err != nil {
		t.Fatalf("error digesting testdata/fedora-minimal.schema2.json: %v", err)
	}
	for _, version := range []string{
		ociFixture,
		dockerFixture,
	} {
		bytes, err := ioutil.ReadFile(version)
		if err != nil {
			t.Fatalf("error loading %s: %v", version, err)
		}
		list, err := FromBlob(bytes)
		if err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		if err = list.AddInstance(manifestDigest, int64(len(manifestBytes)), manifestType, "linux", "amd64", "", nil, "", nil, nil); err != nil {
			t.Fatalf("adding an instance failed in %s: %v", version, err)
		}
		if d, err := list.findDocker(manifestDigest); d == nil || err != nil {
			t.Fatalf("adding an instance failed in %s: %v", version, err)
		}
		if o, err := list.findOCIv1(manifestDigest); o == nil || err != nil {
			t.Fatalf("adding an instance failed in %s: %v", version, err)
		}
	}
}

func TestRemove(t *testing.T) {
	bytes, err := ioutil.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	before := len(list.OCIv1().Manifests)
	instanceDigest := expectedInstance
	if d, err := list.findDocker(instanceDigest); d == nil || err != nil {
		t.Fatalf("finding expected instance failed: %v", err)
	}
	if o, err := list.findOCIv1(instanceDigest); o == nil || err != nil {
		t.Fatalf("finding expected instance failed: %v", err)
	}
	err = list.Remove(instanceDigest)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	after := len(list.Docker().Manifests)
	if after != before-1 {
		t.Fatalf("removing instance should have succeeded")
	}
	if d, err := list.findDocker(instanceDigest); d != nil || err == nil {
		t.Fatalf("finding instance should have failed")
	}
	if o, err := list.findOCIv1(instanceDigest); o != nil || err == nil {
		t.Fatalf("finding instance should have failed")
	}
}

func testString(t *testing.T, values []string, set func(List, digest.Digest, string) error, get func(List, digest.Digest) (string, error)) {
	bytes, err := ioutil.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	for _, testString := range values {
		if err = set(list, expectedInstance, testString); err != nil {
			t.Fatalf("error setting %q: %v", testString, err)
		}
		b, err := list.Serialize("")
		if err != nil {
			t.Fatalf("error serializing list: %v", err)
		}
		list, err := FromBlob(b)
		if err != nil {
			t.Fatalf("error parsing list: %v", err)
		}
		value, err := get(list, expectedInstance)
		if err != nil {
			t.Fatalf("error retrieving value %q: %v", testString, err)
		}
		if value != testString {
			t.Fatalf("expected value %q, got %q: %v", value, testString, err)
		}
	}
}

func testStringSlice(t *testing.T, values [][]string, set func(List, digest.Digest, []string) error, get func(List, digest.Digest) ([]string, error)) {
	bytes, err := ioutil.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	for _, testSlice := range values {
		if err = set(list, expectedInstance, testSlice); err != nil {
			t.Fatalf("error setting %v: %v", testSlice, err)
		}
		b, err := list.Serialize("")
		if err != nil {
			t.Fatalf("error serializing list: %v", err)
		}
		list, err := FromBlob(b)
		if err != nil {
			t.Fatalf("error parsing list: %v", err)
		}
		values, err := get(list, expectedInstance)
		if err != nil {
			t.Fatalf("error retrieving value %v: %v", testSlice, err)
		}
		if !reflect.DeepEqual(values, testSlice) {
			t.Fatalf("expected values %v, got %v: %v", testSlice, values, err)
		}
	}
}

func testMap(t *testing.T, values []map[string]string, set func(List, *digest.Digest, map[string]string) error, get func(List, *digest.Digest) (map[string]string, error)) {
	bytes, err := ioutil.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	instance := expectedInstance
	for _, instanceDigest := range []*digest.Digest{nil, &instance} {
		for _, testMap := range values {
			if err = set(list, instanceDigest, testMap); err != nil {
				t.Fatalf("error setting %v: %v", testMap, err)
			}
			b, err := list.Serialize("")
			if err != nil {
				t.Fatalf("error serializing list: %v", err)
			}
			list, err := FromBlob(b)
			if err != nil {
				t.Fatalf("error parsing list: %v", err)
			}
			values, err := get(list, instanceDigest)
			if err != nil {
				t.Fatalf("error retrieving value %v: %v", testMap, err)
			}
			if len(values) != len(testMap) {
				t.Fatalf("expected %d map entries, got %d", len(testMap), len(values))
			}
			for k, v := range testMap {
				if values[k] != v {
					t.Fatalf("expected map value %q=%q, got %q", k, v, values[k])
				}
			}
		}
	}
}

func TestAnnotations(t *testing.T) {
	testMap(t,
		[]map[string]string{{"A": "B", "C": "D"}, {"E": "F", "G": "H"}},
		func(l List, i *digest.Digest, m map[string]string) error {
			return l.SetAnnotations(i, m)
		},
		func(l List, i *digest.Digest) (map[string]string, error) {
			return l.Annotations(i)
		},
	)
}

func TestArchitecture(t *testing.T) {
	testString(t,
		[]string{"abacus", "sliderule"},
		func(l List, i digest.Digest, s string) error {
			return l.SetArchitecture(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.Architecture(i)
		},
	)
}

func TestFeatures(t *testing.T) {
	testStringSlice(t,
		[][]string{{"chrome", "hubcaps"}, {"climate", "control"}},
		func(l List, i digest.Digest, s []string) error {
			return l.SetFeatures(i, s)
		},
		func(l List, i digest.Digest) ([]string, error) {
			return l.Features(i)
		},
	)
}

func TestOS(t *testing.T) {
	testString(t,
		[]string{"linux", "darwin"},
		func(l List, i digest.Digest, s string) error {
			return l.SetOS(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.OS(i)
		},
	)
}

func TestOSFeatures(t *testing.T) {
	testStringSlice(t,
		[][]string{{"ipv6", "containers"}, {"nested", "virtualization"}},
		func(l List, i digest.Digest, s []string) error {
			return l.SetOSFeatures(i, s)
		},
		func(l List, i digest.Digest) ([]string, error) {
			return l.OSFeatures(i)
		},
	)
}

func TestOSVersion(t *testing.T) {
	testString(t,
		[]string{"el7", "el8"},
		func(l List, i digest.Digest, s string) error {
			return l.SetOSVersion(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.OSVersion(i)
		},
	)
}

func TestURLs(t *testing.T) {
	testStringSlice(t,
		[][]string{{"https://example.com", "https://example.net"}, {"http://example.com", "http://example.net"}},
		func(l List, i digest.Digest, s []string) error {
			return l.SetURLs(i, s)
		},
		func(l List, i digest.Digest) ([]string, error) {
			return l.URLs(i)
		},
	)
}

func TestVariant(t *testing.T) {
	testString(t,
		[]string{"workstation", "cloud", "server"},
		func(l List, i digest.Digest, s string) error {
			return l.SetVariant(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.Variant(i)
		},
	)
}

func TestSerialize(t *testing.T) {
	for _, version := range []string{
		ociFixture,
		dockerFixture,
	} {
		bytes, err := ioutil.ReadFile(version)
		if err != nil {
			t.Fatalf("error loading %s: %v", version, err)
		}
		list, err := FromBlob(bytes)
		if err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		for _, mimeType := range []string{"", v1.MediaTypeImageIndex, manifest.DockerV2ListMediaType} {
			b, err := list.Serialize(mimeType)
			if err != nil {
				t.Fatalf("error serializing %s with type %q: %v", version, mimeType, err)
			}
			l, err := FromBlob(b)
			if err != nil {
				t.Fatalf("error parsing %s re-encoded as %q: %v\n%s", version, mimeType, err, string(b))
			}
			if !reflect.DeepEqual(list.Docker().Manifests, l.Docker().Manifests) {
				t.Fatalf("re-encoded %s as %q was different\n%#v\n%#v", version, mimeType, list, l)
			}
			for i := range list.OCIv1().Manifests {
				manifest := list.OCIv1().Manifests[i]
				m := l.OCIv1().Manifests[i]
				if manifest.Digest != m.Digest ||
					manifest.MediaType != m.MediaType ||
					manifest.Size != m.Size ||
					!reflect.DeepEqual(list.OCIv1().Manifests[i].Platform, l.OCIv1().Manifests[i].Platform) {
					t.Fatalf("re-encoded %s OCI %d as %q was different\n%#v\n%#v", version, i, mimeType, list, l)
				}
			}
		}
	}
}
