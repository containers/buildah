package imagebuildah

import (
	"flag"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	var logLevel string
	debug := false
	if InitReexec() {
		return
	}
	flag.BoolVar(&debug, "debug", false, "turn on debug logging")
	flag.StringVar(&logLevel, "log-level", "error", "log level")
	flag.Parse()
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatalf("error parsing log level %q: %v", logLevel, err)
	}
	if debug && level < logrus.DebugLevel {
		level = logrus.DebugLevel
	}
	logrus.SetLevel(level)
	os.Exit(m.Run())
}
