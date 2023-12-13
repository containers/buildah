#!/usr/bin/env bats

load helpers

function mkcw_check_image() {
  local imageID="$1"
  local expectedEnv="$2"
  # Mount the container and take a look at what it got from the image.
  run_buildah from "$imageID"
  local ctrID="$output"
  run_buildah mount "$ctrID"
  local mountpoint="$output"
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
  # Should have a /bin/sh file from the base image, at least.
  test -s "$TEST_SCRATCH_DIR"/mount/bin/sh || test -L "$TEST_SCRATCH_DIR"/mount/bin/sh
  if shift ; then
    if shift ; then
      for pair in "$@" ; do
        inner=${pair##*:}
        outer=${pair%%:*}
        cmp ${outer} "$TEST_SCRATCH_DIR"/mount/${inner}
      done
    fi
  fi

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
  _prefetch bash
  createrandom randomfile1
  createrandom randomfile2

  echo -n mkcw-convert > "$TEST_SCRATCH_DIR"/key
  # image has one layer, check with all-lower-case TEE type name
  run_buildah mkcw --ignore-attestation-errors --type snp --passphrase=mkcw-convert --add-file randomfile1:/in-a-subdir/rnd1 busybox busybox-cw
  mkcw_check_image busybox-cw "" randomfile1:in-a-subdir/rnd1
  # image has multiple layers, check with all-upper-case TEE type name
  run_buildah mkcw --ignore-attestation-errors --type SNP --passphrase=mkcw-convert --add-file randomfile2:rnd2 bash bash-cw
  mkcw_check_image bash-cw "" randomfile2:/rnd2
}

@test "mkcw-commit" {
  skip_if_in_container
  skip_if_rootless_environment
  if ! which cryptsetup > /dev/null 2> /dev/null ; then
    skip "cryptsetup not found"
  fi
  _prefetch bash

  echo -n "mkcw commit" > "$TEST_SCRATCH_DIR"/key
  run_buildah from bash
  ctrID="$output"
  run_buildah commit --iidfile "$TEST_SCRATCH_DIR"/iid --cw type=SEV,ignore_attestation_errors,passphrase="mkcw commit" "$ctrID"
  mkcw_check_image $(cat "$TEST_SCRATCH_DIR"/iid)
  run_buildah commit --iidfile "$TEST_SCRATCH_DIR"/iid --cw type=sev,ignore_attestation_errors,passphrase="mkcw commit" "$ctrID"
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

  run_buildah build --iidfile "$TEST_SCRATCH_DIR"/iid --cw type=sev,ignore_attestation_errors,passphrase="mkcw build" -f bud/env/Dockerfile.check-env bud/env
  mkcw_check_image $(cat "$TEST_SCRATCH_DIR"/iid)

  # the key thing about this next bit is mixing --layers with a final
  # instruction in the Dockerfile that normally wouldn't produce a layer
  echo -n "mkcw build --layers" > "$TEST_SCRATCH_DIR"/key
  run_buildah build --iidfile "$TEST_SCRATCH_DIR"/iid --cw type=SEV,ignore_attestation_errors,passphrase="mkcw build --layers" --layers -f bud/env/Dockerfile.check-env bud/env
  mkcw_check_image $(cat "$TEST_SCRATCH_DIR"/iid)
}
