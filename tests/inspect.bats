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
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	run_buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" alpine-image

	out1=$(buildah inspect --format '{{.OCIv1.Config}}' alpine)
	out2=$(buildah inspect --type image --format '{{.OCIv1.Config}}' alpine-image)
	[ "$out1" != "" ]
	[ "$out1" = "$out2" ]
}

@test "inspect-config-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "Config" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-manifest-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "Manifest" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-ociv1-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "OCIv1" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-docker-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect alpine | grep "Docker" | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-config-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.Config}}" alpine | grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-manifest-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.Manifest}}" alpine |  grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-ociv1-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.OCIv1}}" alpine |  grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}

@test "inspect-format-docker-is-json" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	out=$(buildah inspect --format "{{.Docker}}" alpine |  grep "{" | wc -l)
	# if there is "{" it's a JSON string
	[ "$out" -ne "0" ]
	buildah rm $cid
	buildah rmi -f alpine
}
