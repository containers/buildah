#!/usr/bin/env bats

load helpers

function setup() {
    # prepare the test mounts file
    mkdir $TESTSDIR/containers
    touch $TESTSDIR/containers/mounts.conf
    MOUNTS_PATH=$TESTSDIR/containers/mounts.conf

    # add the mounts entries
    echo "$TESTSDIR/rhel/secrets:/run/secrets" > $MOUNTS_PATH
    echo "$TESTSDIR/rhel/secrets" >> $MOUNTS_PATH
    echo "$TESTSDIR/rhel/secrets/test.txt:/test.txt" >> $MOUNTS_PATH

    # create the files to be tested
    mkdir -p $TESTSDIR/rhel/secrets
    TESTFILE=$TESTSDIR/rhel/secrets/test.txt
    touch $TESTFILE

    TESTFILE_CONTENT="Testing secrets mounts. I am mounted!"
    echo $TESTFILE_CONTENT > $TESTSDIR/rhel/secrets/test.txt

    mkdir -p $TESTSDIR/symlink/target
    touch $TESTSDIR/symlink/target/key.pem
    ln -s $TESTSDIR/symlink/target $TESTSDIR/rhel/secrets/mysymlink
}

function teardown() {
    buildah rm $cid
    rm -rf $TESTSDIR/containers
    rm -rf $TESTSDIR/rhel
    rm -rf $TESTSDIR/symlink
    for d in containers rhel symlink;do
        rm -rf $TESTSDIR/$d
    done
}

@test "bind secrets mounts to container" {
    if ! which runc ; then
        skip "no runc in PATH"
    fi
    runc --version


    # setup the test container
    cid=$(buildah --default-mounts-file "$MOUNTS_PATH" --log-level=error \
        from --pull --signature-policy ${TESTSDIR}/policy.json alpine)

    # test a standard mount to /run/secrets
    run_buildah --log-level=error run $cid ls /run/secrets
    expect_output --substring "test.txt"

    # test a mount without destination
    run_buildah --log-level=error run $cid ls "$TESTSDIR"/rhel/secrets
    expect_output --substring "test.txt"

    # test a file-based mount
    run_buildah --log-level=error run $cid cat /test.txt
    expect_output --substring $TESTFILE_CONTENT

    # test a symlink
    run_buildah run $cid ls /run/secrets/mysymlink
    expect_output --substring "key.pem"
}
