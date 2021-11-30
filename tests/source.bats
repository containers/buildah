#!/usr/bin/env bats

load helpers

@test "source create" {
  # Create an empty source image and make sure it's properly initialized
  srcdir=${TESTDIR}/newsource
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
  [ "$status" -eq 0 ]

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
  [ "$status" -eq 0 ] # let's not check the size (afraid of time-stamp impacts)
  # Digest of config
  run jq -r .config.digest $srcdir/blobs/sha256/$manifestDigest
  configDigest=${output//sha256:/} # strip off the sha256 prefix
  run stat $srcdir/blobs/sha256/$configDigest
  [ "$status" -eq 0 ]

  # Inspect the config
  run jq -r .created $srcdir/blobs/sha256/$configDigest
  [ "$status" -eq 0 ]
  creatd=$output
  run date --date="$output"
  [ "$status" -eq 0 ]
  run jq -r .author $srcdir/blobs/sha256/$configDigest
  expect_output "Buildah authors"

  # Directory mustn't exist
  run_buildah 125 source create $srcdir
  expect_output --substring "error creating source image: "
  expect_output --substring " already exists"
}

@test "source add" {
  # Create an empty source image and make sure it's properly initialized.
  srcdir=${TESTDIR}/newsource
  run_buildah source create $srcdir

  # Digest of initial manifest
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigestEmpty=${output//sha256:/} # strip off the sha256 prefix
  run stat $srcdir/blobs/sha256/$manifestDigestEmpty
  [ "$status" -eq 0 ]

  # Add layer 1
  echo 111 > ${TESTDIR}/file1
  run_buildah source add $srcdir ${TESTDIR}/file1
  # Make sure the digest of the manifest changed
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigestFile1=${output//sha256:/} # strip off the sha256 prefix
  [ "$manifestDigestEmpty" != "$manifestDigestFile1" ]

  # Inspect layer 1
  run jq -r .layers[0].mediaType $srcdir/blobs/sha256/$manifestDigestFile1
  expect_output "application/vnd.oci.image.layer.v1.tar+gzip"
  run jq -r .layers[0].digest $srcdir/blobs/sha256/$manifestDigestFile1
  layer1Digest=${output//sha256:/} # strip off the sha256 prefix
  # Now make sure the reported size matches the actual one
  run jq -r .layers[0].size $srcdir/blobs/sha256/$manifestDigestFile1
  [ "$status" -eq 0 ]
  layer1Size=$output
  run du -b $srcdir/blobs/sha256/$layer1Digest
  expect_output --substring "$layer1Size"

  # Add layer 2
  echo 222222aBitLongerForAdifferentSize > ${TESTDIR}/file2
  run_buildah source add $srcdir ${TESTDIR}/file2
  # Make sure the digest of the manifest changed
  run jq -r .manifests[0].digest $srcdir/index.json
  manifestDigestFile2=${output//sha256:/} # strip off the sha256 prefix
  [ "$manifestDigestEmpty" != "$manifestDigestFile2" ]
  [ "$manifestDigestFile1" != "$manifestDigestFile2" ]

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
  [ "$status" -eq 0 ]
  layer2Size=$output
  run du -b $srcdir/blobs/sha256/$layer2Digest
  expect_output --substring "$layer2Size"

  # Last but not least, make sure the two layers differ
  [ "$layer1Digest" != "$layer2Digest" ]
  [ "$layer1Size" != "$layer2Size" ]
}

@test "source push/pull" {
  # Create an empty source image and make sure it's properly initialized.
  srcdir=${TESTDIR}/newsource
  run_buildah source create $srcdir

  # Add two layers
  echo 111 > ${TESTDIR}/file1
  run_buildah source add $srcdir ${TESTDIR}/file1
  echo 222... > ${TESTDIR}/file2
  run_buildah source add $srcdir ${TESTDIR}/file2

  run_buildah source push --tls-verify=false --creds testuser:testpassword $srcdir localhost:5000/source:test

  pulldir=${TESTDIR}/pulledsource
  run_buildah source pull --tls-verify=false --creds testuser:testpassword localhost:5000/source:test $pulldir

  run diff -r $srcdir $pulldir
  [ "$status" -eq 0 ]
}
