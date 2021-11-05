#!/usr/bin/env bats

load helpers

@test "inspect-flags-order-verification" {
  run_buildah 125 inspect img1 -f "{{.ContainerID}}" -t="container"
  check_options_flag_err "-f"

  run_buildah 125 inspect img1 --format="{{.ContainerID}}"
  check_options_flag_err "--format={{.ContainerID}}"

  run_buildah 125 inspect img1 -t="image"
  check_options_flag_err "-t=image"
}

@test "inspect" {
  _prefetch alpine
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" alpine-image

  # e.g. { map[] [PATH=/....] [] [/bin/sh] map[]  map[] }
  run_buildah inspect --format '{{.OCIv1.Config}}' alpine
  expect_output --substring "map.*PATH=.*/bin/sh.*map"
  inspect_basic=$output

  # Now inspect the committed image. Output should be _mostly_ the same...
  run_buildah inspect --type image --format '{{.OCIv1.Config}}' alpine-image
  inspect_after_commit=$output

  # ...except that at some point in November 2019 buildah-inspect started
  # including version. Strip it out,
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}
  inspect_cleaned=$(echo "$inspect_after_commit" | sed "s/io.buildah.version:${buildah_version}//g")
  expect_output --from="$inspect_cleaned" "$inspect_basic"

  run_buildah images -q alpine-image
  imageid=$output
  run_buildah containers -q
  containerid=$output

  # This one should not include buildah version
  run_buildah inspect --format '{{.OCIv1.Config}}' $containerid
  expect_output "$inspect_basic"

  # This one should.
  run_buildah inspect --type image --format '{{.OCIv1.Config}}' $imageid
  expect_output "$inspect_after_commit"
}

@test "inspect-config-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect alpine
        expect_output --substring 'Config.*\{'
}

@test "inspect-manifest-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect alpine
        expect_output --substring 'Manifest.*\{'
}

@test "inspect-ociv1-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect alpine
        expect_output --substring 'OCIv1.*\{'
}

@test "inspect-docker-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect alpine
        expect_output --substring 'Docker.*\{'
}

@test "inspect-format-config-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect --format "{{.Config}}" alpine
        expect_output --substring '\{'
}

@test "inspect-format-manifest-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect --format "{{.Manifest}}" alpine
        expect_output --substring '\{'
}

@test "inspect-format-ociv1-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect --format "{{.OCIv1}}" alpine
        expect_output --substring '\{'
}

@test "inspect-format-docker-is-json" {
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah inspect --format "{{.Docker}}" alpine
        expect_output --substring '\{'
}

@test "inspect-format-docker-variant" {
	# github.com/containerd/containerd/platforms.Normalize() converts Arch:"armhf" to Arch:"arm"+Variant:"v7",
	# so check that platform normalization happens at least for that one
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json --arch=armhf scratch
	cid=$output
	run_buildah inspect --format "{{.Docker.Architecture}}" $cid
	[[ "$output" == "arm" ]]
	run_buildah inspect --format "{{.Docker.Variant}}" $cid
	[[ "$output" == "v7" ]]
}
