#!/usr/bin/env bats

load helpers

@test "images-flags-order-verification" {
  run_buildah images --all

  run_buildah 125 images img1 -n
  check_options_flag_err "-n"

  run_buildah 125 images img1 --filter="service=redis" img2
  check_options_flag_err "--filter=service=redis"

  run_buildah 125 images img1 img2 img3 -q
  check_options_flag_err "-q"
}

@test "images" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images
  expect_line_count 3
}

@test "images all test" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test ${TESTSDIR}/bud/use-layers
  run_buildah images
  expect_line_count 3

  run_buildah images -a
  expect_line_count 8

  # create a no name image which should show up when doing buildah images without the --all flag
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/use-layers
  run_buildah images
  expect_line_count 4
}

@test "images filter test" {
  _prefetch k8s.gcr.io/pause busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output

  run_buildah 125 images --noheading --filter since k8s.gcr.io/pause
  expect_output 'invalid image filter "since": must be in the format "filter=value"'


  run_buildah images --noheading --filter since=k8s.gcr.io/pause
  expect_line_count 1

  # pause* and u* should only give us pause image not busybox since its a AND between
  # two filters
  run_buildah images --noheading --filter "reference=pause*" --filter "reference=u*"
  expect_line_count 1
}

@test "images format test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --format "{{.Name}}"
  expect_line_count 2
}

@test "images noheading test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --noheading
  expect_line_count 2
}

@test "images quiet test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --quiet
  expect_line_count 2
}

@test "images no-trunc test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images -q --no-trunc
  expect_line_count 2
  expect_output --substring --from="${lines[0]}" "sha256"
}

@test "images json test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox

  for img in '' alpine busybox; do
      # e.g. [ { "id": "xx", ... },{ "id": "yy", ... } ]
      # We check for the presence of some keys, but not (yet) their values.
      # FIXME: once we can rely on 'jq' tool being present, improve this test!
      run_buildah images --json $img
      expect_output --from="${lines[0]}" "[" "first line of JSON output: array"
      for key in id names digest createdat size readonly history; do
          expect_output --substring "\"$key\": "
      done
  done
}

@test "images json dup test" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah tag test new-name

  run_buildah images --json
  expect_output --substring '"id": '
}

@test "images json valid" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid2=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid1 test
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid2 test2

  run_buildah images --json
  run python3 -m json.tool <<< "$output"
  [ "${status}" -eq 0 ]
}

@test "specify an existing image" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images alpine
  expect_line_count 2
}

@test "specify a nonexistent image" {
  run_buildah 125 images alpine
  expect_output --from="${lines[0]}" "alpine: image not known"
  expect_line_count 1
}

@test "Test dangling images" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah images
  expect_line_count 3

  run_buildah images --filter dangling=true
  expect_output --substring " <none> "
  expect_line_count 2

  run_buildah images --filter dangling=false
  expect_output --substring " latest "
  expect_line_count 2
}

@test "image digest test" {
  _prefetch busybox
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah images --digests
  expect_output --substring "sha256:"
}

@test "images in OCI format with no creation dates" {
  mkdir -p $TESTDIR/blobs/sha256

  # Create a layer.
  dd if=/dev/zero bs=512 count=2 of=$TESTDIR/blob
  layerdigest=$(sha256sum $TESTDIR/blob | awk '{print $1}')
  layersize=$(stat -c %s $TESTDIR/blob)
  mv $TESTDIR/blob $TESTDIR/blobs/sha256/${layerdigest}

  # Create a configuration blob that doesn't include a "created" date.
  now=$(TZ=UTC date +%Y-%m-%dT%H:%M:%S.%NZ)
  arch=$(go env GOARCH)
  cat > $TESTDIR/blob << EOF
  {
    "architecture": "$arch",
    "os": "linux",
    "config": {
        "Env": [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
        ],
        "Cmd": [
            "sh"
        ]
    },
    "rootfs": {
        "type": "layers",
        "diff_ids": [
            "sha256:${layerdigest}"
        ]
    },
    "history": [
        {
            "created": "${now}",
            "created_by": "/bin/sh -c #(nop) ADD file:${layerdigest} in / "
        }
    ]
  }
EOF
  configdigest=$(sha256sum $TESTDIR/blob | awk '{print $1}')
  configsize=$(stat -c %s $TESTDIR/blob)
  mv $TESTDIR/blob $TESTDIR/blobs/sha256/${configdigest}

  # Create a manifest for that configuration blob and layer.
  cat > $TESTDIR/blob << EOF
  {
    "schemaVersion": 2,
    "config": {
        "mediaType": "application/vnd.oci.image.config.v1+json",
        "digest": "sha256:${configdigest}",
        "size": ${configsize}
    },
    "layers": [
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:${layerdigest}",
            "size": ${layersize}
        }
    ]
  }
EOF
  manifestdigest=$(sha256sum $TESTDIR/blob | awk '{print $1}')
  manifestsize=$(stat -c %s $TESTDIR/blob)
  mv $TESTDIR/blob $TESTDIR/blobs/sha256/${manifestdigest}

  # Add the manifest to the image index.
  cat > $TESTDIR/index.json << EOF
  {
    "schemaVersion": 2,
    "manifests": [
        {
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:${manifestdigest}",
            "size": ${manifestsize}
        }
    ]
  }
EOF

  # Mark the directory as a layout directory.
  echo -n '{"imageLayoutVersion": "1.0.0"}' > $TESTDIR/oci-layout

  # Import the image.
  run_buildah pull oci:$TESTDIR

  # Inspect the image.  We shouldn't crash.
  run_buildah inspect ${configdigest}
  # List images.  We shouldn't crash.
  run_buildah images
}
