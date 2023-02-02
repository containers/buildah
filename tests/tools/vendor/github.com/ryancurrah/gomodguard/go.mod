module github.com/ryancurrah/gomodguard

go 1.19

require (
	github.com/Masterminds/semver v1.5.0
	github.com/go-xmlfmt/xmlfmt v1.1.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/phayes/checkstyle v0.0.0-20170904204023-bfd46e6a821d
	github.com/t-yuki/gocover-cobertura v0.0.0-20180217150009-aaee18c8195c
	golang.org/x/mod v0.7.0
	gopkg.in/yaml.v2 v2.4.0
)

retract v1.2.1 // Originally tagged for commit hash that was subsequently removed, and replaced by another commit hash
