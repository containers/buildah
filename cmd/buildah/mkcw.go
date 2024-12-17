package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/parse"
	"github.com/spf13/cobra"
)

func mkcwCmd(c *cobra.Command, args []string, options buildah.CWConvertImageOptions) error {
	ctx := getContext()

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return err
	}

	if options.AttestationURL == "" && options.DiskEncryptionPassphrase == "" {
		return fmt.Errorf("neither --attestation-url nor --passphrase flags provided, disk would not be decryptable")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options.InputImage = args[0]
	options.Tag = args[1]
	options.ReportWriter = os.Stderr
	imageID, _, _, err := buildah.CWConvertImage(ctx, systemContext, store, options)
	if err == nil {
		fmt.Printf("%s\n", imageID)
	}
	return err
}

func init() {
	var teeType string
	var addFile []string
	var options buildah.CWConvertImageOptions
	mkcwDescription := `Convert a conventional image to a confidential workload image.`
	mkcwCommand := &cobra.Command{
		Use:   "mkcw",
		Short: "Convert a conventional image to a confidential workload image",
		Long:  mkcwDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.TeeType = parse.TeeType(teeType)
			if len(addFile) > 0 {
				options.ExtraImageContent = make(map[string]string)
				for _, spec := range addFile {
					source, dest, haveDest := strings.Cut(spec, ":")
					if !haveDest {
						dest = source
					}
					st, err := os.Stat(source)
					if err != nil {
						return fmt.Errorf("parsing add-file argument %q: source %q: %w", spec, source, err)
					}
					if st.IsDir() {
						return fmt.Errorf("parsing add-file argument %q: source %q is not a regular file", spec, source)
					}
					options.ExtraImageContent[dest] = source
				}
			}
			return mkcwCmd(cmd, args, options)
		},
		Example: `buildah mkcw localhost/repository:typical localhost/repository:cw`,
		Args:    cobra.ExactArgs(2),
	}
	mkcwCommand.SetUsageTemplate(UsageTemplate())
	rootCmd.AddCommand(mkcwCommand)
	flags := mkcwCommand.Flags()
	flags.SetInterspersed(false)

	flags.StringVarP(&teeType, "type", "t", "", "TEE (trusted execution environment) type: SEV,SNP (default: SNP)")
	flags.StringArrayVar(&addFile, "add-file", nil, "add contents of a file to the image at a specified path (`source:destination`)")
	flags.StringVarP(&options.AttestationURL, "attestation-url", "u", "", "attestation server URL")
	flags.StringVarP(&options.BaseImage, "base-image", "b", "", "alternate base image (default: scratch)")
	flags.StringVarP(&options.DiskEncryptionPassphrase, "passphrase", "p", "", "disk encryption passphrase")
	flags.IntVarP(&options.CPUs, "cpus", "c", 0, "number of CPUs to expect")
	flags.IntVarP(&options.Memory, "memory", "m", 0, "amount of memory to expect (MB)")
	flags.StringVarP(&options.WorkloadID, "workload-id", "w", "", "workload ID")
	flags.StringVarP(&options.Slop, "slop", "s", "25%", "extra space needed for converting a container rootfs to a disk image")
	flags.StringVarP(&options.FirmwareLibrary, "firmware-library", "f", "", "location of libkrunfw-sev.so")
	flags.BoolVarP(&options.IgnoreAttestationErrors, "ignore-attestation-errors", "", false, "ignore attestation errors")
	if err := flags.MarkHidden("ignore-attestation-errors"); err != nil {
		panic(fmt.Sprintf("error marking ignore-attestation-errors as hidden: %v", err))
	}
	flags.String("signature-policy", "", "`pathname` of signature policy file (not usually used)")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
}
