#!/bin/bash
export PATH=${GOPATH%%:*}/bin:${PATH}
if ! which git-validation > /dev/null 2> /dev/null ; then
	echo git-validation is not installed.
	echo Try installing it with \"make install.tools\" or with
	echo \"go get -u github.com/vbatts/git-validation\"
	exit 1
fi
if test "$TRAVIS" != true ; then
	#GITVALIDATE_EPOCH=":/git-validation epoch"
	GITVALIDATE_EPOCH="5fa3c0bd3fbceca38b6ad2f8a893bc918a8a514a"
fi
exec git-validation -q -run DCO,short-subject ${GITVALIDATE_EPOCH:+-range "${GITVALIDATE_EPOCH}""..${GITVALIDATE_TIP:-@}"} ${GITVALIDATE_FLAGS}
