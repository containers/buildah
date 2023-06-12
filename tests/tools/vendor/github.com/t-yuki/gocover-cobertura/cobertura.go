package main

import (
	"encoding/xml"
)

type Coverage struct {
	XMLName         xml.Name   `xml:"coverage"`
	LineRate        float32    `xml:"line-rate,attr"`
	BranchRate      float32    `xml:"branch-rate,attr"`
	Version         string     `xml:"version,attr"`
	Timestamp       int64      `xml:"timestamp,attr"`
	LinesCovered    int64      `xml:"lines-covered,attr"`
	LinesValid      int64      `xml:"lines-valid,attr"`
	BranchesCovered int64      `xml:"branches-covered,attr"`
	BranchesValid   int64      `xml:"branches-valid,attr"`
	Complexity      float32    `xml:"complexity,attr"`
	Sources         []*Source  `xml:"sources>source"`
	Packages        []*Package `xml:"packages>package"`
}

type Source struct {
	Path string `xml:",chardata"`
}

type Package struct {
	Name       string   `xml:"name,attr"`
	LineRate   float32  `xml:"line-rate,attr"`
	BranchRate float32  `xml:"branch-rate,attr"`
	Complexity float32  `xml:"complexity,attr"`
	Classes    []*Class `xml:"classes>class"`
}

type Class struct {
	Name       string    `xml:"name,attr"`
	Filename   string    `xml:"filename,attr"`
	LineRate   float32   `xml:"line-rate,attr"`
	BranchRate float32   `xml:"branch-rate,attr"`
	Complexity float32   `xml:"complexity,attr"`
	Methods    []*Method `xml:"methods>method"`
	Lines      Lines     `xml:"lines>line"`
}

type Method struct {
	Name       string  `xml:"name,attr"`
	Signature  string  `xml:"signature,attr"`
	LineRate   float32 `xml:"line-rate,attr"`
	BranchRate float32 `xml:"branch-rate,attr"`
	Complexity float32 `xml:"complexity,attr"`
	Lines      Lines   `xml:"lines>line"`
}

type Line struct {
	Number int   `xml:"number,attr"`
	Hits   int64 `xml:"hits,attr"`
}

// Lines is a slice of Line pointers, with some convenience methods
type Lines []*Line

// HitRate returns a float32 from 0.0 to 1.0 representing what fraction of lines
// have hits
func (lines Lines) HitRate() (hitRate float32) {
	return float32(lines.NumLinesWithHits()) / float32(len(lines))
}

// NumLines returns the number of lines
func (lines Lines) NumLines() int64 {
	return int64(len(lines))
}

// NumLinesWithHits returns the number of lines with a hit count > 0
func (lines Lines) NumLinesWithHits() (numLinesWithHits int64) {
	for _, line := range lines {
		if line.Hits > 0 {
			numLinesWithHits++
		}
	}
	return numLinesWithHits
}

// AddOrUpdateLine adds a line if it is a different line than the last line recorded.
// If it's the same line as the last line recorded then we update the hits down
// if the new hits is less; otherwise just leave it as-is
func (lines *Lines) AddOrUpdateLine(lineNumber int, hits int64) {
	if len(*lines) > 0 {
		lastLine := (*lines)[len(*lines)-1]
		if lineNumber == lastLine.Number {
			if hits < lastLine.Hits {
				lastLine.Hits = hits
			}
			return
		}
	}
	*lines = append(*lines, &Line{Number: lineNumber, Hits: hits})
}

// HitRate returns a float32 from 0.0 to 1.0 representing what fraction of lines
// have hits
func (method Method) HitRate() float32 {
	return method.Lines.HitRate()
}

// NumLines returns the number of lines
func (method Method) NumLines() int64 {
	return method.Lines.NumLines()
}

// NumLinesWithHits returns the number of lines with a hit count > 0
func (method Method) NumLinesWithHits() int64 {
	return method.Lines.NumLinesWithHits()
}

// HitRate returns a float32 from 0.0 to 1.0 representing what fraction of lines
// have hits
func (class Class) HitRate() float32 {
	return float32(class.NumLinesWithHits()) / float32(class.NumLines())
}

// NumLines returns the number of lines
func (class Class) NumLines() (numLines int64) {
	for _, method := range class.Methods {
		numLines += method.NumLines()
	}
	return numLines
}

// NumLinesWithHits returns the number of lines with a hit count > 0
func (class Class) NumLinesWithHits() (numLinesWithHits int64) {
	for _, method := range class.Methods {
		numLinesWithHits += method.NumLinesWithHits()
	}
	return numLinesWithHits
}

// HitRate returns a float32 from 0.0 to 1.0 representing what fraction of lines
// have hits
func (pkg Package) HitRate() float32 {
	return float32(pkg.NumLinesWithHits()) / float32(pkg.NumLines())
}

// NumLines returns the number of lines
func (pkg Package) NumLines() (numLines int64) {
	for _, class := range pkg.Classes {
		numLines += class.NumLines()
	}
	return numLines
}

// NumLinesWithHits returns the number of lines with a hit count > 0
func (pkg Package) NumLinesWithHits() (numLinesWithHits int64) {
	for _, class := range pkg.Classes {
		numLinesWithHits += class.NumLinesWithHits()
	}
	return numLinesWithHits
}

// HitRate returns a float32 from 0.0 to 1.0 representing what fraction of lines
// have hits
func (cov Coverage) HitRate() float32 {
	return float32(cov.NumLinesWithHits()) / float32(cov.NumLines())
}

// NumLines returns the number of lines
func (cov Coverage) NumLines() (numLines int64) {
	for _, pkg := range cov.Packages {
		numLines += pkg.NumLines()
	}
	return numLines
}

// NumLinesWithHits returns the number of lines with a hit count > 0
func (cov Coverage) NumLinesWithHits() (numLinesWithHits int64) {
	for _, pkg := range cov.Packages {
		numLinesWithHits += pkg.NumLinesWithHits()
	}
	return numLinesWithHits
}
