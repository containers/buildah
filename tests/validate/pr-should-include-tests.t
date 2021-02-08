#!/bin/bash
#
# tests for pr-should-include-tests.t
#
# FIXME: I don't think this will work in CI, because IIRC the git-checkout
# is a shallow one. But it works fine in a developer tree.
#
ME=$(basename $0)

###############################################################################
# BEGIN test cases
#
# Feel free to add as needed. Syntax is:
#    <exit status>  <sha of commit>  <branch>=<sha of merge base>  # comments
#
# Where:
#    exit status       is the expected exit status of the script
#    sha of merge base is the SHA of the branch point of the commit
#    sha of commit     is the SHA of a real commit in the podman repo
#
# We need the actual sha of the merge base because once a branch is
# merged 'git merge-base' (used in our test script) becomes useless.
#
#
# FIXME: as of 2021-01-07 we don't have "no tests needed" in our git
#        commit history, but once we do, please add a new '0' test here.
#
tests="
0  f466086d  88bc27df  PR 2955: two commits, includes tests
1  c5870ff8  520c7815  PR 2973: single commit, no tests
0  d460e2ed  371e4ca6  PR 2886: .cirrus.yml and contrib/cirrus/*
0  88bc27df  c5870ff8  PR 2972: vendor only
0  d4c696af  faa86c4f  PR 2470: CI:DOCS as well as only a .md change
0  d460e2ed  f52762a9  PR 2927: .md only, without CI:DOCS
"

# The script we're testing
test_script=$(dirname $0)/$(basename $0 .t)

# END   test cases
###############################################################################
# BEGIN test-script runner and status checker

function run_test_script() {
    local expected_rc=$1
    local testname=$2

    testnum=$(( testnum + 1 ))

    # DO NOT COMBINE 'local output=...' INTO ONE LINE. If you do, you lose $?
    local output
    output=$( $test_script )
    local actual_rc=$?

    if [[ $actual_rc != $expected_rc ]]; then
        echo "not ok $testnum $testname"
        echo "#  expected rc $expected_rc"
        echo "#  actual rc   $actual_rc"
        if [[ -n "$output" ]]; then
            echo "# script output: $output"
        fi
        rc=1
    else
        if [[ $expected_rc == 1 ]]; then
            # Confirm we get an error message
            if [[ ! "$output" =~ "Please write a regression test" ]]; then
                echo "not ok $testnum $testname"
                echo "# Expected: ~ 'Please write a regression test'"
                echo "# Actual:   $output"
                rc=1
            else
                echo "ok $testnum $testname"
            fi
        else
            echo "ok $testnum $testname"
        fi
    fi

    # If we expect an error, confirm that we can override it. We only need
    # to do this once.
    if [[ $expected_rc == 1 ]]; then
        if [[ -z "$tested_override" ]]; then
            testnum=$(( testnum + 1 ))

            CIRRUS_CHANGE_TITLE="[CI:DOCS] hi there" $test_script &>/dev/null
            if [[ $? -ne 0 ]]; then
                echo "not ok $testnum $rest (override with CI:DOCS)"
                rc=1
            else
                echo "ok $testnum $rest (override with CI:DOCS)"
            fi

            testnum=$(( testnum + 1 ))
            CIRRUS_CHANGE_MESSAGE="hi there [NO TESTS NEEDED] bye" $test_script &>/dev/null
            if [[ $? -ne 0 ]]; then
                echo "not ok $testnum $rest (override with '[NO TESTS NEEDED]')"
                rc=1
            else
                echo "ok $testnum $rest (override with '[NO TESTS NEEDED]')"
            fi

            tested_override=1
        fi
    fi
}

# END   test-script runner and status checker
###############################################################################
# BEGIN test-case parsing

rc=0
testnum=0
tested_override=

while read expected_rc parent_sha  commit_sha rest; do
    # Skip blank lines
    test -z "$expected_rc" && continue

    export DEST_BRANCH=$parent_sha
    export CIRRUS_CHANGE_IN_REPO=$commit_sha
    export CIRRUS_CHANGE_TITLE=$(git log -1 --format=%s $commit_sha)
    export CIRRUS_CHANGE_MESSAGE=

    run_test_script $expected_rc "$rest"
done <<<"$tests"

echo "1..$testnum"
exit $rc

# END   Test-case parsing
###############################################################################
