package main

import (
	"flag"
	"os"
	"os/user"
	"testing"

	"github.com/containers/buildah"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	signaturePolicyPath = ""
	storeOptions, _     = storage.DefaultStoreOptions()
	testSystemContext   = types.SystemContext{
		SignaturePolicyPath:      "../../tests/policy.json",
		SystemRegistriesConfPath: "../../tests/registries.conf",
	}
)

func TestMain(m *testing.M) {
	flag.StringVar(&signaturePolicyPath, "signature-policy", "", "pathname of signature policy file (not usually used)")
	options := storage.StoreOptions{}
	debug := false
	flag.StringVar(&options.GraphRoot, "root", "", "storage root dir")
	flag.StringVar(&options.RunRoot, "runroot", "", "storage state dir")
	flag.StringVar(&options.GraphDriverName, "storage-driver", "", "storage driver")
	flag.StringVar(&testSystemContext.SystemRegistriesConfPath, "registries-conf", "", "registries list")
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
	testCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := getStore(cmd)
			return err
		},
	}
	flags := testCmd.PersistentFlags()
	flags.String("root", storeOptions.GraphRoot, "")
	flags.String("runroot", storeOptions.RunRoot, "")
	flags.String("storage-driver", storeOptions.GraphDriverName, "")
	flags.String("signature-policy", "", "")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		t.Error(err)
	}
	// The following flags had to be added or we get panics in common.go when
	// the lookups occur
	flags.StringSlice("storage-opt", []string{}, "")
	flags.String("registries-conf", "", "")
	flags.String("userns-uid-map", "", "")
	flags.String("userns-gid-map", "", "")
	err := testCmd.Execute()
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
