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

    sed "s/^label = true/label = false/g" ${TEST_SOURCES}/containers.conf > ${TEST_SCRATCH_DIR}/containers.conf
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah 1 --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"
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

    sed '/^devices.*/a "\/dev\/foo:\/dev\/foo1:rmw",' ${TEST_SOURCES}/containers.conf > ${TEST_SCRATCH_DIR}/containers.conf
    rm -f /dev/foo; mknod /dev/foo c 1 1
    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah from --quiet $WITH_POLICY_JSON alpine
    cid="$output"
    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah  --log-level=error run $cid ls /dev/foo1
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

    sed "/AUDIT_WRITE/d" ${TEST_SOURCES}/containers.conf > ${TEST_SCRATCH_DIR}/containers.conf
    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah from --quiet $WITH_POLICY_JSON alpine
    cid="$output"

    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah --log-level=error run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
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

    ln -s /usr/bin/crun ${TEST_SCRATCH_DIR}/runtime

    cat >${TEST_SCRATCH_DIR}/containers.conf << EOF
[engine]
runtime = "nonstandard_runtime_name"
[engine.runtimes]
nonstandard_runtime_name = ["${TEST_SCRATCH_DIR}/runtime"]
EOF

    _prefetch alpine
    cid=$(buildah from $WITH_POLICY_JSON alpine)
    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah --log-level=error run $cid true
}

@test "containers.conf network sysctls" {
    if test "$BUILDAH_ISOLATION" = "chroot" ; then
        skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi

    cat >${TEST_SCRATCH_DIR}/containers.conf << EOF
[containers]
default_sysctls = [
  "net.ipv4.tcp_timestamps=123"
]
EOF
    _prefetch alpine
    cat >${TEST_SCRATCH_DIR}/Containerfile << _EOF
FROM alpine
RUN echo -n "timestamp="; cat /proc/sys/net/ipv4/tcp_timestamps
RUN echo -n "ping_group_range="; cat /proc/sys/net/ipv4/ping_group_range
_EOF

    run_buildah build ${TEST_SCRATCH_DIR}
    expect_output --substring "timestamp=1"
    expect_output --substring "ping_group_range=0.*0"

    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah build ${TEST_SCRATCH_DIR}
    expect_output --substring "timestamp=123"
    if is_rootless ; then
       expect_output --substring "ping_group_range=65534.*65534"
    else
       expect_output --substring "ping_group_range=1.*0"
    fi

}


@test "containers.conf retry" {
    cat >${TEST_SCRATCH_DIR}/containers.conf << EOF
[engine]
retry=10
retry_delay="5s"
EOF
    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah build --help
    expect_output --substring "retry.*\(default 10\)"
    expect_output --substring "retry-delay.*\(default \"5s\"\)"

    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah push --help
    expect_output --substring "retry.*\(default 10\)"
    expect_output --substring "retry-delay.*\(default \"5s\"\)"

    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah pull --help
    expect_output --substring "retry.*\(default 10\)"
    expect_output --substring "retry-delay.*\(default \"5s\"\)"

    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah from --help
    expect_output --substring "retry.*\(default 10\)"
    expect_output --substring "retry-delay.*\(default \"5s\"\)"

    CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf run_buildah manifest push --help
    expect_output --substring "retry.*\(default 10\)"
    expect_output --substring "retry-delay.*\(default \"5s\"\)"
}
