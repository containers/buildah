#!/usr/bin/env bats

load helpers

@test "bud with CDI" {
  skip_if_chroot
  _prefetch busybox
  cdidir=${TEST_SCRATCH_DIR}/cdi-config-dir
  mkdir -p $cdidir
  sed -e s:@@hostcdipath@@:$cdidir:g $BUDFILES/cdi/containers-cdi.yaml > $cdidir/containers-cdi.yaml
  chmod 644 $cdidir/containers-cdi.yaml
  echo === Begin CDI configuration in $cdidir/containers-cdi.yaml ===
  cat $cdidir/containers-cdi.yaml
  echo === End CDI configuration ===
  run_buildah build $WITH_POLICY_JSON --cdi-config-dir=$cdidir --security-opt label=disable --device=containers.github.io/sample=all --device=/dev/null:/dev/outsidenull:rwm $BUDFILES/cdi
}

@test "from with CDI" {
  skip_if_chroot
  _prefetch busybox
  cdidir=${TEST_SCRATCH_DIR}/cdi-config-dir
  mkdir -p $cdidir
  sed -e s:@@hostcdipath@@:$cdidir:g $BUDFILES/cdi/containers-cdi.yaml > $cdidir/containers-cdi.yaml
  chmod 644 $cdidir/containers-cdi.yaml
  echo === Begin CDI configuration in $cdidir/containers-cdi.yaml ===
  cat $cdidir/containers-cdi.yaml
  echo === End CDI configuration ===
  run_buildah from $WITH_POLICY_JSON --security-opt label=disable --cdi-config-dir=$cdidir --device=containers.github.io/sample=all --device=/dev/null:/dev/outsidenull:rwm busybox
  cid="$output"
  run_buildah run "$cid" cat /dev/containers-cdi.yaml /dev/outsidenull
}

@test "run with CDI" {
  skip_if_chroot
  _prefetch busybox
  cdidir=${TEST_SCRATCH_DIR}/cdi-config-dir
  mkdir -p $cdidir
  sed -e s:@@hostcdipath@@:$cdidir:g $BUDFILES/cdi/containers-cdi.yaml > $cdidir/containers-cdi.yaml
  chmod 644 $cdidir/containers-cdi.yaml
  echo === Begin CDI configuration in $cdidir/containers-cdi.yaml ===
  cat $cdidir/containers-cdi.yaml
  echo === End CDI configuration ===
  run_buildah from $WITH_POLICY_JSON --security-opt label=disable busybox
  cid="$output"
  run_buildah run --cdi-config-dir=$cdidir --device=containers.github.io/sample=all "$cid" cat /dev/containers-cdi.yaml
}
