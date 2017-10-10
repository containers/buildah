package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"encoding/json"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type jsonImage struct {
	ID    string   `json:"id"`
	Names []string `json:"names"`
}

type imageOutputParams struct {
	ID        string
	Name      string
	Digest    string
	CreatedAt string
	Size      string
}

type filterParams struct {
	dangling         string
	label            string
	beforeImage      string // Images are sorted by date, so we can just output until we see the image
	sinceImage       string // Images are sorted by date, so we can just output until we don't see the image
	beforeDate       time.Time
	sinceDate        time.Time
	referencePattern string
}

var (
	imagesFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "digests",
			Usage: "show digests",
		},
		cli.StringFlag{
			Name:  "filter, f",
			Usage: "filter output based on conditions provided",
		},
		cli.StringFlag{
			Name:  "format",
			Usage: "pretty-print images using a Go template. will override --quiet",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "output in JSON format",
		},
		cli.BoolFlag{
			Name:  "noheading, n",
			Usage: "do not print column headings",
		},
		cli.BoolFlag{
			Name:  "no-trunc, notruncate",
			Usage: "do not truncate output",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "display only image IDs",
		},
	}

	imagesDescription = "Lists locally stored images."
	imagesCommand     = cli.Command{
		Name:        "images",
		Usage:       "List images in local storage",
		Description: imagesDescription,
		Flags:       imagesFlags,
		Action:      imagesCmd,
		ArgsUsage:   " ",
	}
)

func imagesCmd(c *cli.Context) error {
	if err := validateFlags(c, imagesFlags); err != nil {
		return err
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	images, err := store.Images()
	if err != nil {
		return errors.Wrapf(err, "error reading images")
	}

	quiet := c.Bool("quiet")
	truncate := !c.Bool("no-trunc")
	digests := c.Bool("digests")
	hasTemplate := c.IsSet("format")

	name := ""
	if len(c.Args()) == 1 {
		name = c.Args().Get(0)
	} else if len(c.Args()) > 1 {
		return errors.New("'buildah images' requires at most 1 argument")
	}
	if c.IsSet("json") {
		JSONImages := []jsonImage{}
		for _, image := range images {
			JSONImages = append(JSONImages, jsonImage{ID: image.ID, Names: image.Names})
		}
		data, err2 := json.MarshalIndent(JSONImages, "", "    ")
		if err2 != nil {
			return err2
		}
		fmt.Printf("%s\n", data)
		return nil
	}
	var params *filterParams
	if c.IsSet("filter") {
		params, err = parseFilter(store, images, c.String("filter"))
		if err != nil {
			return errors.Wrapf(err, "error parsing filter")
		}
	} else {
		params = nil
	}

	if len(images) > 0 && !c.Bool("noheading") && !quiet && !hasTemplate {
		outputHeader(truncate, digests)
	}

	return outputImages(images, c.String("format"), store, params, name, hasTemplate, truncate, digests, quiet)
}

func parseFilter(store storage.Store, images []storage.Image, filter string) (*filterParams, error) {
	params := new(filterParams)
	filterStrings := strings.Split(filter, ",")
	for _, param := range filterStrings {
		pair := strings.SplitN(param, "=", 2)
		switch strings.TrimSpace(pair[0]) {
		case "dangling":
			if pair[1] == "true" || pair[1] == "false" {
				params.dangling = pair[1]
			} else {
				return nil, fmt.Errorf("invalid filter: '%s=[%s]'", pair[0], pair[1])
			}
		case "label":
			params.label = pair[1]
		case "before":
			beforeDate, err := setFilterDate(store, images, pair[1])
			if err != nil {
				return nil, fmt.Errorf("no such id: %s", pair[0])
			}
			params.beforeDate = beforeDate
			params.beforeImage = pair[1]
		case "since":
			sinceDate, err := setFilterDate(store, images, pair[1])
			if err != nil {
				return nil, fmt.Errorf("no such id: %s", pair[0])
			}
			params.sinceDate = sinceDate
			params.sinceImage = pair[1]
		case "reference":
			params.referencePattern = pair[1]
		default:
			return nil, fmt.Errorf("invalid filter: '%s'", pair[0])
		}
	}
	return params, nil
}

func setFilterDate(store storage.Store, images []storage.Image, imgName string) (time.Time, error) {
	for _, image := range images {
		for _, name := range image.Names {
			if matchesReference(name, imgName) {
				// Set the date to this image
				ref, err := is.Transport.ParseStoreReference(store, "@"+image.ID)
				if err != nil {
					return time.Time{}, fmt.Errorf("error parsing reference to image %q: %v", image.ID, err)
				}
				img, err := ref.NewImage(nil)
				if err != nil {
					return time.Time{}, fmt.Errorf("error reading image %q: %v", image.ID, err)
				}
				defer img.Close()
				inspect, err := img.Inspect()
				if err != nil {
					return time.Time{}, fmt.Errorf("error inspecting image %q: %v", image.ID, err)
				}
				date := inspect.Created
				return date, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("Could not locate image %q", imgName)
}

func outputHeader(truncate, digests bool) {
	if truncate {
		fmt.Printf("%-20s %-56s ", "IMAGE ID", "IMAGE NAME")
	} else {
		fmt.Printf("%-64s %-56s ", "IMAGE ID", "IMAGE NAME")
	}

	if digests {
		fmt.Printf("%-64s ", "DIGEST")
	}

	fmt.Printf("%-22s %s\n", "CREATED AT", "SIZE")
}

func outputImages(images []storage.Image, format string, store storage.Store, filters *filterParams, argName string, hasTemplate, truncate, digests, quiet bool) error {
	for _, image := range images {
		createdTime := image.Created

		inspectedTime, digest, size, _ := getDateAndDigestAndSize(image, store)
		if !inspectedTime.IsZero() {
			if createdTime != inspectedTime {
				logrus.Debugf("image record and configuration disagree on the image's creation time for %q, using the one from the configuration", image)
				createdTime = inspectedTime
			}
		}

		names := []string{}
		if len(image.Names) > 0 {
			names = image.Names
		} else {
			// images without names should be printed with "<none>" as the image name
			names = append(names, "<none>")
		}
		for _, name := range names {
			if !matchesFilter(image, store, name, filters) || !matchesReference(name, argName) {
				continue
			}
			if quiet {
				fmt.Printf("%-64s\n", image.ID)
				// We only want to print each id once
				break
			}

			params := imageOutputParams{
				ID:        image.ID,
				Name:      name,
				Digest:    digest,
				CreatedAt: createdTime.Format("Jan 2, 2006 15:04"),
				Size:      formattedSize(size),
			}
			if hasTemplate {
				if err := outputUsingTemplate(format, params); err != nil {
					return err
				}
				continue
			}

			outputUsingFormatString(truncate, digests, params)
		}
	}
	return nil
}

func matchesFilter(image storage.Image, store storage.Store, name string, params *filterParams) bool {
	if params == nil {
		return true
	}
	if params.dangling != "" && !matchesDangling(name, params.dangling) {
		return false
	} else if params.label != "" && !matchesLabel(image, store, params.label) {
		return false
	} else if params.beforeImage != "" && !matchesBeforeImage(image, name, params) {
		return false
	} else if params.sinceImage != "" && !matchesSinceImage(image, name, params) {
		return false
	} else if params.referencePattern != "" && !matchesReference(name, params.referencePattern) {
		return false
	}
	return true
}

func matchesDangling(name string, dangling string) bool {
	if dangling == "false" && name != "<none>" {
		return true
	} else if dangling == "true" && name == "<none>" {
		return true
	}
	return false
}

func matchesLabel(image storage.Image, store storage.Store, label string) bool {
	storeRef, err := is.Transport.ParseStoreReference(store, "@"+image.ID)
	if err != nil {
		return false
	}
	img, err := storeRef.NewImage(nil)
	if err != nil {
		return false
	}
	defer img.Close()
	info, err := img.Inspect()
	if err != nil {
		return false
	}

	pair := strings.SplitN(label, "=", 2)
	for key, value := range info.Labels {
		if key == pair[0] {
			if len(pair) == 2 {
				if value == pair[1] {
					return true
				}
			} else {
				return false
			}
		}
	}
	return false
}

// Returns true if the image was created since the filter image.  Returns
// false otherwise
func matchesBeforeImage(image storage.Image, name string, params *filterParams) bool {
	if image.Created.IsZero() || image.Created.Before(params.beforeDate) {
		return true
	}
	return false
}

// Returns true if the image was created since the filter image.  Returns
// false otherwise
func matchesSinceImage(image storage.Image, name string, params *filterParams) bool {
	if image.Created.IsZero() || image.Created.After(params.sinceDate) {
		return true
	}
	return false
}

func matchesID(imageID, argID string) bool {
	return strings.HasPrefix(imageID, argID)
}

func matchesReference(name, argName string) bool {
	if argName == "" {
		return true
	}
	splitName := strings.Split(name, ":")
	// If the arg contains a tag, we handle it differently than if it does not
	if strings.Contains(argName, ":") {
		splitArg := strings.Split(argName, ":")
		return strings.HasSuffix(splitName[0], splitArg[0]) && (splitName[1] == splitArg[1])
	}
	return strings.HasSuffix(splitName[0], argName)
}

func formattedSize(size int64) string {
	suffixes := [5]string{"B", "KB", "MB", "GB", "TB"}

	count := 0
	formattedSize := float64(size)
	for formattedSize >= 1024 && count < 4 {
		formattedSize /= 1024
		count++
	}
	return fmt.Sprintf("%.4g %s", formattedSize, suffixes[count])
}

func outputUsingTemplate(format string, params imageOutputParams) error {
	tmpl, err := template.New("image").Parse(format)
	if err != nil {
		return errors.Wrapf(err, "Template parsing error")
	}

	err = tmpl.Execute(os.Stdout, params)
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func outputUsingFormatString(truncate, digests bool, params imageOutputParams) {
	if truncate {
		fmt.Printf("%-20.12s %-56s", params.ID, params.Name)
	} else {
		fmt.Printf("%-64s %-56s", params.ID, params.Name)
	}

	if digests {
		fmt.Printf(" %-64s", params.Digest)
	}
	fmt.Printf(" %-22s %s\n", params.CreatedAt, params.Size)
}
