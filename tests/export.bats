#!/usr/bin/env bats

load helpers

@test "extract" {
  touch ${TESTDIR}/reference-time-file
  for source in scratch alpine; do
    cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json ${source})
    mnt=$(buildah mount $cid)
    touch ${mnt}/export.file
    tar -cf - --transform s,^./,,g -C ${mnt} . | tar tf - | grep -v "^./$" | sort > ${TESTDIR}/tar.output
    buildah umount $cid
    buildah export "$cid" > ${TESTDIR}/${source}.tar
    buildah export -o ${TESTDIR}/${source}1.tar "$cid"
    diff ${TESTDIR}/${source}.tar ${TESTDIR}/${source}1.tar
    tar -tf ${TESTDIR}/${source}.tar | sort > ${TESTDIR}/export.output
    diff ${TESTDIR}/tar.output ${TESTDIR}/export.output
    rm -f ${TESTDIR}/tar.output ${TESTDIR}/export.output
    rm -f ${TESTDIR}/${source}1.tar ${TESTDIR}/${source}.tar
    buildah rm "$cid"
  done
}
