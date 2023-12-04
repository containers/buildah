#!/usr/bin/env bats

load helpers

@test "commit-sbom-types" {
  _prefetch alpine ghcr.io/anchore/syft ghcr.io/aquasecurity/trivy
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  for squash in "--squash" "" ; do
    for sbomtype in syft syft-cyclonedx syft-spdx trivy trivy-cyclonedx trivy-spdx; do
      echo "[sbom type $sbomtype${squash:+, $squash}]"
      # clear out one file that we might need to overwrite, but leave the other to
      # ensure that we don't accidentally append content to files that are already
      # present
      rm -f localpurl.json
      # write to both the image and the local filesystem
      run_buildah commit $WITH_POLICY_JSON --sbom ${sbomtype} --sbom-output=localsbom.json --sbom-purl-output=localpurl.json --sbom-image-output=/root/sbom.json --sbom-image-purl-output=/root/purl.json $squash $cid alpine-derived-image
      # both files should exist now, and neither should be empty
      test -s localsbom.json
      test -s localpurl.json
      # compare them to their equivalents in the image
      run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine-derived-image
      dcid=$output
      run_buildah mount $dcid
      mountpoint=$output
      cmp $mountpoint/root/purl.json localpurl.json
      cmp $mountpoint/root/sbom.json localsbom.json
    done
  done
}

@test "bud-sbom-types" {
  _prefetch alpine ghcr.io/anchore/syft ghcr.io/aquasecurity/trivy
  for layers in --layers=true --layers=false --squash ; do
    for sbomtype in syft syft-cyclonedx syft-spdx trivy trivy-cyclonedx trivy-spdx; do
      echo "[sbom type $sbomtype with $layers]"
      # clear out one file that we might need to overwrite, but leave the other to
      # ensure that we don't accidentally append content to files that are already
      # present
      rm -f localpurl.json
      # write to both the image and the local filesystem
      run_buildah build $WITH_POLICY_JSON --sbom ${sbomtype} --sbom-output=localsbom.json --sbom-purl-output=localpurl.json --sbom-image-output=/root/sbom.json --sbom-image-purl-output=/root/purl.json $layers -t alpine-derived-image $BUDFILES/simple-multi-step
      # both files should exist now, and neither should be empty
      test -s localsbom.json
      test -s localpurl.json
      # compare them to their equivalents in the image
      run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine-derived-image
      dcid=$output
      run_buildah mount $dcid
      mountpoint=$output
      cmp $mountpoint/root/purl.json localpurl.json
      cmp $mountpoint/root/sbom.json localsbom.json
    done
  done
}

@test "bud-sbom-with-no-changes" {
  _prefetch alpine ghcr.io/anchore/syft ghcr.io/aquasecurity/trivy
  for sbomtype in syft syft-cyclonedx syft-spdx trivy trivy-cyclonedx trivy-spdx; do
    echo "[sbom type $sbomtype with $layers]"
    run_buildah build $WITH_POLICY_JSON --sbom ${sbomtype} --sbom-output=localsbom.json --sbom-purl-output=localpurl.json --sbom-image-output=/root/sbom.json --sbom-image-purl-output=/root/purl.json -t busybox-derived-image $BUDFILES/pull
    # both files should exist now, and neither should be empty
    test -s localsbom.json
    test -s localpurl.json
  done
}

@test "bud-sbom-with-only-config-changes" {
  _prefetch alpine ghcr.io/anchore/syft ghcr.io/aquasecurity/trivy
  for layers in --layers=true --layers=false ; do
    for sbomtype in syft syft-cyclonedx syft-spdx trivy trivy-cyclonedx trivy-spdx; do
      echo "[sbom type $sbomtype with $layers]"
      # clear out one file that we might need to overwrite, but leave the other to
      # ensure that we don't accidentally append content to files that are already
      # present
      rm -f localpurl.json
      run_buildah build $WITH_POLICY_JSON --sbom ${sbomtype} --sbom-output=localsbom.json --sbom-purl-output=localpurl.json --sbom-image-output=/root/sbom.json --sbom-image-purl-output=/root/purl.json $layers -t alpine-derived-image -f $BUDFILES/env/Dockerfile.check-env $BUDFILES/env
      # both files should exist now, and neither should be empty
      test -s localsbom.json
      test -s localpurl.json
    done
  done
}

@test "bud-sbom-with-non-presets" {
  _prefetch alpine busybox
  run_buildah build --debug $WITH_POLICY_JSON --sbom-output=localsbom.txt --sbom-purl-output=localpurl.txt --sbom-image-output=/root/sbom.txt --sbom-image-purl-output=/root/purl.txt --sbom-scanner-image=alpine --sbom-scanner-command='echo SCANNED ROOT {ROOTFS} > {OUTPUT}' --sbom-scanner-command='echo SCANNED BUILD CONTEXT {CONTEXT} > {OUTPUT}' --sbom-merge-strategy=cat -t busybox-derived-image $BUDFILES/pull
  # both files should exist now, and neither should be empty
  test -s localsbom.json
  test -s localpurl.json
}
