#!/usr/bin/env bats

load helpers

@test "import" {
  touch ${TESTDIR}/reference-time-file
  for source in scratch alpine; do
    cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json ${source})
    mnt=$(buildah mount $cid)
    touch ${mnt}/export.file
    tar -cf - --transform s,^./,,g -C ${mnt} . | tar tf - | grep -v "^./$" | sort > ${TESTDIR}/tar.output
    buildah umount $cid
    buildah export "$cid" > ${TESTDIR}/${source}.tar
    buildah import --signature-policy ${TESTSDIR}/policy.json -c "ARCH X86_64" -c "ENV foo=bar" -c "WORKINGDIR /tmp" -c "label This is a test" -c "entrypoint /usr/sbin/entrypoint" -c "CMD /usr/sbin/cmd --debug 5" -c "ENV foo=bar" -c "MAINTAINER Daniel Boon" -c "USER Daniel" -c "VOLUME /var" -c "OS Fedora" -m "This is a test message" ${TESTDIR}/${source}.tar ${source}_import
    rm -f ${TESTDIR}/${source}.tar
    buildah rmi ${source}_import
    buildah rm "$cid"
  done
}
