#!/usr/bin/env bats

load helpers

@test "containers.conf selinux test" {
    if ! which selinuxenabled > /dev/null 2> /dev/null ; then
        skip "No selinuxenabled executable"
    elif ! selinuxenabled ; then
        skip "selinux is disabled"
    fi

    _prefetch alpine
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    run_buildah --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"

    run_buildah rm $cid

    sed "s/^label = true/label = false/g" ${TESTSDIR}/containers.conf > ${TESTDIR}/containers.conf
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    CONTAINERS_CONF=${TESTDIR}/containers.conf run_buildah 1 --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"
}

@test "containers.conf ulimit test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
        skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi

    _prefetch alpine
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    run_buildah --log-level=error run $cid  awk '/open files/{print $4}' /proc/self/limits
    expect_output "500" "limits: open files (w/file limit)"

    cid=$(buildah from --ulimit nofile=300:400 $WITH_POLICY_JSON alpine)
    run_buildah --log-level=error run $cid awk '/open files/{print $4}' /proc/self/limits
    expect_output "300" "limits: open files (w/file limit)"
}

@test "containers.conf additional devices test" {
    skip_if_rootless_environment
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
        skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi

    _prefetch alpine
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    CONTAINERS_CONF=$CONTAINERS_CONF run_buildah 1 --log-level=error run $cid ls /dev/foo1
    run_buildah rm $cid

    sed '/^devices.*/a "\/dev\/foo:\/dev\/foo1:rmw",' ${TESTSDIR}/containers.conf > ${TESTDIR}/containers.conf
    rm -f /dev/foo; mknod /dev/foo c 1 1
    CONTAINERS_CONF=${TESTDIR}/containers.conf run_buildah from --quiet $WITH_POLICY_JSON alpine
    cid="$output"
    CONTAINERS_CONF=${TESTDIR}/containers.conf run_buildah  --log-level=error run $cid ls /dev/foo1
    rm -f /dev/foo
}

@test "containers.conf capabilities test" {
    _prefetch alpine

    run_buildah from --quiet $WITH_POLICY_JSON alpine
    cid="$output"
    run_buildah --log-level=error run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
    CapEff="$output"
    expect_output "00000000a80425fb"
    run_buildah rm $cid

    sed "/AUDIT_WRITE/d" ${TESTSDIR}/containers.conf > ${TESTDIR}/containers.conf
    CONTAINERS_CONF=${TESTDIR}/containers.conf run_buildah from --quiet $WITH_POLICY_JSON alpine
    cid="$output"

    CONTAINERS_CONF=${TESTDIR}/containers.conf run_buildah --log-level=error run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
    run_buildah rm $cid

    test "$output" != "$CapEff"
}

@test "containers.conf /dev/shm test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
        skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi

    _prefetch alpine
    run_buildah from --quiet $WITH_POLICY_JSON alpine
    cid="$output"
    run_buildah --log-level=error run $cid sh -c 'df /dev/shm | awk '\''/shm/{print $4}'\'''
    expect_output "200"
}

@test "containers.conf custom runtime" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
        skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi

    test -x /usr/bin/crun || skip "/usr/bin/crun doesn't exist"

    ln -s /usr/bin/crun ${TESTDIR}/runtime

    cat >${TESTDIR}/containers.conf << EOF
[engine]
runtime = "nonstandard_runtime_name"
[engine.runtimes]
nonstandard_runtime_name = ["${TESTDIR}/runtime"]
EOF

    _prefetch alpine
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    CONTAINERS_CONF=${TESTDIR}/containers.conf run_buildah --log-level=error run $cid true
}
