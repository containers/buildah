#!/bin/bash
#
# tests for helpers.bash
#

. $(dirname ${BASH_SOURCE})/helpers.bash

INDEX=1
RC=0

# t (true) : tests that should pass
function t() {
    result=$(assert "$@" 2>&1)
    status=$?

    if [[ $status -eq 0 ]]; then
        echo "ok $INDEX $*"
    else
        echo "not ok $INDEX $*"
        echo "$result"
        RC=1
    fi

    INDEX=$((INDEX + 1))
}

# f (false) : tests that should fail
function f() {
    result=$(assert "$@" 2>&1)
    status=$?

    if [[ $status -ne 0 ]]; then
        echo "ok $INDEX ! $*"
    else
        echo "not ok $INDEX ! $*  [passed, should have failed]"
        RC=1
    fi

    INDEX=$((INDEX + 1))
}



t "" = ""
t "a" != ""
t "" != "a"

t "a" = "a"
t "aa" == "aa"
t "a[b]{c}" = "a[b]{c}"

t "abcde"  =~ "a"
t "abcde"  =~  "b"
t "abcde"  =~   "c"
t "abcde"  =~    "d"
t "abcde"  =~     "e"
t "abcde"  =~ "ab"
t "abcde"  =~ "abc"
t "abcde"  =~ "abcd"
t "abcde"  =~  "bcde"
t "abcde"  =~   "cde"
t "abcde"  =~    "de"

t    "foo"    =~ "foo"
t    "foobar" =~ "foo"
t "barfoo"    =~ "foo"

t 'a "AB \"CD": ef' =  'a "AB \"CD": ef'
t 'a "AB \"CD": ef' =~ 'a "AB \\"CD": ef'

t 'abcdef' !~ 'efg'
t 'abcdef' !~ 'x'

###########

f "a" = "b"
f "a" == "b"

f "abcde" =~ "x"

f "abcde" !~ "a"
f "abcde" !~ "ab"
f "abcde" !~ "abc"

f "" != ""

exit $RC
