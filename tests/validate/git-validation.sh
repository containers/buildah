#!/bin/bash
export PATH=tests/tools/build:${PATH}
if ! which git-validation > /dev/null 2> /dev/null ; then
	echo git-validation is not in PATH.
	echo Make sure to call this script from the project root.
	exit 1
fi
if test "$TRAVIS" != true ; then
	#GITVALIDATE_EPOCH=":/git-validation epoch"
	GITVALIDATE_EPOCH="8b1d11f52721ab062c587abc711b40a6e1a1f7dc"
fi
exec git-validation -q -run DCO,short-subject ${GITVALIDATE_EPOCH:+-range "${GITVALIDATE_EPOCH}""..${GITVALIDATE_TIP:-@}"} ${GITVALIDATE_FLAGS}
