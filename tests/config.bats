#!/usr/bin/env bats

load helpers

@test "config-flags-order-verification" {
  run_buildah 125 config cnt1 --author=user1
  check_options_flag_err "--author=user1"

  run_buildah 125 config cnt1 --arch x86_54
  check_options_flag_err "--arch"

  run_buildah 125 config cnt1 --created-by buildahcli --cmd "/usr/bin/run.sh" --hostname "localhost1"
  check_options_flag_err "--created-by"

  run_buildah 125 config cnt1 --annotation=service=cache
  check_options_flag_err "--annotation=service=cache"
}

@test "config-flags-verification" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --label LABEL $cid
  run_buildah config --annotation ANNOTATION $cid

  run_buildah 125 config --healthcheck 'AB "CD' $cid
  expect_output --substring 'error parsing --healthcheck "AB \\"CD": invalid command line string'

  run_buildah 125 config --healthcheck-interval ABCD $cid
  expect_output --substring 'error parsing --healthcheck-interval "ABCD": time: invalid duration "?ABCD"?'

  run_buildah 125 config --cmd 'AB "CD' $cid
  expect_output --substring 'error parsing --cmd "AB \\"CD": invalid command line string'

  run_buildah 125 config --env ENV $cid
  expect_output --substring 'error setting env "ENV": no value given'

  run_buildah 125 config --shell 'AB "CD' $cid
  expect_output --substring 'error parsing --shell "AB \\"CD": invalid command line string'
}

function check_matrix() {
  local setting=$1
  local expect=$2

  # matrix test: all permutations of .Docker.* and .OCIv1.* in all image types
  for image in docker oci; do
      for which in Docker OCIv1; do
	run_buildah inspect --type=image --format "{{.$which.$setting}}" scratch-image-$image
	expect_output "$expect"
    done
  done
}

@test "config entrypoint using single element in JSON array (exec form)" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix "Config.Entrypoint" '[/ENTRYPOINT]'
}

@test "config entrypoint using multiple elements in JSON array (exec form)" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --entrypoint '[ "/ENTRYPOINT", "ELEMENT2" ]' $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Entrypoint' '[/ENTRYPOINT ELEMENT2]'
}

@test "config entrypoint using string (shell form)" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --entrypoint /ENTRYPOINT $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Entrypoint' '[/bin/sh -c /ENTRYPOINT]'
}

@test "config set empty entrypoint doesn't wipe cmd" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --cmd "command" $cid
  run_buildah config --entrypoint "" $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Cmd' '[command]'
}

@test "config cmd without entrypoint" {
  run_buildah from --pull-never --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --cmd COMMAND-OR-ARGS \
  $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Cmd' '[COMMAND-OR-ARGS]'
  check_matrix 'Config.Entrypoint' '[]'
}

@test "config entrypoint with cmd" {
  run_buildah from --pull-never --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Cmd' '[COMMAND-OR-ARGS]'

  run_buildah from --pull-never --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --entrypoint /ENTRYPOINT \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Cmd' '[]'

  run_buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  check_matrix 'Config.Cmd' '[COMMAND-OR-ARGS]'

  run_buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd '[ "/COMMAND", "ARG1", "ARG2"]' \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  check_matrix 'Config.Cmd' '[/COMMAND ARG1 ARG2]'
}

@test "config remove all" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --port 12345 \
   --annotation ANNOTATION=VALUE1,VALUE2 \
   --env VARIABLE=VALUE1,VALUE2 \
   --volume /VOLUME \
   --label LABEL=VALUE \
  $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  run_buildah inspect --type=image --format '{{index .ImageAnnotations "ANNOTATION"}}' scratch-image-oci
  expect_output "VALUE1,VALUE2"
  run_buildah inspect              --format '{{index .ImageAnnotations "ANNOTATION"}}' $cid
  expect_output "VALUE1,VALUE2"
  check_matrix 'Config.ExposedPorts' 'map[12345:{}]'
  check_matrix 'Config.Env'          '[VARIABLE=VALUE1,VALUE2]'
  check_matrix 'Config.Labels.LABEL' 'VALUE'

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --port - \
   --annotation - \
   --env - \
   --volume - \
   --label - \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  run_buildah inspect --type=image --format '{{.ImageAnnotations}}'                      scratch-image-oci
  expect_output "map[]"
  run_buildah inspect              --format '{{.ImageAnnotations}}'                      $cid
  expect_output "map[]"
  check_matrix 'Config.ExposedPorts' 'map[]'
  check_matrix 'Config.Env'          '[]'
  check_matrix 'Config.Labels.LABEL' '<no value>'
}

@test "config" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --author TESTAUTHOR \
   --created-by COINCIDENCE \
   --arch amd64 \
   --os linux \
   --user likes:things \
   --port 12345 \
   --env VARIABLE=VALUE1,VALUE2 \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
   --comment INFORMATIVE \
   --history-comment PROBABLY-EMPTY \
   --volume /VOLUME \
   --workingdir /tmp \
   --label LABEL=VALUE \
   --label exec='podman run -it --mount=type=bind,bind-propagation=Z,source=foo,destination=bar /script buz'\
   --stop-signal SIGINT \
   --annotation ANNOTATION=VALUE1,VALUE2 \
   --shell /bin/arbitrarysh \
   --domainname mydomain.local \
   --hostname cleverhostname \
   --healthcheck "CMD /bin/true" \
   --healthcheck-start-period 5s \
   --healthcheck-interval 6s \
   --healthcheck-timeout 7s \
   --healthcheck-retries 8 \
   --onbuild "RUN touch /foo" \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Author'       'TESTAUTHOR'
  check_matrix 'Architecture' 'amd64'
  check_matrix 'OS'           'linux'

  run_buildah inspect --format '{{.ImageCreatedBy}}' $cid
  expect_output "COINCIDENCE"

  check_matrix 'Config.Cmd'          '[COMMAND-OR-ARGS]'
  check_matrix 'Config.Entrypoint'   '[/bin/sh -c /ENTRYPOINT]'
  check_matrix 'Config.Env'          '[VARIABLE=VALUE1,VALUE2]'
  check_matrix 'Config.ExposedPorts' 'map[12345:{}]'
  check_matrix 'Config.Labels.exec'  'podman run -it --mount=type=bind,bind-propagation=Z,source=foo,destination=bar /script buz'
  check_matrix 'Config.Labels.LABEL' 'VALUE'
  check_matrix 'Config.StopSignal'   'SIGINT'
  check_matrix 'Config.User'         'likes:things'
  check_matrix 'Config.Volumes'      "map[/VOLUME:{}]"
  check_matrix 'Config.WorkingDir'   '/tmp'

  run_buildah inspect --type=image --format '{{(index .Docker.History 0).Comment}}' scratch-image-docker
  expect_output "PROBABLY-EMPTY"
  run_buildah inspect --type=image --format '{{(index .OCIv1.History 0).Comment}}' scratch-image-docker
  expect_output "PROBABLY-EMPTY"
  run_buildah inspect --type=image --format '{{(index .Docker.History 0).Comment}}' scratch-image-oci
  expect_output "PROBABLY-EMPTY"
  run_buildah inspect --type=image --format '{{(index .OCIv1.History 0).Comment}}' scratch-image-oci
  expect_output "PROBABLY-EMPTY"

  # The following aren't part of the Docker v2 spec, so they're discarded when we save to Docker format.
  run_buildah inspect --type=image --format '{{index .ImageAnnotations "ANNOTATION"}}'   scratch-image-oci
  expect_output "VALUE1,VALUE2"
  run_buildah inspect              --format '{{index .ImageAnnotations "ANNOTATION"}}'   $cid
  expect_output "VALUE1,VALUE2"
  run_buildah inspect --type=image --format '{{.Docker.Comment}}'                        scratch-image-docker
  expect_output "INFORMATIVE"
  run_buildah inspect --type=image --format '{{.Docker.Config.Domainname}}'              scratch-image-docker
  expect_output "mydomain.local"
  run_buildah inspect --type=image --format '{{.Docker.Config.Hostname}}'                scratch-image-docker
  expect_output "cleverhostname"
  run_buildah inspect --type=image --format '{{.Docker.Config.Shell}}'                   scratch-image-docker
  expect_output "[/bin/arbitrarysh]"
  run_buildah inspect               -f      '{{.Docker.Config.Healthcheck.Test}}'        scratch-image-docker
  expect_output "[CMD /bin/true]"
  run_buildah inspect               -f      '{{.Docker.Config.Healthcheck.StartPeriod}}' scratch-image-docker
  expect_output "5s"
  run_buildah inspect               -f      '{{.Docker.Config.Healthcheck.Interval}}'    scratch-image-docker
  expect_output "6s"
  run_buildah inspect               -f      '{{.Docker.Config.Healthcheck.Timeout}}'     scratch-image-docker
  expect_output "7s"
  run_buildah inspect               -f      '{{.Docker.Config.Healthcheck.Retries}}'     scratch-image-docker
  expect_output "8"
  run_buildah inspect               -f      '{{.Docker.Config.OnBuild}}'                 scratch-image-docker
  expect_output "[RUN touch /foo]"
  rm -rf /VOLUME
}

@test "config env using local environment" {
  export foo=bar
  run_buildah from --pull-never --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --env 'foo' $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid env-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid env-image-oci

  run_buildah inspect --type=image --format '{{.Docker.Config.Env}}' env-image-docker
  expect_output --substring "foo=bar"

  run_buildah inspect --type=image --format '{{.OCIv1.Config.Env}}' env-image-docker
  expect_output --substring "foo=bar"
}

@test "config env using --env expansion" {
  run_buildah from --pull-never --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --env 'foo=bar' --env 'foo1=bar1' $cid
  run_buildah config --env 'combined=$foo/${foo1}' $cid
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid env-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid env-image-oci

  run_buildah inspect --type=image --format '{{.Docker.Config.Env}}' env-image-docker
  expect_output --substring "combined=bar/bar1"

  run_buildah inspect --type=image --format '{{.OCIv1.Config.Env}}' env-image-docker
  expect_output --substring "combined=bar/bar1"
}

@test "user" {
  _prefetch alpine
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid grep CapBnd /proc/self/status
  bndoutput=$output
  run_buildah config --user 1000 $cid
  run_buildah run $cid id -u
  expect_output "1000"

  run_buildah run $cid sh -c "grep CapEff /proc/self/status | cut -f2"
  expect_output "0000000000000000"

  run_buildah run $cid grep CapBnd /proc/self/status
  expect_output "$bndoutput"
}

@test "remove configs using '-' syntax" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config \
   --created-by COINCIDENCE \
   --volume /VOLUME \
   --env VARIABLE=VALUE1,VALUE2 \
   --label LABEL=VALUE \
   --port 12345 \
   --annotation ANNOTATION=VALUE1,VALUE2 \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  run_buildah inspect --format '{{.ImageCreatedBy}}' $cid
  expect_output "COINCIDENCE"

  check_matrix 'Config.Volumes'      "map[/VOLUME:{}]"
  check_matrix 'Config.Env'          '[VARIABLE=VALUE1,VALUE2]'
  check_matrix 'Config.Labels.LABEL' 'VALUE'
  check_matrix 'Config.ExposedPorts' 'map[12345:{}]'
  run_buildah inspect --type=image --format '{{index .ImageAnnotations "ANNOTATION"}}' scratch-image-oci
  expect_output "VALUE1,VALUE2"
  run_buildah inspect              --format '{{index .ImageAnnotations "ANNOTATION"}}' $cid
  expect_output "VALUE1,VALUE2"

  run_buildah config \
   --created-by COINCIDENCE \
   --volume /VOLUME- \
   --env VARIABLE- \
   --label LABEL- \
   --port 12345- \
   --annotation ANNOTATION- \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  run_buildah inspect --format '{{.ImageCreatedBy}}' $cid
  expect_output "COINCIDENCE"
  check_matrix 'Config.Volumes'      'map[]'
  check_matrix 'Config.Env'          '[]'
  check_matrix 'Config.Labels.LABEL' '<no value>'
  check_matrix 'Config.ExposedPorts' 'map[]'
  run_buildah inspect --type=image --format '{{index .ImageAnnotations "ANNOTATION"}}' scratch-image-oci
  expect_output ""
  run_buildah inspect              --format '{{index .ImageAnnotations "ANNOTATION"}}' $cid
  expect_output ""

  run_buildah config \
   --created-by COINCIDENCE \
   --volume /VOLUME- \
   --env VARIABLE=VALUE1,VALUE2 \
   --label LABEL=VALUE \
   --annotation ANNOTATION=VALUE1,VALUE2 \
  $cid

  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  run_buildah inspect --format '{{.ImageCreatedBy}}' $cid
  expect_output "COINCIDENCE"

  check_matrix 'Config.Volumes'      "map[]"
}
