package buildah

import (
	"io"

	"github.com/containers/image/copy"
	"github.com/containers/image/types"
)

func getCopyOptions(reportWriter io.Writer) *copy.Options {
	return &copy.Options{
		ReportWriter: reportWriter,
	}
}

func getSystemContext(signaturePolicyPath string) *types.SystemContext {
	sc := &types.SystemContext{}
	if signaturePolicyPath != "" {
		sc.SignaturePolicyPath = signaturePolicyPath
	}
	return sc
}
