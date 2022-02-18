package gci

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"

	sectionsPkg "github.com/daixiang0/gci/pkg/gci/sections"
	"github.com/daixiang0/gci/pkg/io"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"golang.org/x/sync/errgroup"
)

type SectionList []sectionsPkg.Section

func (list SectionList) String() []string {
	var output []string
	for _, section := range list {
		output = append(output, section.String())
	}
	return output
}

func DefaultSections() SectionList {
	return SectionList{sectionsPkg.StandardPackage{}, sectionsPkg.DefaultSection{nil, nil}}
}

func DefaultSectionSeparators() SectionList {
	return SectionList{sectionsPkg.NewLine{}}
}
func LocalFlagsToSections(localFlags []string) SectionList {
	sections := DefaultSections()
	// Add all local arguments as ImportPrefix sections
	for _, prefix := range localFlags {
		sections = append(sections, sectionsPkg.Prefix{prefix, nil, nil})
	}
	return sections
}

func PrintFormattedFiles(paths []string, cfg GciConfiguration) error {
	return processStdInAndGoFilesInPaths(paths, cfg, func(filePath string, unmodifiedFile, formattedFile []byte) error {
		fmt.Print(string(formattedFile))
		return nil
	})
}

func WriteFormattedFiles(paths []string, cfg GciConfiguration) error {
	return processGoFilesInPaths(paths, cfg, func(filePath string, unmodifiedFile, formattedFile []byte) error {
		if bytes.Equal(unmodifiedFile, formattedFile) {
			log.Printf("Skipping correctly formatted File: %s", filePath)
			return nil
		}
		log.Printf("Writing formatted File: %s", filePath)
		return os.WriteFile(filePath, formattedFile, 0644)
	})
}

func DiffFormattedFiles(paths []string, cfg GciConfiguration) error {
	return processStdInAndGoFilesInPaths(paths, cfg, func(filePath string, unmodifiedFile, formattedFile []byte) error {
		fileURI := span.URIFromPath(filePath)
		edits := myers.ComputeEdits(fileURI, string(unmodifiedFile), string(formattedFile))
		unifiedEdits := gotextdiff.ToUnified(filePath, filePath, string(unmodifiedFile), edits)
		fmt.Printf("%v", unifiedEdits)
		return nil
	})
}

type fileFormattingFunc func(filePath string, unmodifiedFile, formattedFile []byte) error

func processStdInAndGoFilesInPaths(paths []string, cfg GciConfiguration, fileFunc fileFormattingFunc) error {
	return processFiles(io.StdInGenerator.Combine(io.GoFilesInPathsGenerator(paths)), cfg, fileFunc)
}

func processGoFilesInPaths(paths []string, cfg GciConfiguration, fileFunc fileFormattingFunc) error {
	return processFiles(io.GoFilesInPathsGenerator(paths), cfg, fileFunc)
}

func processFiles(fileGenerator io.FileGeneratorFunc, cfg GciConfiguration, fileFunc fileFormattingFunc) error {
	var taskGroup errgroup.Group
	files, err := fileGenerator()
	if err != nil {
		return err
	}
	for _, file := range files {
		// run file processing in parallel
		taskGroup.Go(processingFunc(file, cfg, fileFunc))
	}
	return taskGroup.Wait()
}

func processingFunc(file io.FileObj, cfg GciConfiguration, formattingFunc fileFormattingFunc) func() error {
	return func() error {
		unmodifiedFile, formattedFile, err := LoadFormatGoFile(file, cfg)
		if err != nil {
			if errors.Is(err, FileParsingError{}) {
				// do not process files that are improperly formatted
				return nil
			}
			return err
		}
		return formattingFunc(file.Path(), unmodifiedFile, formattedFile)
	}
}

func LoadFormatGoFile(file io.FileObj, cfg GciConfiguration) (unmodifiedFile, formattedFile []byte, err error) {
	unmodifiedFile, err = file.Load()
	log.Printf("Loaded File: %s", file.Path())
	if err != nil {
		return nil, nil, err
	}

	formattedFile, err = formatGoFile(unmodifiedFile, cfg)
	if err != nil {
		// ignore missing import statements
		if !errors.Is(err, MissingImportStatementError) {
			return unmodifiedFile, nil, err
		}
		log.Printf("File does not contain an import statement: %s", file.Path())
		formattedFile = unmodifiedFile
	}
	return unmodifiedFile, formattedFile, nil
}
