package imageprogress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/containers/storage/pkg/regexp"
)

const (
	defaultProgressTimeThreshhold = 30 * time.Second
	defaultStableThreshhold       = 10
)

// progressLine is a structure representation of a Docker pull progress line
type progressLine struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Detail *progressDetail `json:"progressDetail"`
	Error  string          `json:"error"`
}

// progressDetail is the progressDetail structure in a Docker pull progress line
type progressDetail struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// layerDetail is layer information associated with a specific layerStatus
type layerDetail struct {
	Count   int
	Current int64
	Total   int64
}

// layerStatus is one of different possible status for layers detected by
// the ProgressWriter
type layerStatus int

const (
	statusPending layerStatus = iota
	statusDownloading
	statusExtracting
	statusComplete
	statusPushing
)

// layerStatusFromDockerString translates a string in a Docker status
// line to a layerStatus
func layerStatusFromDockerString(dockerStatus string) layerStatus {
	switch dockerStatus {
	case "Pushing":
		return statusPushing
	case "Downloading":
		return statusDownloading
	case "Extracting", "Verifying Checksum", "Download complete":
		return statusExtracting
	case "Pull complete", "Already exists", "Pushed", "Layer already exists":
		return statusComplete
	default:
		return statusPending
	}
}

type report map[layerStatus]*layerDetail

func (r report) count(status layerStatus) int {
	detail, ok := r[status]
	if !ok {
		return 0
	}
	return detail.Count
}

func (r report) percentProgress(status layerStatus) float32 {
	detail, ok := r[status]
	if !ok {
		return 0
	}
	if detail.Total == 0 {
		return 0
	}
	pct := float32(detail.Current) / float32(detail.Total) * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	if pct < 0.0 {
		pct = 0.0
	}
	return pct
}

func (r report) totalCount() int {
	cnt := 0
	for _, detail := range r {
		cnt += detail.Count
	}
	return cnt
}

// String is used for test output
func (r report) String() string {
	result := &bytes.Buffer{}
	fmt.Fprintf(result, "{")
	for k := range r {
		var status string
		switch k {
		case statusPending:
			status = "pending"
		case statusDownloading:
			status = "downloading"
		case statusExtracting:
			status = "extracting"
		case statusComplete:
			status = "complete"
		}
		fmt.Fprintf(result, "%s:{Count: %d, Current: %d, Total: %d}, ", status, r[k].Count, r[k].Current, r[k].Total)
	}
	fmt.Fprintf(result, "}")
	return result.String()
}

// newWriter creates a writer that periodically reports
// on pull/push progress of a Docker image. It only reports when the state of the
// different layers has changed and uses time thresholds to limit the
// rate of the reports.
func newWriter(reportFn func(report), layersChangedFn func(report, report) bool) io.WriteCloser {
	writer := &imageProgressWriter{
		mutex:                  &sync.Mutex{},
		layerStatus:            map[string]*progressLine{},
		reportFn:               reportFn,
		layersChangedFn:        layersChangedFn,
		progressTimeThreshhold: defaultProgressTimeThreshhold,
		stableThreshhold:       defaultStableThreshhold,
	}
	return writer
}

type imageProgressWriter struct {
	mutex                  *sync.Mutex
	internalWriter         io.WriteCloser
	readingGoroutineStatus <-chan error // Exists if internalWriter != nil
	layerStatus            map[string]*progressLine
	lastLayerCount         int
	stableLines            int
	stableThreshhold       int
	progressTimeThreshhold time.Duration
	lastReport             report
	lastReportTime         time.Time
	reportFn               func(report)
	layersChangedFn        func(report, report) bool
}

func (w *imageProgressWriter) Write(data []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.internalWriter == nil {
		var pipeIn *io.PipeReader
		statusChannel := make(chan error, 1) // Buffered, so that sending a value after this or our caller has failed and exited does not block.
		pipeIn, w.internalWriter = io.Pipe()
		w.readingGoroutineStatus = statusChannel
		go w.readingGoroutine(statusChannel, pipeIn)
	}
	return w.internalWriter.Write(data)
}

func (w *imageProgressWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.internalWriter == nil {
		return nil
	}

	err := w.internalWriter.Close() // As of Go 1.9 and 1.10, PipeWriter.Close() always returns nil
	readingErr := <-w.readingGoroutineStatus
	if err == nil && readingErr != nil {
		err = readingErr
	}
	return err
}

func (w *imageProgressWriter) readingGoroutine(statusChannel chan<- error, pipeIn *io.PipeReader) {
	err := errors.New("Internal error: unexpected panic in imageProgressWriter.readingGoroutine")
	defer func() { statusChannel <- err }()
	defer func() {
		if err != nil {
			pipeIn.CloseWithError(err)
		} else {
			pipeIn.Close()
		}
	}()

	err = w.readProgress(pipeIn)
	// err is nil on reaching EOF
}

func (w *imageProgressWriter) readProgress(pipeIn *io.PipeReader) error {
	decoder := json.NewDecoder(pipeIn)
	for {
		line := &progressLine{}
		err := decoder.Decode(line)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = w.processLine(line)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *imageProgressWriter) processLine(line *progressLine) error {

	if err := getError(line); err != nil {
		return err
	}

	// determine if it's a line we want to process
	if !islayerStatus(line) {
		return nil
	}

	w.layerStatus[line.ID] = line

	// if the number of layers has not stabilized yet, return and wait for more
	// progress
	if !w.isStableLayerCount() {
		return nil
	}

	r := createReport(w.layerStatus)

	// check if the count of layers in each state has changed
	if w.layersChangedFn(w.lastReport, r) {
		w.lastReport = r
		w.lastReportTime = time.Now()
		w.reportFn(r)
		return nil
	}
	// If layer counts haven't changed, but enough time has passed (30 sec by default),
	// at least report on download/push progress
	if time.Since(w.lastReportTime) > w.progressTimeThreshhold {
		w.lastReport = r
		w.lastReportTime = time.Now()
		w.reportFn(r)
	}
	return nil
}

func (w *imageProgressWriter) isStableLayerCount() bool {
	// If the number of layers has changed since last status, we're not stable
	if w.lastLayerCount != len(w.layerStatus) {
		w.lastLayerCount = len(w.layerStatus)
		w.stableLines = 0
		return false
	}
	// Only proceed after we've received status for the same number
	// of layers at least stableThreshhold times. If not, they're still increasing
	w.stableLines++
	return w.stableLines >= w.stableThreshhold
}

var layerIDRegexp = regexp.Delayed("^[a-f0-9]*$")

func islayerStatus(line *progressLine) bool {
	// ignore status lines with no layer id
	if len(line.ID) == 0 {
		return false
	}
	// ignore layer ids that are not hex string
	if !layerIDRegexp.MatchString(line.ID) {
		return false
	}
	// ignore retrying status
	if strings.HasPrefix(line.Status, "Retrying") {
		return false
	}
	return true
}

func getError(line *progressLine) error {
	if len(line.Error) > 0 {
		return errors.New(line.Error)
	}
	return nil
}

func createReport(dockerProgress map[string]*progressLine) report {
	r := report{}
	for _, line := range dockerProgress {
		layerStatus := layerStatusFromDockerString(line.Status)
		detail, exists := r[layerStatus]
		if !exists {
			detail = &layerDetail{}
			r[layerStatus] = detail
		}
		detail.Count++
		if line.Detail != nil {
			detail.Current += line.Detail.Current
			detail.Total += line.Detail.Total
		}
	}
	return r
}
