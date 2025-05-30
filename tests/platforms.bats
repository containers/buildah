#!/usr/bin/env bats

load helpers

# read the platform information from the configuration of the main image for
# the oci layout in $1
read_oci_layout_platform() {
  run jq -r '.manifests[0].digest' "$1"/index.json
  assert $status -eq 0
  local alg="${output%%:*}"
  local hex="${output##*:}"
  run jq -r '.config.digest' "$1"/blobs/"$alg"/"$hex"
  assert $status -eq 0
  alg="${output%%:*}"
  hex="${output##*:}"
  run jq -r '.os' "$1"/blobs/"$alg"/"$hex"
  assert $status -eq 0
  local os="$output"
  run jq -r '.architecture' "$1"/blobs/"$alg"/"$hex"
  assert $status -eq 0
  local arch="$output"
  run jq -r '.variant' "$1"/blobs/"$alg"/"$hex"
  assert $status -eq 0
  local variant="$output"
  if test "$variant" = null ; then
    variant=
  fi
  echo "$os"/"$arch""${variant:+/$variant}"
}

@test "implicit-and-explicit-platforms" {
  _prefetch busybox
  local context="$TEST_SCRATCH_DIR"/context
  mkdir -p "$context"
  cat > "$context"/Dockerfile.scratch << EOF
  FROM scratch
  COPY . .
EOF
  cat > "$context"/Dockerfile.base << EOF
  FROM busybox
EOF
  cat > "$context"/Dockerfile.derived << EOF
  FROM busybox
  COPY . .
EOF
  run_buildah version --json
  run jq -r .buildPlatform <<< "$output"
  assert $status -eq 0
  local buildplatform="$output"

  # these should either get the default determined at runtime, or the value passed
  for platform in "" linux/amd64 linux/arm64 linux/arm64/v8 ; do
    local arch="${platform##*/}"
    run_buildah build --layers --no-cache -t oci:"$TEST_SCRATCH_DIR"/scratch-"${arch:-default}" ${platform:+--platform "$platform"} -f "$context"/Dockerfile.scratch "$context"
    run read_oci_layout_platform "$TEST_SCRATCH_DIR"/scratch-"${arch:-default}"
    assert $status -eq 0
    assert "$output" = "${platform:-${buildplatform}}" "for build based on scratch for ${platform:-default platform}"
  done

  # these should inherit the values from the base image that we used for the given platform
  for platform in "" linux/amd64 linux/arm64 linux/arm64/v8 ; do
    arch="${platform##*/}"
    for base in base derived ; do
      run_buildah build --layers --no-cache -t "$base"-"${arch:-default}" ${platform:+--platform "$platform"} -f "$context"/Dockerfile."$base" "$context"
      run_buildah push "$base"-"${arch:-default}" oci:"$TEST_SCRATCH_DIR"/"$base"-"${arch:-default}"
    done
    run read_oci_layout_platform "$TEST_SCRATCH_DIR"/base-"${arch:-default}"
    assert $status -eq 0
    baseplatform="$output"
    run read_oci_layout_platform "$TEST_SCRATCH_DIR"/derived-"${arch:-default}"
    assert $status -eq 0
    derivedplatform="$output"
    assert "$baseplatform" = "$derivedplatform" "for build based for ${platform:-default platform}"
  done
}
