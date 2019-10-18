#!/usr/bin/env bats

load helpers

@test "containers.conf env test" {
    export HOME=$TESTDIR
    CONFIGDIR=$HOME/.config/containers
    mkdir -p $CONFIGDIR
    cp containers.conf $CONFIGDIR

    # check env foo=bar from containers.conf
    cid=$(buildah from --pull=true docker.io/alpine)
    run_buildah --log-level=error run $cid env | grep "foo=bar"

}

@test "containers.conf selinux test" {
    if ! which selinuxenabled > /dev/null 2> /dev/null ; then
        skip "No selinuxenabled"
    elif ! selinuxenabled ; then
        skip "selinux is disabled"
    fi
    export HOME=$TESTDIR
    CONFIGDIR=$HOME/.config/containers
    mkdir -p $CONFIGDIR
    cp containers.conf $CONFIGDIR

    cid=$(buildah from --pull=true docker.io/alpine)
    run_buildah --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"
    buildah rm $cid


    sed "s/^selinux = true/selinux = false/g" -i $CONFIGDIR/containers.conf
    cid=$(buildah from --pull=true docker.io/alpine)
    run_buildah 1 --log-level=error run $cid sh -c "cat /proc/self/attr/current | grep container_t"
}

@test "containers.conf ulimit test" {
    export HOME=$TESTDIR
    CONFIGDIR=$HOME/.config/containers
    mkdir -p $CONFIGDIR
    cp containers.conf $CONFIGDIR

    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	run_buildah --log-level=error run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "500" "limits: open files (w/file limit)"

    cid=$(buildah from --pull --ulimit nofile=300:400 --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	run_buildah --log-level=error run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file limit)"
}

@test "containers.conf additional devices test" {
    if [ $UID != 0 ]; then
        skip "requires root"
    fi
    export HOME=$TESTDIR
    CONFIGDIR=$HOME/.config/containers
    mkdir -p $CONFIGDIR
    cp containers.conf $CONFIGDIR
    #  TODO: root does not read $HOME/.config/containers
    # cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	# run_buildah --log-level=error run $cid ls /dev/fuse
    # buildah rm $cid

    # sed "/\/dev\/fuse/d" $CONFIGDIR/containers.conf
    # cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	# run_buildah 1 --log-level=error run $cid ls /dev/fuse
}

@test "containers.conf capabilities test" {
    export HOME=$TESTDIR
    CONFIGDIR=$HOME/.config/containers
    mkdir -p $CONFIGDIR
    cp containers.conf $CONFIGDIR

    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	run_buildah run $cid  grep CapEff /proc/self/status
    CapEff=$output
    buildah rm $cid

    sed "/AUDIT_WRITE/d" $CONFIGDIR/containers.conf
    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	run_buildah run $cid  grep CapEff /proc/self/status
    test "$output" != "$CapEff"
}

@test "containers.conf /dev/shm test" {
    export HOME=$TESTDIR
    CONFIGDIR=$HOME/.config/containers
    mkdir -p $CONFIGDIR
    cp containers.conf $CONFIGDIR

    cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine)
	output=$(buildah run $cid df /dev/shm | awk '/shm/{print $4}')
    [ $output -eq 100 ]
}