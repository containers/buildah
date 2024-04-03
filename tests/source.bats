#!/usr/bin/env bats

load helpers

@test "source create" {
  # Create an empty source image and make sure it's properly initialized
  srcdir=${TEST_SCRATCH_DIR}/newsource
  run_buildah source create --author="Buildah authors" $srcdir

  # Inspect the index.json
  run jq -r .manifests[0].mediaType $srcdir/index.json
  expect_output "application/vnd.oci.image.manifest.v1+json"
  run jq -r .mediaType $srcdir/index.json
  expect_output null # TODO: common#839 will change this to "application/vnd.oci.image.index.v1+json"
  # Digest of manifest
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigest=${output//sha256:/} # strip off the sha256 prefix
  run stat $srcdir/blobs/sha256/$manifestDigest
  assert "$status" -eq 0 "status of stat(manifestDigest)"

  # Inspect the manifest
  run jq -r .schemaVersion $srcdir/blobs/sha256/$manifestDigest
  expect_output "2"
  run jq -r .layers $srcdir/blobs/sha256/$manifestDigest
  expect_output "null"
  run jq -r .config.mediaType $srcdir/blobs/sha256/$manifestDigest
  expect_output "application/vnd.oci.source.image.config.v1+json"
  run jq -r .mediaType $srcdir/blobs/sha256/$manifestDigest
  expect_output "application/vnd.oci.image.manifest.v1+json"
  run jq -r .config.size $srcdir/blobs/sha256/$manifestDigest
  # let's not check the size (afraid of time-stamp impacts)
  assert "$status" -eq 0 "status of jq .config.size"
  # Digest of config
  run jq -r .config.digest $srcdir/blobs/sha256/$manifestDigest
  configDigest=${output//sha256:/} # strip off the sha256 prefix
  run stat $srcdir/blobs/sha256/$configDigest
  assert "$status" -eq 0 "status of stat(configDigest)"

  # Inspect the config
  run jq -r .created $srcdir/blobs/sha256/$configDigest
  assert "$status" -eq 0 "status of jq .created on configDigest"
  creatd=$output
  run date --date="$output"
  assert "$status" -eq 0 "status of date (this should never ever fail)"
  run jq -r .author $srcdir/blobs/sha256/$configDigest
  expect_output "Buildah authors"

  # Directory mustn't exist
  run_buildah 125 source create $srcdir
  expect_output --substring "creating source image: "
  expect_output --substring " already exists"
}

@test "source add" {
  # Create an empty source image and make sure it's properly initialized.
  srcdir=${TEST_SCRATCH_DIR}/newsource
  run_buildah source create $srcdir

  # Digest of initial manifest
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigestEmpty=${output//sha256:/} # strip off the sha256 prefix
  run stat $srcdir/blobs/sha256/$manifestDigestEmpty
  assert "$status" -eq 0 "status of stat(manifestDigestEmpty)"

  # Add layer 1
  echo 111 > ${TEST_SCRATCH_DIR}/file1
  run_buildah source add $srcdir ${TEST_SCRATCH_DIR}/file1
  # Make sure the digest of the manifest changed
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigestFile1=${output//sha256:/} # strip off the sha256 prefix
  assert "$manifestDigestEmpty" != "$manifestDigestFile1" \
         "manifestDigestEmpty should differ from manifestDigestFile1"

  # Inspect layer 1
  run jq -r .layers[0].mediaType $srcdir/blobs/sha256/$manifestDigestFile1
  expect_output "application/vnd.oci.image.layer.v1.tar+gzip"
  run jq -r .layers[0].digest $srcdir/blobs/sha256/$manifestDigestFile1
  layer1Digest=${output//sha256:/} # strip off the sha256 prefix
  # Now make sure the reported size matches the actual one
  run jq -r .layers[0].size $srcdir/blobs/sha256/$manifestDigestFile1
  assert "$status" -eq 0 "status of jq .layers[0].size on manifestDigestFile1"
  layer1Size=$output
  run du -b $srcdir/blobs/sha256/$layer1Digest
  expect_output --substring "$layer1Size"

  # Add layer 2
  echo 222222aBitLongerForAdifferentSize > ${TEST_SCRATCH_DIR}/file2
  run_buildah source add $srcdir ${TEST_SCRATCH_DIR}/file2
  # Make sure the digest of the manifest changed
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigestFile2=${output//sha256:/} # strip off the sha256 prefix
  assert "$manifestDigestEmpty" != "$manifestDigestFile2" \
         "manifestDigestEmpty should differ from manifestDigestFile2"
  assert "$manifestDigestFile1" != "$manifestDigestFile2" \
         "manifestDigestFile1 should differ from manifestDigestFile2"

  # Make sure layer 1 is still in the manifest and remains unchanged
  run jq -r .layers[0].digest $srcdir/blobs/sha256/$manifestDigestFile2
  expect_output "sha256:$layer1Digest"
  run jq -r .layers[0].size $srcdir/blobs/sha256/$manifestDigestFile2
  expect_output "$layer1Size"

  # Inspect layer 2
  run jq -r .layers[1].mediaType $srcdir/blobs/sha256/$manifestDigestFile2
  expect_output "application/vnd.oci.image.layer.v1.tar+gzip"
  run jq -r .layers[1].digest $srcdir/blobs/sha256/$manifestDigestFile2
  layer2Digest=${output//sha256:/} # strip off the sha256 prefix
  # Now make sure the reported size matches the actual one
  run jq -r .layers[1].size $srcdir/blobs/sha256/$manifestDigestFile2
  assert "$status" -eq 0 "status of jq .layers[1].size on manifestDigestFile2"
  layer2Size=$output
  run du -b $srcdir/blobs/sha256/$layer2Digest
  expect_output --substring "$layer2Size"

  # Last but not least, make sure the two layers differ
  assert "$layer1Digest" != "$layer2Digest" "layer1Digest vs layer2Digest"
  assert "$layer1Size" != "$layer2Size"  "layer1Size vs layer2Size"
}

@test "source push/pull" {
  # Create an empty source image and make sure it's properly initialized.
  srcdir=${TEST_SCRATCH_DIR}/newsource
  run_buildah source create $srcdir

  # Add two layers
  echo 111 > ${TEST_SCRATCH_DIR}/file1
  run_buildah source add $srcdir ${TEST_SCRATCH_DIR}/file1
  echo 222... > ${TEST_SCRATCH_DIR}/file2
  run_buildah source add $srcdir ${TEST_SCRATCH_DIR}/file2

  start_registry

  # --quiet=true
  run_buildah source push --quiet --tls-verify=false --creds testuser:testpassword $srcdir localhost:${REGISTRY_PORT}/source:test
  expect_output ""
  # --quiet=false (implicit)
  run_buildah source push --digestfile=${TEST_SCRATCH_DIR}/digest.txt --tls-verify=false --creds testuser:testpassword $srcdir localhost:${REGISTRY_PORT}/source:test
  expect_output --substring "Copying blob"
  expect_output --substring "Copying config"
  cat ${TEST_SCRATCH_DIR}/digest.txt
  test -s ${TEST_SCRATCH_DIR}/digest.txt

  pulldir=${TEST_SCRATCH_DIR}/pulledsource
  # --quiet=true
  run_buildah source pull --quiet --tls-verify=false --creds testuser:testpassword localhost:${REGISTRY_PORT}/source:test $pulldir
  expect_output ""
  # --quiet=false (implicit)
  rm -rf $pulldir
  run_buildah source pull --tls-verify=false --creds testuser:testpassword localhost:${REGISTRY_PORT}/source:test $pulldir
  expect_output --substring "Copying blob"
  expect_output --substring "Copying config"

  run diff -r $srcdir $pulldir
  # FIXME: if there's a nonzero chance of this failing, include actual diffs
  assert "$status" -eq 0 "status from diff of srcdir vs pulldir"

  stop_registry
}
