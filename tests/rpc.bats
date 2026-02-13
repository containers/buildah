#!/usr/bin/env bats

load helpers

@test "rpc noop" {
  run_buildah rpc -l $TEST_SCRATCH_DIR/socket pwd
  assert "$output" = $(pwd)
  run_buildah rpc --env LISTENER ${GRPCNOOP_BINARY} --env LISTENER first-arg second-arg
  assert "$output" = 'ignored:"first-arg,second-arg"'
}
