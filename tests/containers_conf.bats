#!/usr/bin/env bats

load helpers

@test "containers.conf env test" {
    cid=$(buildah --containers-conf ${TESTSDIR}/containers.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error --containers-conf ${TESTSDIR}/containers.conf run $cid sh -c 'env | grep "foo=bar"'
}

@test "containers.conf selinux test" {
    if ! which selinuxenabled > /dev/null 2> /dev/null ; then
        skip "No selinuxenabled"
    elif ! selinuxenabled ; then
        skip "selinux is disabled"
    fi


    cid=$(buildah --containers-conf ${TESTSDIR}/containers.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error --containers-conf ${TESTSDIR}/containers.conf run $cid sh -c "cat /proc/self/attr/current | grep container_t"

    buildah rm $cid

    sed "s/^selinux = true/selinux = false/g" ${TESTSDIR}/containers.conf > ${TESTSDIR}/containers1.conf
    cid=$(buildah --containers-conf ${TESTSDIR}/containers1.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah 1 --log-level=error --containers-conf ${TESTSDIR}/containers1.conf run $cid sh -c "cat /proc/self/attr/current | grep container_t"
    rm ${TESTSDIR}/containers1.conf
}

@test "containers.conf ulimit test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi


    cid=$(buildah --containers-conf ${TESTSDIR}/containers.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error --containers-conf ${TESTSDIR}/containers.conf run $cid  awk '/open files/{print $4}' /proc/self/limits
	expect_output "500" "limits: open files (w/file limit)"

    cid=$(buildah from --pull --ulimit nofile=300:400 --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	run_buildah --log-level=error run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file limit)"
}

@test "containers.conf additional devices test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
        skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
    fi
    cid=$(buildah --containers-conf ${TESTSDIR}/containers.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah 1 --log-level=error --containers-conf ${TESTSDIR}/containers.conf run $cid ls /dev/fuse1
    buildah rm $cid

    sed '/additional_devices.*/a "\/dev\/foo:\/dev\/fuse1:rmw",' ${TESTSDIR}/containers.conf > ${TESTSDIR}/containers1.conf
    mknod /dev/foo c 1 3
    cid=$(buildah --containers-conf ${TESTSDIR}/containers1.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah  --log-level=error --containers-conf ${TESTSDIR}/containers1.conf run $cid ls /dev/fuse1
    rm ${TESTSDIR}/containers1.conf
}

@test "containers.conf capabilities test" {
    cid=$(buildah --containers-conf ${TESTSDIR}/containers.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error --containers-conf ${TESTSDIR}/containers.conf run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
    CapEff=$output
    expect_output "00000000280425fb"
    buildah rm $cid

    sed "/AUDIT_WRITE/d" ${TESTSDIR}/containers.conf > ${TESTSDIR}/containers1.conf
    cid=$(buildah --containers-conf ${TESTSDIR}/containers1.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)

    run_buildah --log-level=error --containers-conf ${TESTSDIR}/containers1.conf run $cid sh -c 'grep  CapEff /proc/self/status | cut -f2'
    buildah rm $cid

    test "$output" != "$CapEff"
    rm ${TESTSDIR}/containers1.conf
}

@test "containers.conf /dev/shm test" {
    if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi

    cid=$(buildah --containers-conf ${TESTSDIR}/containers.conf from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
    run_buildah --log-level=error --containers-conf ${TESTSDIR}/containers.conf run $cid sh -c 'df /dev/shm | awk '\''/shm/{print $4}'\'''
    expect_output "200"
}