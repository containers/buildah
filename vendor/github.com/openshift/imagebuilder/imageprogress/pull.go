package imageprogress

import (
	"fmt"
	"io"
)

// NewPullWriter creates a writer that periodically reports
// on pull progress of a Docker image. It only reports when the state of the
// different layers has changed and uses time thresholds to limit the
// rate of the reports.
func NewPullWriter(printFn func(string)) io.WriteCloser {
	return newWriter(pullReporter(printFn), pullLayersChanged)
}

func pullReporter(printFn func(string)) func(report) {
	extracting := false
	return func(r report) {
		if extracting {
			return
		}
		if r.count(statusDownloading) == 0 &&
			r.count(statusPending) == 0 &&
			r.count(statusExtracting) > 0 {

			printFn(fmt.Sprintf("Pulled %[1]d/%[1]d layers, 100%% complete", r.totalCount()))
			printFn("Extracting")
			extracting = true
			return
		}

		completeCount := r.count(statusComplete) + r.count(statusExtracting)
		var pctComplete float32 = 0.0
		pctComplete += float32(completeCount) / float32(r.totalCount())
		pctComplete += float32(r.count(statusDownloading)) / float32(r.totalCount()) * r.percentProgress(statusDownloading) / 100.0
		pctComplete *= 100.0
		printFn(fmt.Sprintf("Pulled %d/%d layers, %.0f%% complete", completeCount, r.totalCount(), pctComplete))
	}
}

func pullLayersChanged(older, newer report) bool {
	olderCompleteCount := older.count(statusComplete) + older.count(statusExtracting)
	newerCompleteCount := newer.count(statusComplete) + newer.count(statusExtracting)
	return olderCompleteCount != newerCompleteCount
}
