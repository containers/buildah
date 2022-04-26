#!/usr/bin/env bats

load helpers

@test "bind secrets mounts to container" {
    skip_if_no_runtime

    # Setup
    SECRETS_DIR=$TEST_SCRATCH_DIR/rhel/secrets
    mkdir -p $SECRETS_DIR

    TESTFILE1=$SECRETS_DIR/test.txt
    TESTFILE_CONTENT="Testing secrets mounts. I am mounted!"
    echo $TESTFILE_CONTENT > $TESTFILE1

    TESTFILE2=$SECRETS_DIR/file.txt
    touch     $TESTFILE2
    chmod 604 $TESTFILE2

    TEST_SCRATCH_DIR1=$SECRETS_DIR/test-dir
    mkdir -m704 $TEST_SCRATCH_DIR1

    TESTFILE3=$TEST_SCRATCH_DIR1/file.txt
    touch     $TESTFILE3
    chmod 777 $TESTFILE3

    mkdir -p $TEST_SCRATCH_DIR/symlink/target
    touch    $TEST_SCRATCH_DIR/symlink/target/key.pem
    ln -s    $TEST_SCRATCH_DIR/symlink/target $SECRETS_DIR/mysymlink

    # prepare the test mounts file
    mkdir $TEST_SCRATCH_DIR/containers
    MOUNTS_PATH=$TEST_SCRATCH_DIR/containers/mounts.conf

    # add the mounts entries
    echo "$SECRETS_DIR:/run/secrets"  > $MOUNTS_PATH
    echo "$SECRETS_DIR"              >> $MOUNTS_PATH
    echo "$TESTFILE1:/test.txt"      >> $MOUNTS_PATH


    # setup the test container
    _prefetch alpine
    run_buildah --default-mounts-file "$MOUNTS_PATH" \
		from --quiet --pull $WITH_POLICY_JSON alpine
    cid=$output

    # test a standard mount to /run/secrets
    run_buildah run $cid ls /run/secrets
    expect_output --substring "test.txt"

    # test a mount without destination
    run_buildah run $cid ls "$TEST_SCRATCH_DIR"/rhel/secrets
    expect_output --substring "test.txt"

    # test a file-based mount
    run_buildah run $cid cat /test.txt
    expect_output "$TESTFILE_CONTENT"

    # test permissions for a file-based mount
    run_buildah run $cid stat -c %a /run/secrets/file.txt
    expect_output 604

    # test permissions for a directory-based mount
    run_buildah run $cid stat -c %a /run/secrets/test-dir
    expect_output 704

    # test permissions for a file-based mount within a sub-directory
    run_buildah run $cid stat -c %a /run/secrets/test-dir/file.txt
    expect_output 777

    cat > $TEST_SCRATCH_DIR/Containerfile << _EOF
from alpine
run stat -c %a /run/secrets/file.txt
run stat -c %a /run/secrets/test-dir
run stat -c %a /run/secrets/test-dir/file.txt
_EOF

    run_buildah --default-mounts-file "$MOUNTS_PATH" bud $TEST_SCRATCH_DIR
    expect_output --substring "604"
    expect_output --substring "704"
    expect_output --substring "777"

    # test a symlink
    run_buildah run $cid ls /run/secrets/mysymlink
    expect_output --substring "key.pem"
}
