#!/usr/bin/env bats

load helpers

function setup() {
    mkdir $TESTSDIR/containers
    touch $TESTSDIR/mounts.conf
    MOUNTS_PATH=$TESTSDIR/containers/mounts.conf
    echo "$TESTSDIR/rhel/secrets:/run/secrets" > $MOUNTS_PATH

    mkdir $TESTSDIR/rhel
    mkdir $TESTSDIR/rhel/secrets
    touch $TESTSDIR/rhel/secrets/test.txt
    echo "Testing secrets mounts. I am mounted!" > $TESTSDIR/rhel/secrets/test.txt
}

@test "bind secrets mounts to container" {
    if ! which runc ; then
		skip
    fi
    runc --version
    cid=$(buildah --default-mounts-file "$MOUNTS_PATH" --debug=false from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah --debug=false run $cid ls /run
    echo "$output"
    [ "$status" -eq 0 ]
    mounts="$output"
    run grep "secrets" <<< "$mounts"
    echo "$output"
    [ "$status" -eq 0 ]
    buildah rm $cid
    rm -rf $TESTSDIR/containers
    rm -rf $TESTSDIR/rhel
}
