#!/usr/bin/env bash
set -e

# Check only tracked files (ignore untracked files like build artifacts)
STATUS=$(git status --porcelain | grep -v '^??' || true)
if [[ -z $STATUS ]]
then
	echo "tree is clean"
else
	echo "tree is dirty, please commit all changes and sync the vendor.conf"
	echo ""
	echo "$STATUS"
	git diff
	exit 1
fi
