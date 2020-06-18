package imageprogress

import (
	"fmt"
	"io"
)

// NewPushWriter creates a writer that periodically reports
// on push progress of a Docker image. It only reports when the state of the
// different layers has changed and uses time thresholds to limit the
// rate of the reports.
func NewPushWriter(printFn func(string)) io.WriteCloser {
	return newWriter(pushReporter(printFn), pushLayersChanged)
}

func pushReporter(printFn func(string)) func(report) {
	return func(r report) {
		var pctComplete float32 = 0.0
		pctComplete += float32(r.count(statusComplete)) / float32(r.totalCount())
		pctComplete += float32(r.count(statusPushing)) / float32(r.totalCount()) * r.percentProgress(statusPushing) / 100.0
		pctComplete *= 100.0

		printFn(fmt.Sprintf("Pushed %d/%d layers, %.0f%% complete", r.count(statusComplete), r.totalCount(), pctComplete))
	}
}

func pushLayersChanged(older, newer report) bool {
	return older.count(statusComplete) != newer.count(statusComplete)
}
