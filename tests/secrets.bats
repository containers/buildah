#!/usr/bin/env bats

load helpers

function setup() {
    # create the files and directories to be tested
    SECRETS_DIR=$TESTSDIR/rhel/secrets
    mkdir -p $SECRETS_DIR

    TESTFILE1=$SECRETS_DIR/test.txt
    touch $TESTFILE1
    TESTFILE_CONTENT="Testing secrets mounts. I am mounted!"
    echo $TESTFILE_CONTENT > $TESTFILE1

    TESTFILE2=$SECRETS_DIR/file.txt
    touch $TESTFILE2
    chmod 604 $TESTFILE2

    TESTDIR1=$SECRETS_DIR/test-dir
    mkdir -m704 $TESTDIR1

    TESTFILE3=$TESTDIR1/file.txt
    touch $TESTFILE3
    chmod 777 $TESTFILE3

    mkdir -p $TESTSDIR/symlink/target
    touch $TESTSDIR/symlink/target/key.pem
    ln -s $TESTSDIR/symlink/target $SECRETS_DIR/mysymlink

    # prepare the test mounts file
    mkdir $TESTSDIR/containers
    touch $TESTSDIR/containers/mounts.conf
    MOUNTS_PATH=$TESTSDIR/containers/mounts.conf

    # add the mounts entries
    echo "$SECRETS_DIR:/run/secrets" > $MOUNTS_PATH
    echo "$SECRETS_DIR" >> $MOUNTS_PATH
    echo "$TESTFILE1:/test.txt" >> $MOUNTS_PATH
}

function teardown() {
    run_buildah rm $cid
    rm -rf $TESTSDIR/containers
    rm -rf $TESTSDIR/rhel
    rm -rf $TESTSDIR/symlink
    for d in containers rhel symlink;do
        rm -rf $TESTSDIR/$d
    done
}

@test "bind secrets mounts to container" {
    skip_if_no_runtime

    runc --version


    # setup the test container
    cid=$(buildah --default-mounts-file "$MOUNTS_PATH" \
        from --pull --signature-policy ${TESTSDIR}/policy.json alpine)

    # test a standard mount to /run/secrets
    run_buildah run $cid ls /run/secrets
    expect_output --substring "test.txt"

    # test a mount without destination
    run_buildah run $cid ls "$TESTSDIR"/rhel/secrets
    expect_output --substring "test.txt"

    # test a file-based mount
    run_buildah run $cid cat /test.txt
    expect_output --substring $TESTFILE_CONTENT

    # test permissions for a file-based mount
    run_buildah run $cid stat -c %a /run/secrets/file.txt
    expect_output --substring 604

    # test permissions for a directory-based mount
    run_buildah run $cid stat -c %a /run/secrets/test-dir
    expect_output --substring 704

    # test permissions for a file-based mount within a sub-directory
    run_buildah run $cid stat -c %a /run/secrets/test-dir/file.txt
    expect_output --substring 777

    # test a symlink
    run_buildah run $cid ls /run/secrets/mysymlink
    expect_output --substring "key.pem"
}
