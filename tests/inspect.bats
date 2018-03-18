#!/usr/bin/env bats

load helpers

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

