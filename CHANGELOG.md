# Changelog

## 0.4 - 2017-09-22
### Added
    Update buildah spec file to match new version
    Bump to version 0.4
    Add default transport to push if not provided
    Add authentication to commit and push
    Remove --transport flag
    Run: don't complain about missing volume locations
    Add credentials to buildah from
    Remove export command
    Bump containers/storage and containers/image

## 0.3 - 2017-07-20
## 0.2 - 2017-07-18
### Added
    Vendor in latest containers/image and containers/storage
    Update image-spec and runtime-spec to v1.0.0
    Add support for -- ending options parsing to buildah run
    Add/Copy need to support glob syntax
    Add flag to remove containers on commit
    Add buildah export support
    update 'buildah images' and 'buildah rmi' commands
    buildah containers/image: Add JSON output option
    Add 'buildah version' command
    Handle "run" without an explicit command correctly
    Ensure volume points get created, and with perms
    Add a -a/--all option to "buildah containers"

## 0.1 - 2017-06-14
### Added
    Vendor in latest container/storage container/image
    Add a "push" command
    Add an option to specify a Create date for images
    Allow building a source image from another image
    Improve buildah commit performance
    Add a --volume flag to "buildah run"
    Fix inspect/tag-by-truncated-image-ID
    Include image-spec and runtime-spec versions
    buildah mount command should list mounts when no arguments are given.
    Make the output image format selectable
    commit images in multiple formats
    Also import configurations from V2S1 images
    Add a "tag" command
    Add an "inspect" command
    Update reference comments for docker types origins
    Improve configuration preservation in imagebuildah
    Report pull/commit progress by default
    Contribute buildah.spec
    Remove --mount from buildah-from
    Add a build-using-dockerfile command (alias: bud)
    Create manpages for the buildah project
    Add installation for buildah and bash completions
    Rename "list"/"delete" to "containers"/"rm"
    Switch `buildah list quiet` option to only list container id's
    buildah delete should be able to delete multiple containers
    Correctly set tags on the names of pulled images
    Don't mix "config" in with "run" and "commit"
    Add a "list" command, for listing active builders
    Add "add" and "copy" commands
    Add a "run" command, using runc
    Massive refactoring
    Make a note to distinguish compression of layers

## 0.0 - 2017-01-26
### Added
    Initial version, needs work
