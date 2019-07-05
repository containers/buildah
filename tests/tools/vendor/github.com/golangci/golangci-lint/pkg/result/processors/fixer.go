package processors

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golangci/golangci-lint/pkg/fsutils"

	"github.com/golangci/golangci-lint/pkg/logutils"

	"github.com/pkg/errors"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/result"
)

type Fixer struct {
	cfg       *config.Config
	log       logutils.Log
	fileCache *fsutils.FileCache
}

func NewFixer(cfg *config.Config, log logutils.Log, fileCache *fsutils.FileCache) *Fixer {
	return &Fixer{cfg: cfg, log: log, fileCache: fileCache}
}

func (f Fixer) Process(issues <-chan result.Issue) <-chan result.Issue {
	if !f.cfg.Issues.NeedFix {
		return issues
	}

	outCh := make(chan result.Issue, 1024)

	go func() {
		issuesToFixPerFile := map[string][]result.Issue{}
		for issue := range issues {
			if issue.Replacement == nil {
				outCh <- issue
				continue
			}

			issuesToFixPerFile[issue.FilePath()] = append(issuesToFixPerFile[issue.FilePath()], issue)
		}

		for file, issuesToFix := range issuesToFixPerFile {
			if err := f.fixIssuesInFile(file, issuesToFix); err != nil {
				f.log.Errorf("Failed to fix issues in file %s: %s", file, err)

				// show issues only if can't fix them
				for _, issue := range issuesToFix {
					outCh <- issue
				}
			}
		}
		close(outCh)
	}()

	return outCh
}

func (f Fixer) fixIssuesInFile(filePath string, issues []result.Issue) error {
	// TODO: don't read the whole file into memory: read line by line;
	// can't just use bufio.scanner: it has a line length limit
	origFileData, err := f.fileCache.GetFileBytes(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to get file bytes for %s", filePath)
	}
	origFileLines := bytes.Split(origFileData, []byte("\n"))

	tmpFileName := filepath.Join(filepath.Dir(filePath), fmt.Sprintf(".%s.golangci_fix", filepath.Base(filePath)))
	tmpOutFile, err := os.Create(tmpFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to make file %s", tmpFileName)
	}

	// merge multiple issues per line into one issue
	issuesPerLine := map[int][]result.Issue{}
	for _, i := range issues {
		issuesPerLine[i.Line()] = append(issuesPerLine[i.Line()], i)
	}

	issues = issues[:0] // reuse the same memory
	for line, lineIssues := range issuesPerLine {
		if mergedIssue := f.mergeLineIssues(line, lineIssues, origFileLines); mergedIssue != nil {
			issues = append(issues, *mergedIssue)
		}
	}

	issues = f.findNotIntersectingIssues(issues)

	if err = f.writeFixedFile(origFileLines, issues, tmpOutFile); err != nil {
		tmpOutFile.Close()
		os.Remove(tmpOutFile.Name())
		return err
	}

	tmpOutFile.Close()
	if err = os.Rename(tmpOutFile.Name(), filePath); err != nil {
		os.Remove(tmpOutFile.Name())
		return errors.Wrapf(err, "failed to rename %s -> %s", tmpOutFile.Name(), filePath)
	}

	return nil
}

//nolint:gocyclo
func (f Fixer) mergeLineIssues(lineNum int, lineIssues []result.Issue, origFileLines [][]byte) *result.Issue {
	origLine := origFileLines[lineNum-1] // lineNum is 1-based

	if len(lineIssues) == 1 && lineIssues[0].Replacement.Inline == nil {
		return &lineIssues[0]
	}

	// check issues first
	for _, i := range lineIssues {
		if i.LineRange != nil {
			f.log.Infof("Line %d has multiple issues but at least one of them is ranged: %#v", lineNum, lineIssues)
			return &lineIssues[0]
		}

		r := i.Replacement
		if r.Inline == nil || len(r.NewLines) != 0 || r.NeedOnlyDelete {
			f.log.Infof("Line %d has multiple issues but at least one of them isn't inline: %#v", lineNum, lineIssues)
			return &lineIssues[0]
		}

		if r.Inline.StartCol < 0 || r.Inline.Length <= 0 || r.Inline.StartCol+r.Inline.Length > len(origLine) {
			f.log.Warnf("Line %d (%q) has invalid inline fix: %#v, %#v", lineNum, origLine, i, r.Inline)
			return nil
		}
	}

	return f.applyInlineFixes(lineIssues, origLine, lineNum)
}

func (f Fixer) applyInlineFixes(lineIssues []result.Issue, origLine []byte, lineNum int) *result.Issue {
	sort.Slice(lineIssues, func(i, j int) bool {
		return lineIssues[i].Replacement.Inline.StartCol < lineIssues[j].Replacement.Inline.StartCol
	})

	var newLineBuf bytes.Buffer
	newLineBuf.Grow(len(origLine))

	//nolint:misspell
	// example: origLine="it's becouse of them", StartCol=5, Length=7, NewString="because"

	curOrigLinePos := 0
	for _, i := range lineIssues {
		fix := i.Replacement.Inline
		if fix.StartCol < curOrigLinePos {
			f.log.Warnf("Line %d has multiple intersecting issues: %#v", lineNum, lineIssues)
			return nil
		}

		if curOrigLinePos != fix.StartCol {
			newLineBuf.Write(origLine[curOrigLinePos:fix.StartCol])
		}
		newLineBuf.WriteString(fix.NewString)
		curOrigLinePos = fix.StartCol + fix.Length
	}
	if curOrigLinePos != len(origLine) {
		newLineBuf.Write(origLine[curOrigLinePos:])
	}

	mergedIssue := lineIssues[0] // use text from the first issue (it's not really used)
	mergedIssue.Replacement = &result.Replacement{
		NewLines: []string{newLineBuf.String()},
	}
	return &mergedIssue
}

func (f Fixer) findNotIntersectingIssues(issues []result.Issue) []result.Issue {
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j] //nolint:scopelint
		return a.Line() < b.Line()
	})

	var ret []result.Issue
	var currentEnd int
	for _, issue := range issues {
		rng := issue.GetLineRange()
		if rng.From <= currentEnd {
			f.log.Infof("Skip issue %#v: intersects with end %d", issue, currentEnd)
			continue // skip intersecting issue
		}
		f.log.Infof("Fix issue %#v with range %v", issue, issue.GetLineRange())
		ret = append(ret, issue)
		currentEnd = rng.To
	}

	return ret
}

func (f Fixer) writeFixedFile(origFileLines [][]byte, issues []result.Issue, tmpOutFile *os.File) error {
	// issues aren't intersecting

	nextIssueIndex := 0
	for i := 0; i < len(origFileLines); i++ {
		var outLine string
		var nextIssue *result.Issue
		if nextIssueIndex != len(issues) {
			nextIssue = &issues[nextIssueIndex]
		}

		origFileLineNumber := i + 1
		if nextIssue == nil || origFileLineNumber != nextIssue.Line() {
			outLine = string(origFileLines[i])
		} else {
			nextIssueIndex++
			rng := nextIssue.GetLineRange()
			i += rng.To - rng.From
			if nextIssue.Replacement.NeedOnlyDelete {
				continue
			}
			outLine = strings.Join(nextIssue.Replacement.NewLines, "\n")
		}

		if i < len(origFileLines)-1 {
			outLine += "\n"
		}
		if _, err := tmpOutFile.WriteString(outLine); err != nil {
			return errors.Wrap(err, "failed to write output line")
		}
	}

	return nil
}
