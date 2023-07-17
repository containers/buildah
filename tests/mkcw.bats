#!/usr/bin/env bats

load helpers

function mkcw_check_image() {
  local imageID="$1"
  local expectedEnv="$2"
  # Mount the container and take a look at what it got from the image.
  run_buildah from "$imageID"
  ctrID="$output"
  run_buildah mount "$ctrID"
  mountpoint="$output"
  # Should have a /disk.img file.
  test -s "$mountpoint"/disk.img
  # Should have a krun-sev.json file.
  test -s "$mountpoint"/krun-sev.json
  # Should have an executable entrypoint binary.
  test -s "$mountpoint"/entrypoint
  test -x "$mountpoint"/entrypoint
  # Should have a sticky /tmp directory.
  test -d "$mountpoint"/tmp
  test -k "$mountpoint"/tmp

  # Decrypt, mount, and take a look around.
  uuid=$(cryptsetup luksUUID "$mountpoint"/disk.img)
  cryptsetup luksOpen --key-file "$TEST_SCRATCH_DIR"/key "$mountpoint"/disk.img "$uuid"
  mkdir -p "$TEST_SCRATCH_DIR"/mount
  mount /dev/mapper/"$uuid" "$TEST_SCRATCH_DIR"/mount
  # Should have a not-empty config file with parts of an image's config.
  test -s "$TEST_SCRATCH_DIR"/mount/.krun_config.json
  if test -n "$expectedEnv" ; then
    grep -q "expectedEnv" "$TEST_SCRATCH_DIR"/mount/.krun_config.json
  fi
  # Should have a /tmp directory, at least.
  test -d "$TEST_SCRATCH_DIR"/mount/tmp

  # Clean up.
  umount "$TEST_SCRATCH_DIR"/mount
  cryptsetup luksClose "$uuid"
  buildah umount "$ctrID"
}

@test "mkcw-convert" {
  skip_if_in_container
  skip_if_rootless_environment
  if ! which cryptsetup > /dev/null 2> /dev/null ; then
    skip "cryptsetup not found"
  fi
  _prefetch busybox

  echo -n mkcw-convert > "$TEST_SCRATCH_DIR"/key
  run_buildah mkcw --ignore-attestation-errors --passphrase=mkcw-convert busybox busybox-cw
  mkcw_check_image busybox-cw
}

@test "mkcw-commit" {
  skip_if_in_container
  skip_if_rootless_environment
  if ! which cryptsetup > /dev/null 2> /dev/null ; then
    skip "cryptsetup not found"
  fi
  _prefetch busybox

  echo -n "mkcw commit" > "$TEST_SCRATCH_DIR"/key
  run_buildah from busybox
  ctrID="$output"
  run_buildah commit --iidfile "$TEST_SCRATCH_DIR"/iid --cw type=SEV,ignore_attestation_errors,passphrase="mkcw commit" "$ctrID"
  mkcw_check_image $(cat "$TEST_SCRATCH_DIR"/iid)
}

@test "mkcw build" {
  skip_if_in_container
  skip_if_rootless_environment
  if ! which cryptsetup > /dev/null 2> /dev/null ; then
    skip "cryptsetup not found"
  fi
  _prefetch alpine

  echo -n "mkcw build" > "$TEST_SCRATCH_DIR"/key
  run_buildah build --iidfile "$TEST_SCRATCH_DIR"/iid --cw type=SEV,ignore_attestation_errors,passphrase="mkcw build" -f bud/env/Dockerfile.check-env bud/env
  mkcw_check_image $(cat "$TEST_SCRATCH_DIR"/iid)
}
