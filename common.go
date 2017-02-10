package buildah

import (
	"github.com/containers/image/copy"
	"github.com/containers/image/types"
)

func getCopyOptions() *copy.Options {
	return &copy.Options{}
}

func getSystemContext(signaturePolicyPath string) *types.SystemContext {
	sc := &types.SystemContext{}
	if signaturePolicyPath != "" {
		sc.SignaturePolicyPath = signaturePolicyPath
	}
	return sc
}
