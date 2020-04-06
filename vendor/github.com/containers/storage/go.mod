module github.com/containers/storage

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Microsoft/hcsshim v0.8.7
	github.com/docker/go-units v0.4.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/klauspost/compress v1.10.3
	github.com/klauspost/pgzip v1.2.3
	github.com/mattn/go-shellwords v1.0.10
	github.com/mistifyio/go-zfs v2.1.1+incompatible
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/runc v1.0.0-rc9
	github.com/opencontainers/runtime-spec v0.1.2-0.20190507144316-5b71a03e2700
	github.com/opencontainers/selinux v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.5.1
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2
	github.com/tchap/go-patricia v2.3.0+incompatible
	github.com/vbatts/tar-split v0.11.1
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7
	golang.org/x/sys v0.0.0-20191127021746-63cb32ae39b2
	gotest.tools v2.2.0+incompatible
)

go 1.13
