package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/containers/buildah/imagebuildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
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
	Tag       string
	ID        string
	Name      string
	Digest    string
	CreatedAt string
	Size      string
}

type imageOptions struct {
	all       bool
	digests   bool
	format    string
	json      bool
	noHeading bool
	truncate  bool
	quiet     bool
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
			Name:  "all, a",
			Usage: "show all images, including intermediate images from a build",
		},
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
			Usage: "pretty-print images using a Go template",
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
		Name:                   "images",
		Usage:                  "List images in local storage",
		Description:            imagesDescription,
		Flags:                  sortFlags(imagesFlags),
		Action:                 imagesCmd,
		ArgsUsage:              "[imageName]",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func imagesCmd(c *cli.Context) error {
	name := ""
	args := c.Args()
	if len(args) > 0 {
		if c.Bool("all") {
			return errors.Errorf("when using the --all switch, you may not pass any images names or IDs")
		}

		if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
			return err
		}
		if len(args) == 1 {
			name = args.Get(0)
		} else {
			return errors.New("'buildah images' requires at most 1 argument")
		}
	}

	if err := parse.ValidateFlags(c, imagesFlags); err != nil {
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

	if c.IsSet("quiet") && c.IsSet("format") {
		return errors.Errorf("quiet and format are mutually exclusive")
	}

	opts := imageOptions{
		all:       c.Bool("all"),
		digests:   c.Bool("digests"),
		format:    c.String("format"),
		json:      c.Bool("json"),
		noHeading: c.Bool("noheading"),
		truncate:  !c.Bool("notruncate"),
		quiet:     c.Bool("quiet"),
	}
	ctx := getContext()

	var params *filterParams
	if c.IsSet("filter") {
		params, err = parseFilter(ctx, store, images, c.String("filter"))
		if err != nil {
			return errors.Wrapf(err, "error parsing filter")
		}
	}

	if len(images) > 0 && !opts.noHeading && !opts.quiet && opts.format == "" && !opts.json {
		outputHeader(opts.truncate, opts.digests)
	}

	return outputImages(ctx, images, store, params, name, opts)
}

func parseFilter(ctx context.Context, store storage.Store, images []storage.Image, filter string) (*filterParams, error) {
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
			beforeDate, err := setFilterDate(ctx, store, images, pair[1])
			if err != nil {
				return nil, fmt.Errorf("no such id: %s", pair[0])
			}
			params.beforeDate = beforeDate
			params.beforeImage = pair[1]
		case "since":
			sinceDate, err := setFilterDate(ctx, store, images, pair[1])
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

func setFilterDate(ctx context.Context, store storage.Store, images []storage.Image, imgName string) (time.Time, error) {
	for _, image := range images {
		for _, name := range image.Names {
			if matchesReference(name, imgName) {
				// Set the date to this image
				ref, err := is.Transport.ParseStoreReference(store, image.ID)
				if err != nil {
					return time.Time{}, fmt.Errorf("error parsing reference to image %q: %v", image.ID, err)
				}
				img, err := ref.NewImage(ctx, nil)
				if err != nil {
					return time.Time{}, fmt.Errorf("error reading image %q: %v", image.ID, err)
				}
				defer img.Close()
				inspect, err := img.Inspect(ctx)
				if err != nil {
					return time.Time{}, fmt.Errorf("error inspecting image %q: %v", image.ID, err)
				}
				date := *inspect.Created
				return date, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("Could not locate image %q", imgName)
}

func outputHeader(truncate, digests bool) {
	if truncate {
		fmt.Printf("%-56s %-20s %-20s ", "IMAGE NAME", "IMAGE TAG", "IMAGE ID")
	} else {
		fmt.Printf("%-56s %-20s %-64s ", "IMAGE NAME", "IMAGE TAG", "IMAGE ID")
	}

	if digests {
		fmt.Printf("%-71s ", "DIGEST")
	}

	fmt.Printf("%-22s %s\n", "CREATED AT", "SIZE")
}

func outputImages(ctx context.Context, images []storage.Image, store storage.Store, filters *filterParams, argName string, opts imageOptions) error {
	found := false
	for _, image := range images {
		createdTime := image.Created

		inspectedTime, digest, size, _ := getDateAndDigestAndSize(ctx, image, store)
		if !inspectedTime.IsZero() {
			if createdTime != inspectedTime {
				logrus.Debugf("image record and configuration disagree on the image's creation time for %q, using the one from the configuration", image)
				createdTime = inspectedTime
			}
		}
		createdTime = createdTime.Local()

		// If all is false and the image doesn't have a name, check to see if the top layer of the image is a parent
		// to another image's top layer. If it is, then it is an intermediate image so don't print out if the --all flag
		// is not set.
		isParent, err := imageIsParent(store, image.TopLayer)
		if err != nil {
			logrus.Errorf("error checking if image is a parent %q: %v", image.ID, err)
		}
		if !opts.all && len(image.Names) == 0 && isParent {
			continue
		}

		names := []string{}
		if len(image.Names) > 0 {
			names = image.Names
		} else {
			// images without names should be printed with "<none>" as the image name
			names = append(names, "<none>:<none>")
		}
		breakOuter := false
		for name, tags := range imagebuildah.ReposToMap(names) {
			for _, tag := range tags {
				if !matchesReference(name+":"+tag, argName) {
					continue
				}
				found = true

				if !matchesFilter(ctx, image, store, name+":"+tag, filters) {
					continue
				}
				if opts.quiet {
					fmt.Printf("%-64s\n", image.ID)
					// We only want to print each id once
					breakOuter = true
					break
				}
				if opts.json {
					JSONImage := jsonImage{ID: image.ID, Names: image.Names}
					data, err2 := json.MarshalIndent(JSONImage, "", "    ")
					if err2 != nil {
						return err2
					}
					fmt.Printf("%s\n", data)
					break
				}
				params := imageOutputParams{
					Tag:       tag,
					ID:        image.ID,
					Name:      name,
					Digest:    digest,
					CreatedAt: createdTime.Format("Jan 2, 2006 15:04"),
					Size:      formattedSize(size),
				}
				if opts.format != "" {
					if err := outputUsingTemplate(opts.format, params); err != nil {
						return err
					}
					continue
				}
				outputUsingFormatString(opts.truncate, opts.digests, params)
			}
			if breakOuter { // Show only one imageID when quiet
				break
			}
		}
	}

	if !found && argName != "" {
		return errors.Errorf("No such image %s", argName)
	}

	return nil
}

func matchesFilter(ctx context.Context, image storage.Image, store storage.Store, name string, params *filterParams) bool {
	if params == nil {
		return true
	}
	if params.dangling != "" && !matchesDangling(name, params.dangling) {
		return false
	} else if params.label != "" && !matchesLabel(ctx, image, store, params.label) {
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
	if dangling == "false" && !strings.Contains(name, "<none>") {
		return true
	} else if dangling == "true" && strings.Contains(name, "<none>") {
		return true
	}
	return false
}

func matchesLabel(ctx context.Context, image storage.Image, store storage.Store, label string) bool {
	storeRef, err := is.Transport.ParseStoreReference(store, image.ID)
	if err != nil {
		return false
	}
	img, err := storeRef.NewImage(ctx, nil)
	if err != nil {
		return false
	}
	defer img.Close()
	info, err := img.Inspect(ctx)
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
	return image.Created.IsZero() || image.Created.Before(params.beforeDate)
}

// Returns true if the image was created since the filter image.  Returns
// false otherwise
func matchesSinceImage(image storage.Image, name string, params *filterParams) bool {
	return image.Created.IsZero() || image.Created.After(params.sinceDate)
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

/*
According to  https://en.wikipedia.org/wiki/Binary_prefix
We should be return numbers based on 1000, rather then 1024
*/
func formattedSize(size int64) string {
	suffixes := [5]string{"B", "KB", "MB", "GB", "TB"}

	count := 0
	formattedSize := float64(size)
	for formattedSize >= 1000 && count < 4 {
		formattedSize /= 1000
		count++
	}
	return fmt.Sprintf("%.3g %s", formattedSize, suffixes[count])
}

func outputUsingTemplate(format string, params imageOutputParams) error {
	if matched, err := regexp.MatchString("{{.*}}", format); err != nil {
		return errors.Wrapf(err, "error validating format provided: %s", format)
	} else if !matched {
		return errors.Errorf("error invalid format provided: %s", format)
	}

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
		fmt.Printf("%-56s %-20s %-20.12s", params.Name, params.Tag, params.ID)
	} else {
		fmt.Printf("%-56s %-20s %-64s", params.Name, params.Tag, params.ID)
	}

	if digests {
		fmt.Printf(" %-64s", params.Digest)
	}
	fmt.Printf(" %-22s %s\n", params.CreatedAt, params.Size)
}
