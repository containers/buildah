#!/usr/bin/env bats

load helpers


function setup() {
    setup_tests
    unset SSH_AUTH_SOCK
}

function teardown(){
  if [[ -n "$SSH_AUTH_SOCK" ]]; then ssh-agent -k;fi
  teardown_tests
}

@test "bud with ssh key" {
  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  ssh-keygen -b 2048 -t rsa -f $mytmpdir/sshkey -q -N ""
  fingerprint=$(ssh-keygen -l -f $mytmpdir/sshkey -E md5 | awk '{ print $2; }')

  run_buildah bud --ssh default=$mytmpdir/sshkey $WITH_POLICY_JSON  -t sshimg -f $BUDFILES/run-mounts/Dockerfile.ssh $BUDFILES/run-mounts
  expect_output --substring $fingerprint

  run_buildah from sshimg
  run_buildah 1 run sshimg-working-container cat /run/buildkit/ssh_agent.0
  expect_output --substring "cat: can't open '/run/buildkit/ssh_agent.0': No such file or directory"
  run_buildah rm -a
}

@test "bud with ssh key secret accessed on second RUN" {
 _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  ssh-keygen -b 2048 -t rsa -f $mytmpdir/sshkey -q -N ""
  fingerprint=$(ssh-keygen -l -f $mytmpdir/sshkey -E md5 | awk '{ print $2; }')

  run_buildah 2 bud --ssh default=$mytmpdir/sshkey $WITH_POLICY_JSON  -t sshimg -f $BUDFILES/run-mounts/Dockerfile.ssh_access $BUDFILES/run-mounts
  expect_output --substring "Could not open a connection to your authentication agent."
}

@test "bud with containerfile ssh options" {
  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  ssh-keygen -b 2048 -t rsa -f $mytmpdir/sshkey -q -N ""
  fingerprint=$(ssh-keygen -l -f $mytmpdir/sshkey -E md5 | awk '{ print $2; }')

  run_buildah bud --ssh default=$mytmpdir/sshkey $WITH_POLICY_JSON  -t secretopts -f $BUDFILES/run-mounts/Dockerfile.ssh_options $BUDFILES/run-mounts
  expect_output --substring "444"
  expect_output --substring "1000"
  expect_output --substring "1001"
}

@test "bud with ssh sock" {
  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  ssh-keygen -b 2048 -t rsa -f $mytmpdir/sshkey -q -N ""
  fingerprint=$(ssh-keygen -l -f $mytmpdir/sshkey -E md5 | awk '{ print $2; }')
  eval "$(ssh-agent -s)"
  ssh-add $mytmpdir/sshkey

  run_buildah bud --ssh default=$mytmpdir/sshkey $WITH_POLICY_JSON  -t sshimg -f $BUDFILES/run-mounts/Dockerfile.ssh $BUDFILES/run-mounts
  expect_output --substring $fingerprint

  run_buildah from sshimg
  run_buildah 1 run sshimg-working-container cat /run/buildkit/ssh_agent.0
  expect_output --substring "cat: can't open '/run/buildkit/ssh_agent.0': No such file or directory"
  run_buildah rm -a
}

