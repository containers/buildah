#!/usr/bin/env bats

load helpers

@test "config entrypoint" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid
  buildah commit --format dockerv2 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format ociv1 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-docker | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-oci | grep COMMAND-OR-ARGS

  buildah config \
   --entrypoint /ENTRYPOINT \
  $cid

  buildah commit --format dockerv2 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format ociv1 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  # Cmd should now be nil
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-docker | grep '\[\]'
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-docker | grep '\[\]'
  buildah --debug=false inspect --type=image --format '{{.Docker.Config.Cmd}}' scratch-image-oci | grep '\[\]'
  buildah --debug=false inspect --type=image --format '{{.OCIv1.Config.Cmd}}' scratch-image-oci | grep '\[\]'

  buildah config \
   --entrypoint /ENTRYPOINT \
   --cmd COMMAND-OR-ARGS \
  $cid

  buildah commit --format dockerv2 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format ociv1 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
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
   --volume /VOLUME \
   --workingdir /tmp \
   --label LABEL=VALUE \
   --stop-signal SIGINT \
   --annotation ANNOTATION=VALUE \
  $cid

  buildah commit --format dockerv2 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format ociv1 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

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

  # Annotations aren't part of the Docker v2 spec, so they're discarded when we save to Docker format.
  buildah --debug=false inspect --type=image --format '{{.ImageAnnotations}}' scratch-image-oci | grep ANNOTATION:VALUE
  buildah --debug=false inspect --type=image --format '{{.ImageAnnotations}}' scratch-image-oci | grep ANNOTATION:VALUE
}

@test "config with entrypoint" {
  ctr=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah config --workingdir /tmp $ctr
  buildah config --entrypoint /bin/sh $ctr
  buildah commit --signature-policy ${TESTSDIR}/policy.json $ctr test1
  buildah inspect --format '{{.Docker.Config.Entrypoint}}' test1 | grep '/bin/sh -c /bin/sh'
}
