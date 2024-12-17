package config

import "github.com/openshift/imagebuilder"

var _ imagebuilder.Executor = &configOnlyExecutor{}
