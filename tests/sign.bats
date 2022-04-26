#!/usr/bin/env bats

load helpers

function _gpg_setup() {
  if ! which gpg > /dev/null 2> /dev/null ; then
    skip 'gpg command not found in $PATH'
  fi

  export GNUPGHOME=${TEST_SCRATCH_DIR}/.gnupg
  mkdir -p --mode=0700 $GNUPGHOME

  # gpg on f30 and above needs this, otherwise:
  #   gpg: agent_genkey failed: Inappropriate ioctl for device
  # ...but gpg on f29 (and, probably, Ubuntu) doesn't grok this
  GPGOPTS='--pinentry-mode loopback'
  if gpg --pinentry-mode asdf 2>&1 | grep -qi 'Invalid option'; then
      GPGOPTS=
  fi

  cat > ${TEST_SCRATCH_DIR}/genkey-answers <<- EOF
	%echo Generating a basic OpenPGP key
	Key-Type: RSA
	Key-Length: 2048
	Name-Real: Amanda Lorian
	Name-Comment: Mandy to her friends
	Name-Email: amanda@localhost
	%commit
	%echo done
	EOF
  gpg --batch $GPGOPTS --gen-key --passphrase '' < ${TEST_SCRATCH_DIR}/genkey-answers
}


@test "commit-pull-push-signatures" {
  _gpg_setup
  _prefetch alpine

  mkdir -p ${TEST_SCRATCH_DIR}/signed-image ${TEST_SCRATCH_DIR}/unsigned-image

  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit $WITH_POLICY_JSON --sign-by amanda@localhost $cid signed-alpine-image

  # Pushing should preserve the signature.
  run_buildah push $WITH_POLICY_JSON signed-alpine-image dir:${TEST_SCRATCH_DIR}/signed-image
  ls -l ${TEST_SCRATCH_DIR}/signed-image/
  test -s ${TEST_SCRATCH_DIR}/signed-image/signature-1

  # Pushing with --remove-signatures should remove the signature.
  run_buildah push $WITH_POLICY_JSON --remove-signatures signed-alpine-image dir:${TEST_SCRATCH_DIR}/unsigned-image
  ls -l ${TEST_SCRATCH_DIR}/unsigned-image/
  ! test -s ${TEST_SCRATCH_DIR}/unsigned-image/signature-1

  run_buildah commit $WITH_POLICY_JSON $cid unsigned-alpine-image
  # Pushing with --sign-by should fail add the signature to a dir: location, if it tries to add them.
  run_buildah 125 push $WITH_POLICY_JSON --sign-by amanda@localhost unsigned-alpine-image dir:${TEST_SCRATCH_DIR}/signed-image
  expect_output --substring "Cannot determine canonical Docker reference"

  # Clear out images, so that we don't have leftover signatures when we pull in an image that will end up
  # causing us to merge its contents with the image with the same ID.
  run_buildah rmi -a -f

  # Pulling with --remove-signatures should remove signatures, and pushing should have none to keep.
  run_buildah pull $WITH_POLICY_JSON --quiet dir:${TEST_SCRATCH_DIR}/signed-image
  imageID="$output"
  run_buildah push $WITH_POLICY_JSON "$imageID" dir:${TEST_SCRATCH_DIR}/unsigned-image
  ls -l ${TEST_SCRATCH_DIR}/unsigned-image/
  ! test -s ${TEST_SCRATCH_DIR}/unsigned-image/signature-1

  # Build a manifest list and try to push the list with signatures.
  run_buildah manifest create list
  run_buildah manifest add list $imageID
  run_buildah 125 manifest push $WITH_POLICY_JSON --sign-by amanda@localhost --all list dir:${TEST_SCRATCH_DIR}/signed-image
  expect_output --substring "Cannot determine canonical Docker reference"
  run_buildah manifest push $WITH_POLICY_JSON --all list dir:${TEST_SCRATCH_DIR}/unsigned-image
}

@test "build-with-dockerfile-signatures" {
  _gpg_setup

  builddir=${TEST_SCRATCH_DIR}/builddir
  mkdir -p $builddir
  cat > ${builddir}/Dockerfile <<- EOF
	FROM scratch
	ADD Dockerfile /
	EOF

  # We should be able to sign at build-time.
  run_buildah bud $WITH_POLICY_JSON --sign-by amanda@localhost -t signed-scratch-image ${builddir}

  mkdir -p ${TEST_SCRATCH_DIR}/signed-image
  # Pushing should preserve the signature.
  run_buildah push $WITH_POLICY_JSON signed-scratch-image dir:${TEST_SCRATCH_DIR}/signed-image
  ls -l ${TEST_SCRATCH_DIR}/signed-image/
  test -s ${TEST_SCRATCH_DIR}/signed-image/signature-1
}
