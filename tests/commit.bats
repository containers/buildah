#!/usr/bin/env bats

load helpers

@test "commit-flags-order-verification" {
  run_buildah 1 commit cnt1 --tls-verify
  check_options_flag_err "--tls-verify"

  run_buildah 1 commit cnt1 -q
  check_options_flag_err "-q"

  run_buildah 1 commit cnt1 -f=docker --quiet --creds=bla:bla
  check_options_flag_err "-f=docker"

  run_buildah 1 commit cnt1 --creds=bla:bla
  check_options_flag_err "--creds=bla:bla"
}

@test "commit" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid alpine-image
  run_buildah images alpine-image
  buildah rm $cid
  buildah rmi -a
}

@test "commit format test" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid alpine-image-oci
  buildah commit --format docker --disable-compression=false --signature-policy ${TESTSDIR}/policy.json $cid alpine-image-docker

  buildah --log-level=error inspect --type=image --format '{{.Manifest}}' alpine-image-oci | grep "application/vnd.oci.image.layer.v1.tar"
  buildah --log-level=error inspect --type=image --format '{{.Manifest}}' alpine-image-docker | grep "application/vnd.docker.image.rootfs.diff.tar.gzip"
  buildah rm $cid
  buildah rmi -a
}

@test "commit quiet test" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah --log-level=error commit --iidfile /dev/null --signature-policy ${TESTSDIR}/policy.json -q $cid alpine-image
  expect_output ""
  buildah rm $cid
  buildah rmi -a
}

@test "commit rm test" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah commit --signature-policy ${TESTSDIR}/policy.json --rm $cid alpine-image
  run_buildah 1 --log-level=error rm $cid
  expect_output --substring "error removing container \"alpine-working-container\": error reading build container: container not known"
  buildah rmi -a
}

@test "commit-alternate-storage" {
  echo FROM
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json openshift/hello-openshift)
  echo COMMIT
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid "containers-storage:[vfs@${TESTDIR}/root2+${TESTDIR}/runroot2]newimage"
  echo FROM
  buildah --storage-driver vfs --root ${TESTDIR}/root2 --runroot ${TESTDIR}/runroot2 from --signature-policy ${TESTSDIR}/policy.json newimage
}

@test "commit-rejected-name" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah 1 --log-level=error commit --signature-policy ${TESTSDIR}/policy.json $cid ThisNameShouldBeRejected
  expect_output --substring "must be lower"
}

@test "commit-no-empty-created-by" {
  if ! python3 -c 'import json, sys' 2> /dev/null ; then
    skip "python interpreter with json module not found"
  fi
  target=new-image
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)

  run_buildah --log-level=error config --created-by "untracked actions" $cid
  run_buildah --log-level=error commit --signature-policy ${TESTSDIR}/policy.json $cid ${target}
  run_buildah --log-level=error inspect --format '{{.Config}}' ${target}
  config="$output"
  run python3 -c 'import json, sys; config = json.load(sys.stdin); print(config["history"][len(config["history"])-1]["created_by"])' <<< "$config"
  echo "$output"
  [ "${status}" -eq 0 ]
  [ "$output" == "untracked actions" ]

  run_buildah --log-level=error config --created-by "" $cid
  run_buildah --log-level=error commit --signature-policy ${TESTSDIR}/policy.json $cid ${target}
  run_buildah --log-level=error inspect --format '{{.Config}}' ${target}
  config="$output"
  run python3 -c 'import json, sys; config = json.load(sys.stdin); print(config["history"][len(config["history"])-1]["created_by"])' <<< "$config"
  echo "$output"
  [ "${status}" -eq 0 ]
  [ "$output" == "/bin/sh" ]
}

@test "commit-no-name" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid
}

@test "commit should fail with nonexist authfile" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah 1 commit --authfile /tmp/nonexist --signature-policy ${TESTSDIR}/policy.json $cid alpine-image
  buildah rm $cid
  buildah rmi -a
}

@test "commit-builder-identity" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	buildah commit --signature-policy ${TESTSDIR}/policy.json $cid alpine-image

	buildah_version=$(buildah --version | awk '{ print $3 }')
	version=$(buildah inspect --format '{{ index .Docker.Config.Labels "io.buildah.version"}}' alpine-image)

	[ "$version" == "$buildah_version" ]
	buildah rm $cid
	buildah rmi -f alpine-image
}

@test "commit-parent-id" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  iid=$(buildah inspect --format '{{.FromImageID}}' $cid)
  echo image ID: "$iid"
  buildah commit --signature-policy ${TESTSDIR}/policy.json --format docker $cid alpine-image
  parentid=$(buildah inspect --format '{{.Docker.Parent}}' alpine-image)
  echo parent ID: "$parentid"
  [ "$parentid" = sha256:"$iid" ]
}

@test "commit-container-id" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid=$(buildah containers --format '{{.ContainerID}}:{{.ContainerName}}' | grep :"$cid"'$' | cut -f1 -d:)
  echo container ID: "$cid"
  buildah commit --signature-policy ${TESTSDIR}/policy.json --format docker $cid alpine-image
  containerid=$(buildah inspect --format '{{.Docker.Container}}' alpine-image)
  echo recorded container ID: "$containerid"
  [ "$containerid" = "$cid" ]
}

@test "commit with name" {
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json --name busyboxc busybox
  expect_output "busyboxc"

  # Commit with a new name
  newname="commitbyname/busyboxname"
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json busyboxc $newname

  run_buildah from --signature-policy ${TESTSDIR}/policy.json localhost/$newname
  expect_output "busyboxname-working-container"

  cname=$output
  run_buildah inspect --format '{{.FromImage}}' $cname
  expect_output "localhost/$newname:latest"

  buildah rm busyboxc $cname
  buildah rmi $newname
}

@test "commit to docker-distribution" {
  buildah from --signature-policy ${TESTSDIR}/policy.json --name busyboxc busybox
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword busyboxc docker://localhost:5000/commit/busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name fromdocker --tls-verify=false --creds testuser:testpassword docker://localhost:5000/commit/busybox
  buildah rm busyboxc fromdocker
}
