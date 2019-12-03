#!/usr/bin/env bats

load helpers

@test "inspect-flags-order-verification" {
  run_buildah 1 inspect img1 -f "{{.ContainerID}}" -t="container"
  check_options_flag_err "-f"

  run_buildah 1 inspect img1 --format="{{.ContainerID}}"
  check_options_flag_err "--format={{.ContainerID}}"

  run_buildah 1 inspect img1 -t="image"
  check_options_flag_err "-t=image"
}

@test "inspect" {
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
  buildah_version=$(buildah --version | awk '{ print $3 }')
  inspect_cleaned=$(echo "$inspect_after_commit" | sed "s/io.buildah.version:${buildah_version}//g")
  expect_output --from="$inspect_cleaned" "$inspect_basic"

  imageid=$(buildah images -q alpine-image)
  containerid=$(buildah containers -q)

  # This one should not include buildah version
  run_buildah inspect --format '{{.OCIv1.Config}}' $containerid
  expect_output "$inspect_basic"

  # This one should.
  run_buildah inspect --type image --format '{{.OCIv1.Config}}' $imageid
  expect_output "$inspect_after_commit"

  buildah rm $cid
  buildah rmi alpine-image alpine
}

@test "inspect-config-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "Config" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-manifest-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "Manifest" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-ociv1-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "OCIv1" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-docker-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "Docker" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-config-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.Config}}" alpine | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-manifest-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.Manifest}}" alpine |  grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-ociv1-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.OCIv1}}" alpine |  grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-docker-is-json" {
	cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.Docker}}" alpine |  grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}
