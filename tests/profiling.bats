#!/usr/bin/env bats

load helpers

@test "trace-profile flag creates a valid Go trace" {
  tmpfile="$TEST_SCRATCH_DIR/buildah-trace.$RANDOM.$RANDOM"
  run_buildah info --trace-profile="$tmpfile"
  assert "$status" -eq 0 "buildah info with --trace-profile should succeed"
  [ -s "$tmpfile" ] || die "trace profile should not be empty"
  run go tool trace -d=2 "$tmpfile"
  assert "$status" -eq 0 "go tool trace should succeed"
  rm -f "$tmpfile"
}

@test "cpu-profile flag creates a valid CPU profile" {
  tmpfile="$TEST_SCRATCH_DIR/buildah-cpu.$RANDOM.$RANDOM"
  run_buildah info --cpu-profile="$tmpfile"
  assert "$status" -eq 0 "buildah info with --cpu-profile should succeed"
  [ -s "$tmpfile" ] || die "CPU profile should not be empty"
  run go tool pprof -top "$tmpfile"
  assert "$status" -eq 0 "go tool pprof should succeed on CPU profile"
  rm -f "$tmpfile"
}

@test "memory-profile flag creates a valid memory profile" {
  tmpfile="$TEST_SCRATCH_DIR/buildah-mem.$RANDOM.$RANDOM"
  run_buildah info --memory-profile="$tmpfile"
  assert "$status" -eq 0 "buildah info with --memory-profile should succeed"
  [ -s "$tmpfile" ] || die "memory profile should not be empty"
  run go tool pprof -top "$tmpfile"
  assert "$status" -eq 0 "go tool pprof should succeed on memory profile"
  rm -f "$tmpfile"
}