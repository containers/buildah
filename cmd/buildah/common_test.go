package main

import (
	"flag"
	"os"
	"os/user"
	"testing"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
	"github.com/projectatomic/buildah"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	signaturePolicyPath = ""
	storeOptions        = storage.DefaultStoreOptions
)

func TestMain(m *testing.M) {
	flag.StringVar(&signaturePolicyPath, "signature-policy", "", "pathname of signature policy file (not usually used)")
	options := storage.StoreOptions{}
	debug := false
	flag.StringVar(&options.GraphRoot, "root", "", "storage root dir")
	flag.StringVar(&options.RunRoot, "runroot", "", "storage state dir")
	flag.StringVar(&options.GraphDriverName, "storage-driver", "", "storage driver")
	flag.BoolVar(&debug, "debug", false, "turn on debug logging")
	flag.Parse()
	if options.GraphRoot != "" || options.RunRoot != "" || options.GraphDriverName != "" {
		storeOptions = options
	}
	if buildah.InitReexec() {
		return
	}
	logrus.SetLevel(logrus.ErrorLevel)
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	os.Exit(m.Run())
}

func TestGetStore(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	set := flag.NewFlagSet("test", 0)
	globalSet := flag.NewFlagSet("test", 0)
	globalSet.String("root", "", "path to the directory in which data, including images, is stored")
	globalSet.String("runroot", "", "path to the directory in which state is stored")
	globalSet.String("storage-driver", "", "storage driver")
	globalCtx := cli.NewContext(nil, globalSet, nil)
	globalCtx.GlobalSet("root", storeOptions.GraphRoot)
	globalCtx.GlobalSet("runroot", storeOptions.RunRoot)
	globalCtx.GlobalSet("storage-driver", storeOptions.GraphDriverName)
	command := cli.Command{Name: "TestGetStore"}
	c := cli.NewContext(nil, set, globalCtx)
	c.Command = command

	_, err := getStore(c)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSize(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	_, _, _, err = getDateAndDigestAndSize(images[0], store)
	if err != nil {
		t.Error(err)
	}
}

func failTestIfNotRoot(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Log("Could not determine user.  Running without root may cause tests to fail")
	} else if u.Uid != "0" {
		t.Fatal("tests will fail unless run as root")
	}
}

func pullTestImage(t *testing.T, imageName string) (string, error) {
	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	}
	options := buildah.BuilderOptions{
		FromImage:           imageName,
		SignaturePolicyPath: signaturePolicyPath,
	}

	b, err := buildah.NewBuilder(store, options)
	if err != nil {
		t.Fatal(err)
	}
	id := b.FromImageID
	err = b.Delete()
	if err != nil {
		t.Fatal(err)
	}
	return id, nil
}
