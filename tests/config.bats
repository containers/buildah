#!/usr/bin/env bats

load helpers

@test "config-flags-order-verification" {
  run_buildah 1 config cnt1 --author=user1
  check_options_flag_err "--author=user1"

  run_buildah 1 config cnt1 --arch x86_54
  check_options_flag_err "--arch"

  run_buildah 1 config cnt1 --created-by buildahcli --cmd "/usr/bin/run.sh" --hostname "localhost1"
  check_options_flag_err "--created-by"

  run_buildah 1 config cnt1 --annotation=service=cache
  check_options_flag_err "--annotation=service=cache"
}

function check_matrix() {
  local setting=$1
  local expect=$2

  # matrix test: all permutations of .Docker.* and .OCIv1.* in all image types
  for image in docker oci; do
      for which in Docker OCIv1; do
        run_buildah --debug=false inspect --type=image --format "{{.$which.$setting}}" scratch-image-$image
        expect_output "$expect"
    done
  done
}

@test "config entrypoint using single element in JSON array (exec form)" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix "Config.Entrypoint" '[/ENTRYPOINT]'

  buildah rm $cid
  buildah rmi scratch-image-{docker,oci}
}

@test "config entrypoint using multiple elements in JSON array (exec form)" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --entrypoint '[ "/ENTRYPOINT", "ELEMENT2" ]' $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Entrypoint' '[/ENTRYPOINT ELEMENT2]'

  buildah rm $cid
  buildah rmi scratch-image-{docker,oci}
}

@test "config entrypoint using string (shell form)" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --entrypoint /ENTRYPOINT $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Entrypoint' '[/bin/sh -c /ENTRYPOINT]'

  buildah rm $cid
  buildah rmi scratch-image-{docker,oci}
}

@test "config set empty entrypoint doesn't wipe cmd" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --cmd "command" $cid
  buildah config --entrypoint "" $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Cmd' '[command]'

  buildah rm $cid
  buildah rmi scratch-image-{docker,oci}
}

@test "config entrypoint with cmd" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Config.Cmd' '[COMMAND-OR-ARGS]'

  buildah config \
   --entrypoint /ENTRYPOINT \
  $cid

  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid

  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  check_matrix 'Config.Cmd' '[COMMAND-OR-ARGS]'
}

@test "config" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config \
   --author TESTAUTHOR \
   --created-by COINCIDENCE \
   --arch SOMEARCH \
   --os SOMEOS \
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
  $cid

  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_matrix 'Author'       'TESTAUTHOR'
  check_matrix 'Architecture' 'SOMEARCH'
  check_matrix 'OS'           'SOMEOS'

  buildah --debug=false inspect --format '{{.ImageCreatedBy}}' $cid | grep COINCIDENCE

  check_matrix 'Config.Cmd'          '[COMMAND-OR-ARGS]'
  check_matrix 'Config.Entrypoint'   '[/bin/sh -c /ENTRYPOINT]'
  check_matrix 'Config.Env'          '[VARIABLE=VALUE1,VALUE2]'
  check_matrix 'Config.ExposedPorts' 'map[12345:{}]'
  check_matrix 'Config.Labels.exec'  'podman run -it --mount=type=bind,bind-propagation=Z,source=foo,destination=bar /script buz'
  check_matrix 'Config.Labels.LABEL' 'VALUE'
  check_matrix 'Config.StopSignal'   'SIGINT'
  check_matrix 'Config.User'         'likes:things'
  check_matrix 'Config.Volumes'      'map[/VOLUME:{}]'
  check_matrix 'Config.WorkingDir'   '/tmp'

  buildah --debug=false inspect --type=image --format '{{(index .Docker.History 0).Comment}}' scratch-image-docker | grep PROBABLY-EMPTY
  buildah --debug=false inspect --type=image --format '{{(index .OCIv1.History 0).Comment}}' scratch-image-docker | grep PROBABLY-EMPTY
  buildah --debug=false inspect --type=image --format '{{(index .Docker.History 0).Comment}}' scratch-image-oci | grep PROBABLY-EMPTY
  buildah --debug=false inspect --type=image --format '{{(index .OCIv1.History 0).Comment}}' scratch-image-oci | grep PROBABLY-EMPTY

  # The following aren't part of the Docker v2 spec, so they're discarded when we save to Docker format.
  buildah --debug=false inspect --type=image --format '{{.ImageAnnotations}}'                      scratch-image-oci    | grep ANNOTATION:VALUE1,VALUE2
  buildah --debug=false inspect              --format '{{.ImageAnnotations}}'                      $cid                 | grep ANNOTATION:VALUE1,VALUE2
  buildah --debug=false inspect --type=image --format '{{.Docker.Comment}}'                        scratch-image-docker | grep INFORMATIVE
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Domainname}}'              scratch-image-docker | grep mydomain.local
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Hostname}}'                scratch-image-docker | grep cleverhostname
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Shell}}'                   scratch-image-docker | grep /bin/arbitrarysh
  buildah --debug=false inspect               -f      '{{.Docker.Config.Healthcheck.Test}}'        scratch-image-docker | grep true
  buildah --debug=false inspect               -f      '{{.Docker.Config.Healthcheck.StartPeriod}}' scratch-image-docker | grep 5
  buildah --debug=false inspect               -f      '{{.Docker.Config.Healthcheck.Interval}}'    scratch-image-docker | grep 6
  buildah --debug=false inspect               -f      '{{.Docker.Config.Healthcheck.Timeout}}'     scratch-image-docker | grep 7
  buildah --debug=false inspect               -f      '{{.Docker.Config.Healthcheck.Retries}}'     scratch-image-docker | grep 8
}

@test "config env using --env expansion" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --env 'foo=bar' --env 'foo1=bar1' $cid
  buildah config --env 'combined=$foo/${foo1}' $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid env-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid env-image-oci

  run_buildah --debug=false inspect --type=image --format '{{.Docker.Config.Env}}' env-image-docker
  expect_output --substring "combined=bar/bar1"

  run_buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Env}}' env-image-docker
  expect_output --substring "combined=bar/bar1"

  buildah rm $cid
  buildah rmi env-image-docker env-image-oci
}

@test "user" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  bndoutput=$(buildah --debug=false run $cid grep CapBnd /proc/self/status)
  buildah config --user 1000 $cid
  run_buildah --debug=false run $cid id -u
  expect_output "1000"

  run_buildah --debug=false run $cid sh -c "grep CapEff /proc/self/status | cut -f2"
  expect_output "0000000000000000"

  run_buildah --debug=false run $cid grep CapBnd /proc/self/status
  expect_output "$bndoutput"

  buildah rm $cid
}
