#!/bin/bash
# Generate a digest in an effort to create something that looks like headers in
# https://www.openssl.org/source/openssl-1.1.1g.tar.gz, per #2717.
comment=$(sha1sum content.txt)
comment="${comment// *}"
# Expects GNU tar.
gtar --pax-option=globexthdr.comment="$comment" -czf content.tar.gz content.txt
