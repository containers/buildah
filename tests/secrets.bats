#!/usr/bin/env bats

load helpers

function setup() {
    mkdir $TESTSDIR/containers
    touch $TESTSDIR/containers/mounts.conf
    MOUNTS_PATH=$TESTSDIR/containers/mounts.conf
    echo "$TESTSDIR/rhel/secrets:/run/secrets" > $MOUNTS_PATH

    mkdir -p $TESTSDIR/rhel/secrets
    touch $TESTSDIR/rhel/secrets/test.txt
    echo "Testing secrets mounts. I am mounted!" > $TESTSDIR/rhel/secrets/test.txt
    mkdir -p $TESTSDIR/symlink/target
    touch $TESTSDIR/symlink/target/key.pem
    ln -s $TESTSDIR/symlink/target $TESTSDIR/rhel/secrets/mysymlink
}

function teardown() {
    for d in containers rhel symlink;do
        rm -rf $TESTSDIR/$d
    done
}

@test "bind secrets mounts to container" {
    if ! which runc ; then
		skip "no runc in PATH"
    fi
    runc --version
    cid=$(buildah --default-mounts-file "$MOUNTS_PATH" --debug=false from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run_buildah --debug=false run $cid ls /run/secrets
    expect_output --substring "test.txt"
    run_buildah --debug run $cid ls /run/secrets/mysymlink
    expect_output --substring "key.pem"
    buildah rm $cid
    rm -rf $TESTSDIR/containers
    rm -rf $TESTSDIR/rhel
    rm -rf $TESTSDIR/symlink
}
