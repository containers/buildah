#!/usr/bin/env bats

load helpers

@test "config-flags-order-verification" {
  run buildah config cnt1 --author=user1
  check_options_flag_err "--author=user1"

  run buildah config cnt1 --arch x86_54
  check_options_flag_err "--arch"

  run buildah config cnt1 --created-by buildahcli --cmd "/usr/bin/run.sh" --hostname "localhost1"
  check_options_flag_err "--created-by"

  run buildah config cnt1 --annotation=service=cache
  check_options_flag_err "--annotation=service=cache"
}

@test "config entrypoint using single element in JSON array (exec form)" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid entry-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid entry-image-oci

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' entry-image-docker
  [ "$output" = "[/ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' entry-image-oci
  [ "$output" = "[/ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' entry-image-docker
  [ "$output" = "[/ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' entry-image-oci
  [ "$output" = "[/ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  buildah rm $cid
  buildah rmi entry-image-docker entry-image-oci
}

@test "config entrypoint using multiple elements in JSON array (exec form)" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --entrypoint '[ "/ENTRYPOINT", "ELEMENT2" ]' $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid entry-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid entry-image-oci

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' entry-image-docker
  [ "$output" = "[/ENTRYPOINT ELEMENT2]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' entry-image-oci
  [ "$output" = "[/ENTRYPOINT ELEMENT2]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' entry-image-docker
  [ "$output" = "[/ENTRYPOINT ELEMENT2]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' entry-image-oci
  [ "$output" = "[/ENTRYPOINT ELEMENT2]" ]
  [ "$status" -eq 0 ]

  buildah rm $cid
  buildah rmi entry-image-docker entry-image-oci
}

@test "config entrypoint using string (shell form)" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --entrypoint /ENTRYPOINT $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid entry-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid entry-image-oci

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' entry-image-docker
  [ "$output" = "[/bin/sh -c /ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' entry-image-oci
  [ "$output" = "[/bin/sh -c /ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' entry-image-docker
  [ "$output" = "[/bin/sh -c /ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' entry-image-oci
  [ "$output" = "[/bin/sh -c /ENTRYPOINT]" ]
  [ "$status" -eq 0 ]

  buildah rm $cid
  buildah rmi entry-image-docker entry-image-oci
}

@test "config set empty entrypoint doesn't wipe cmd" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --cmd "command" $cid
  buildah config --entrypoint "" $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid entry-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid entry-image-oci

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' entry-image-docker
  [ "$output" = "[command]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' entry-image-oci
  [ "$output" = "[command]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' entry-image-docker
  [ "$output" = "[command]" ]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' entry-image-oci
  [ "$output" = "[command]" ]
  [ "$status" -eq 0 ]

  buildah rm $cid
  buildah rmi entry-image-docker entry-image-oci
}

@test "config entrypoint with cmd" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS

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
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS
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
   --env VARIABLE=VALUE \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
   --comment INFORMATIVE \
   --history-comment PROBABLY-EMPTY \
   --volume /VOLUME \
   --workingdir /tmp \
   --label LABEL=VALUE \
   --stop-signal SIGINT \
   --annotation ANNOTATION=VALUE \
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

  buildah --debug=false inspect --type=image --format '{{.Docker.Author}}' scratch-image-docker | grep TESTAUTHOR
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Author}}' scratch-image-docker | grep TESTAUTHOR
  buildah --debug=false inspect --type=image --format '{{.Docker.Author}}' scratch-image-oci | grep TESTAUTHOR
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Author}}' scratch-image-oci | grep TESTAUTHOR

  buildah --debug=false inspect --format '{{.ImageCreatedBy}}' $cid | grep COINCIDENCE

  buildah --debug=false inspect --type=image --format '{{.Docker.Architecture}}' scratch-image-docker | grep SOMEARCH
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Architecture}}' scratch-image-docker | grep SOMEARCH
  buildah --debug=false inspect --type=image --format '{{.Docker.Architecture}}' scratch-image-oci | grep SOMEARCH
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Architecture}}' scratch-image-oci | grep SOMEARCH

  buildah --debug=false inspect --type=image --format '{{.Docker.OS}}' scratch-image-docker | grep SOMEOS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.OS}}' scratch-image-docker | grep SOMEOS
  buildah --debug=false inspect --type=image --format '{{.Docker.OS}}' scratch-image-oci | grep SOMEOS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.OS}}' scratch-image-oci | grep SOMEOS

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.User}}' scratch-image-docker | grep likes:things
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.User}}' scratch-image-docker | grep likes:things
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.User}}' scratch-image-oci | grep likes:things
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.User}}' scratch-image-oci | grep likes:things

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.ExposedPorts}}' scratch-image-docker | grep 12345
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.ExposedPorts}}' scratch-image-docker | grep 12345
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.ExposedPorts}}' scratch-image-oci | grep 12345
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.ExposedPorts}}' scratch-image-oci | grep 12345

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Env}}' scratch-image-docker | grep VARIABLE=VALUE
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Env}}' scratch-image-docker | grep VARIABLE=VALUE
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Env}}' scratch-image-oci | grep VARIABLE=VALUE
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Env}}' scratch-image-oci | grep VARIABLE=VALUE

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' scratch-image-docker | grep /ENTRYPOINT
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' scratch-image-docker | grep /ENTRYPOINT
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Entrypoint}}' scratch-image-oci | grep /ENTRYPOINT
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Entrypoint}}' scratch-image-oci | grep /ENTRYPOINT

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Volumes}}' scratch-image-docker | grep /VOLUME
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Volumes}}' scratch-image-docker | grep /VOLUME
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Volumes}}' scratch-image-oci | grep /VOLUME
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Volumes}}' scratch-image-oci | grep /VOLUME

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.WorkingDir}}' scratch-image-docker | grep /tmp
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.WorkingDir}}' scratch-image-docker | grep /tmp
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.WorkingDir}}' scratch-image-oci | grep /tmp
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.WorkingDir}}' scratch-image-oci | grep /tmp

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Labels}}' scratch-image-docker | grep LABEL:VALUE
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Labels}}' scratch-image-docker | grep LABEL:VALUE
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Labels}}' scratch-image-oci | grep LABEL:VALUE
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Labels}}' scratch-image-oci | grep LABEL:VALUE

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.StopSignal}}' scratch-image-docker | grep SIGINT
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.StopSignal}}' scratch-image-docker | grep SIGINT
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.StopSignal}}' scratch-image-oci | grep SIGINT
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.StopSignal}}' scratch-image-oci | grep SIGINT

  buildah --debug=false inspect --type=image --format '{{(index .Docker.History 0).Comment}}' scratch-image-docker | grep PROBABLY-EMPTY
  buildah --debug=false inspect --type=image --format '{{(index .OCIv1.History 0).Comment}}' scratch-image-docker | grep PROBABLY-EMPTY
  buildah --debug=false inspect --type=image --format '{{(index .Docker.History 0).Comment}}' scratch-image-oci | grep PROBABLY-EMPTY
  buildah --debug=false inspect --type=image --format '{{(index .OCIv1.History 0).Comment}}' scratch-image-oci | grep PROBABLY-EMPTY

  # Annotations aren't part of the Docker v2 spec, so they're discarded when we save to Docker format.
  buildah --debug=false inspect --type=image --format '{{.ImageAnnotations}}' scratch-image-oci | grep ANNOTATION:VALUE
  buildah --debug=false inspect --format '{{.ImageAnnotations}}' $cid | grep ANNOTATION:VALUE

  # Comment isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect --type=image --format '{{.Docker.Comment}}' scratch-image-docker | grep INFORMATIVE
  # Domainname isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Domainname}}' scratch-image-docker | grep mydomain.local
  # Hostname isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Hostname}}' scratch-image-docker | grep cleverhostname
  # Shell isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Shell}}' scratch-image-docker | grep /bin/arbitrarysh
  # Healthcheck command isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect -f '{{.Docker.Config.Healthcheck.Test}}' scratch-image-docker | grep true
  # Healthcheck start period isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect -f '{{.Docker.Config.Healthcheck.StartPeriod}}' scratch-image-docker | grep 5
  # Healthcheck interval isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect -f '{{.Docker.Config.Healthcheck.Interval}}' scratch-image-docker | grep 6
  # Healthcheck timeout isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect -f '{{.Docker.Config.Healthcheck.Timeout}}' scratch-image-docker | grep 7
  # Healthcheck retry count isn't part of the OCI spec, so it's discarded when we save to OCI format.
  buildah --debug=false inspect -f '{{.Docker.Config.Healthcheck.Retries}}' scratch-image-docker | grep 8
}

@test "config env using --env expansion" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --env 'foo=bar' --env 'foo1=bar1' $cid
  buildah config --env 'combined=$foo/${foo1}' $cid
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid env-image-docker
  buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid env-image-oci

  run buildah --debug=false inspect --type=image --format '{{.Docker.Config.Env}}' env-image-docker
  echo $output
  [[ "$output" =~ combined=bar/bar1 ]]
  [ "$status" -eq 0 ]

  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Env}}' env-image-docker
  echo $output
  [[ "$output" =~ combined=bar/bar1 ]]
  [ "$status" -eq 0 ]

  buildah rm $cid
  buildah rmi env-image-docker env-image-oci
}

@test "user" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  bndoutput=$(buildah --debug=false run $cid grep CapBnd /proc/self/status)
  buildah config --user 1000 $cid
  run buildah --debug=false run $cid id -u
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ 1000 ]]

  run buildah --debug=false run $cid sh -c "grep CapEff /proc/self/status | cut -f2"
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "0000000000000000" ]]

  run buildah --debug=false run $cid grep CapBnd /proc/self/status
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ $bndoutput ]]

  buildah rm $cid
}
