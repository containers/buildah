#!/usr/bin/env bats

load helpers

@test "containers.conf selinux test" {
    if ! which selinuxenabled > /dev/null 2> /dev/null ; then
	skip "No selinuxenabled executable"
    elif ! selinuxenabled ; then
	skip "selinux is disabled"
    fi


    export CONTAINERS_CONF=${TESTSDIR}/containers.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"

    buildah rm $cid

    export CONTAINERS_CONF=${TESTSDIR}/containers1.conf
    sed "s/^label = true/label = false/g" ${TESTSDIR}/containers.conf > ${TESTSDIR}/containers1.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah 1 --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"
    rm ${TESTSDIR}/containers1.conf
}

@test "containers.conf ulimit test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi

    export CONTAINERS_CONF=${TESTSDIR}/containers.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error run $cid  awk '/open files/{print $4}' /proc/self/limits
	expect_output "500" "limits: open files (w/file limit)"

    cid=$(buildah from --pull --ulimit nofile=300:400 --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	run_buildah --log-level=error run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file limit)"
}

@test "containers.conf additional devices test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
	skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi
    export CONTAINERS_CONF=${TESTSDIR}/containers.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah 1 --log-level=error run $cid ls /dev/foo1
    buildah rm $cid

    sed '/^devices.*/a "/dev/foo:\/dev\/foo1:rmw",' ${TESTSDIR}/containers.conf > ${TESTSDIR}/containers1.conf
    rm -f /dev/foo; mknod /dev/foo c 1 1
    export CONTAINERS_CONF=${TESTSDIR}/containers1.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah  --log-level=error run $cid ls /dev/foo1
    rm -f /dev/foo
    rm ${TESTSDIR}/containers1.conf
}

@test "containers.conf capabilities test" {
    export CONTAINERS_CONF=${TESTSDIR}/containers.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
    CapEff=$output
    expect_output "00000000a80425fb"
    buildah rm $cid

    sed "/AUDIT_WRITE/d" ${TESTSDIR}/containers.conf > ${TESTSDIR}/containers1.conf
    export CONTAINERS_CONF=${TESTSDIR}/containers1.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)

    run_buildah --log-level=error run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
    buildah rm $cid

    test "$output" != "$CapEff"
    rm ${TESTSDIR}/containers1.conf
}

@test "containers.conf /dev/shm test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi

    export CONTAINERS_CONF=${TESTSDIR}/containers.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error run $cid sh -c 'df /dev/shm | awk '\''/shm/{print $4}'\'''
    expect_output "200"
}
