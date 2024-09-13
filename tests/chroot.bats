#!/usr/bin/env bats

load helpers

@test chroot-mount-flags {
  skip_if_no_unshare
  if ! test -e /etc/subuid ; then
    skip "we can't bind mount over /etc/subuid during the test if there is no /etc/subuid file"
  fi
  if ! test -e /etc/subgid ; then
    skip "we can't bind mount over /etc/subgid during the test if there is no /etc/subgid file"
  fi
  # whom should we map to root in a nested namespace?
  if is_rootless ; then
    subid=128
    rangesize=1024
  else
    subid=1048576
    rangesize=16384
  fi
  # we're going to have to prefetch into storage used by someone else image
  # chosen because its rootfs doesn't have any uid/gid ownership above
  # $rangesize, because the nested namespace needs to be able to represent all
  # of them
  baseimage=registry.access.redhat.com/ubi9-micro:latest
  _prefetch $baseimage
  baseimagef=$(tr -c a-zA-Z0-9.- - <<< "$baseimage")
  # create the directories that we need
  tmpfs=${TEST_SCRATCH_DIR}/tmpfs
  mkdir $tmpfs
  context=${TEST_SCRATCH_DIR}/context
  mkdir $context
  storagedir=${TEST_SCRATCH_DIR}/storage
  mkdir $storagedir
  rootdir=${storagedir}/rootdir
  mkdir $rootdir
  runrootdir=${storagedir}/runrootdir
  mkdir $runrootdir
  xdgruntimedir=${storagedir}/xdgruntime
  mkdir $xdgruntimedir
  xdgconfighome=${storagedir}/xdgconfighome
  mkdir $xdgconfighome
  xdgdatahome=${storagedir}/xdgdatahome
  mkdir $xdgdatahome
  storageopts="--storage-driver vfs --root $rootdir --runroot $runrootdir"
  # our temporary parent directory might not be world-searchable, which will
  # cause someone in the nested user namespace to hit permissions issues even
  # looking for $storagedir, so tweak perms to let them do at least that much
  fixupdir=$storagedir
  while test $(stat -c %d:%i $fixupdir) != $(stat -c %d:%i /) ; do
    # walk up to root, or the first parent that we don't own
    if test $(stat -c %u $fixupdir) -ne $(id -u) ; then
      break
    fi
    chmod +x $fixupdir
    fixupdir=$fixupdir/..
  done
  # start writing the script to run in the nested user namespace
  cp -v ${TEST_SOURCES}/containers.conf ${TEST_SCRATCH_DIR}/containers.conf
  chmod ugo+r ${TEST_SCRATCH_DIR}/containers.conf
  echo set -e > ${TEST_SCRATCH_DIR}/script.sh
  echo export XDG_RUNTIME_DIR=$xdgruntimedir >> ${TEST_SCRATCH_DIR}/script.sh
  echo export XDG_CONFIG_HOME=$xdgconfighome >> ${TEST_SCRATCH_DIR}/script.sh
  echo export XDG_DATA_HOME=$xdgdatahome >> ${TEST_SCRATCH_DIR}/script.sh
  echo export CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf >> ${TEST_SCRATCH_DIR}/script.sh
  # give our would-be user ownership of that directory
  echo chown --recursive ${subid}:${subid} ${storagedir} >> ${TEST_SCRATCH_DIR}/script.sh
  # make newuidmap/newgidmap, invoked by unshare even for uid=0, happy
  echo root:0:4294967295 > ${TEST_SCRATCH_DIR}/subid
  echo mount --bind -r ${TEST_SCRATCH_DIR}/subid /etc/subuid >> ${TEST_SCRATCH_DIR}/script.sh
  echo mount --bind -r ${TEST_SCRATCH_DIR}/subid /etc/subgid >> ${TEST_SCRATCH_DIR}/script.sh
  # don't get tripped up by ${TEST_SCRATCH_DIR} potentially being on a filesystem with non-default mount flags
  echo mount -t tmpfs -o size=256K tmpfs $tmpfs >> ${TEST_SCRATCH_DIR}/script.sh
  # mount a small tmpfs with every mount flag combination that concerns us, and
  # be ready to tell buildah to mount everything conservatively, to mirror the
  # TransientMounts API being used to nodev/noexec/nosuid/ro bind in a source
  # that doesn't necessarily have those flags already set on it
  for d in dev nodev ; do
    for e in exec noexec ; do
      for s in suid nosuid ; do
        for r in ro rw ; do
          subdir=$tmpfs/d-$d-$e-$s-$r
          echo mkdir ${subdir} >> ${TEST_SCRATCH_DIR}/script.sh
          echo mount -t tmpfs -o size=256K,$d,$e,$s,$r tmpfs ${subdir} >> ${TEST_SCRATCH_DIR}/script.sh
          mounts="${mounts:+${mounts} }--volume ${subdir}:/mounts/d-$d-$e-$s-$r:nodev,noexec,nosuid,ro"
        done
      done
    done
  done
  # make sure that RUN doesn't just break when we try to use volume mounts with
  # flags set that we're not allowed to modify
  echo FROM $baseimage > $context/Dockerfile
  echo RUN cat /proc/mounts >> $context/Dockerfile
  # copy in the prefetched image
  # unshare from util-linux 2.39 also accepts INNER:OUTER:SIZE for --map-users
  # and --map-groups, but fedora 37's is too old, so the older OUTER,INNER,SIZE
  # (using commas instead of colons as field separators) will have to do
  echo "env | sort" >> ${TEST_SCRATCH_DIR}/script.sh
  echo "env _CONTAINERS_USERNS_CONFIGURED=done unshare -Umpf --mount-proc --setuid 0 --setgid 0 --map-users=${subid},0,${rangesize} --map-groups=${subid},0,${rangesize} ${COPY_BINARY} ${storageopts} dir:$_BUILDAH_IMAGE_CACHEDIR/$baseimagef containers-storage:$baseimage" >> ${TEST_SCRATCH_DIR}/script.sh
  # try to do a build with all of the volume mounts
  echo "env _CONTAINERS_USERNS_CONFIGURED=done unshare -Umpf --mount-proc --setuid 0 --setgid 0 --map-users=${subid},0,${rangesize} --map-groups=${subid},0,${rangesize} ${BUILDAH_BINARY} ${BUILDAH_REGISTRY_OPTS} ${storageopts} build --isolation chroot --pull=never $mounts $context" >> ${TEST_SCRATCH_DIR}/script.sh
  # run that whole script in a nested mount namespace with no $XDG_...
  # variables leaked into it
  if is_rootless ; then
    run_buildah unshare env -i bash -x ${TEST_SCRATCH_DIR}/script.sh
  else
    unshare -mpf --mount-proc env -i bash -x ${TEST_SCRATCH_DIR}/script.sh
  fi
}
