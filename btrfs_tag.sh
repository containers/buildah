#!/usr/bin/env bash
cc -E - > /dev/null 2> /dev/null << EOF
#include <btrfs/version.h>
EOF
if test $? -ne 0 ; then
	echo btrfs_noversion
fi
