#!/usr/bin/env bats

load helpers

@test "chroot mount flags" {
  skip_if_no_unshare
  if ! test -e /etc/subuid ; then
    skip "we can't bind mount over /etc/subuid during the test if there is no /etc/subuid file"
  fi
  if ! test -e /etc/subgid ; then
    skip "we can't bind mount over /etc/subgid during the test if there is no /etc/subgid file"
  fi
  if is_rootless || ! have_cap_sys_admin ; then
    subid=128
    rangesize=1024
  else
    subid=1048576
    rangesize=16384
  fi
  # we're going to have to prefetch into storage used by someone else an image
  # chosen because its rootfs doesn't have any uid/gid ownership above
  # $rangesize, because the nested namespace needs to be able to represent all
  # of them
  baseimage=registry.access.redhat.com/ubi9-micro:latest
  _prefetch $baseimage
  baseimagef=$(tr -c a-zA-Z0-9.- - <<< "$baseimage")
  # create the directories that we need
  tmpfs=${TEST_SCRATCH_DIR}/tmpfs
  mkdir ${tmpfs}
  mkdir ${tmpfs}/oldroot
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
  local uidmapsize=$(awk 'BEGIN { uidmapsize = 0 } { uidmapsize = uidmapsize + $3 } END { print uidmapsize }' /proc/self/uid_map)
  local gidmapsize=$(awk 'BEGIN { gidmapsize = 0 } { gidmapsize = gidmapsize + $3 } END { print gidmapsize }' /proc/self/gid_map)
  echo root:0:${uidmapsize} > ${TEST_SCRATCH_DIR}/subuid
  echo root:0:${gidmapsize} > ${TEST_SCRATCH_DIR}/subgid
  echo mount --bind -r ${TEST_SCRATCH_DIR}/subuid /etc/subuid >> ${TEST_SCRATCH_DIR}/script.sh
  echo mount --bind -r ${TEST_SCRATCH_DIR}/subgid /etc/subgid >> ${TEST_SCRATCH_DIR}/script.sh
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
          for a in async sync dirsync ; do
            for t in noatime nodiratime lazytime relatime strictatime ; do
              for S in shared slave private ; do
                subdir=$tmpfs/d-$d-$e-$s-$r-$a-$t-$S
                echo mkdir ${subdir} >> ${TEST_SCRATCH_DIR}/script.sh
                echo mount -t tmpfs -o size=64K,$d,$e,$s,$r,$a,$t,$S tmpfs ${subdir} >> ${TEST_SCRATCH_DIR}/script.sh
                mounts="${mounts:+${mounts} }--volume ${subdir}:/mounts/d-$d-$e-$s-$r-$a-$t-$S:nodev,noexec,nosuid,ro"
                mounts="${mounts:+${mounts} }--volume ${subdir}:/mounts/r-$d-$e-$s-$r-$a-$t-$S:ro"
              done
            done
          done
        done
      done
    done
  done
  # copy binaries to a location where parent directory permissions are less
  # likely to interfere with running them from a different UID
  cp ${COPY_BINARY} ${TEST_SCRATCH_DIR}/copy
  cp ${BUILDAH_BINARY} ${TEST_SCRATCH_DIR}/buildah
  # make sure that RUN doesn't just break when we try to use volume mounts with
  # flags set that we're not allowed to modify
  echo FROM $baseimage > $context/Dockerfile
  echo RUN cat /proc/mounts >> $context/Dockerfile
  # have the image give us a litte information about its environment
  echo "env | sort" >> ${TEST_SCRATCH_DIR}/script.sh
  echo "cat /proc/self/mountinfo" >> ${TEST_SCRATCH_DIR}/script.sh
  # copy in the prefetched image so that a completely different user ($subid in
  # the current namespace (the current namespace's root is either root in the
  # parent namespace, or an unprivileged user who ran this test script)) can
  # try to use it, and use bind mounts from locations which the user doesn't
  # even own
  # unshare from util-linux 2.39 also accepts INNER:OUTER:SIZE for --map-users
  # and --map-groups, but fedora 37's is too old, so the older OUTER,INNER,SIZE
  # (using commas instead of colons as field separators) will have to do
  echo "env _CONTAINERS_USERNS_CONFIGURED=done unshare -Um --setuid 0 --setgid 0 --map-users=${subid},0,${rangesize} --map-groups=${subid},0,${rangesize} ${TEST_SCRATCH_DIR}/copy ${storageopts} dir:$_BUILDAH_IMAGE_CACHEDIR/$baseimagef containers-storage:$baseimage" >> ${TEST_SCRATCH_DIR}/script.sh
  # try to do a build with all of the volume mounts
  echo "env _CONTAINERS_USERNS_CONFIGURED=done unshare -Um --setuid 0 --setgid 0 --map-users=${subid},0,${rangesize} --map-groups=${subid},0,${rangesize} ${TEST_SCRATCH_DIR}/buildah ${BUILDAH_REGISTRY_OPTS} ${storageopts} build --isolation chroot --pull=never $mounts $context" >> ${TEST_SCRATCH_DIR}/script.sh
  # run that whole script in a nested mount namespace with no $XDG_...
  # variables leaked into it, with the invoking user as "root"
  if is_rootless ; then
    run_buildah unshare env -i bash -x ${TEST_SCRATCH_DIR}/script.sh
  else
    local unshareflags="-m --setuid 0 --setgid 0"
    if ! have_cap_sys_admin ; then
      unshareflags="$unshareflags -U"
      unshareflags="$unshareflags $(awk 'BEGIN { start=0 }{print "--map-users="start","start","$3; start=start+$3}' /proc/self/uid_map)"
      unshareflags="$unshareflags $(awk 'BEGIN { start=0 }{print "--map-groups="start","start","$3; start=start+$3}' /proc/self/gid_map)"
    fi
    echo '#' unshare $unshareflags env -i bash -x ${TEST_SCRATCH_DIR}/script.sh
    unshare $unshareflags env -i bash -x ${TEST_SCRATCH_DIR}/script.sh
  fi
}

@test "chroot with overlay root" {
  if test `uname` != Linux ; then
    skip "not meaningful except on Linux"
  fi
  skip_if_no_unshare
  if [ "$(id -u)" -ne 0 ]; then
    skip "expects to already be root"
  fi
  skip_if_root_is_on_overlay
  _prefetch docker.io/library/busybox
  cp -v ${TEST_SOURCES}/containers.conf ${TEST_SCRATCH_DIR}/containers.conf
  chmod ugo+r ${TEST_SCRATCH_DIR}/containers.conf
  mkdir -p ${TEST_SCRATCH_DIR}/chroot
  chown -R 1:1 ${TEST_SCRATCH_DIR}/root ${TEST_SCRATCH_DIR}/runroot ${TEST_SCRATCH_DIR}/chroot
  cat > ${TEST_SCRATCH_DIR}/script1 <<- EOF
  # mount an overlay filesystem with the real root as the "lower", pivot into it,
  # and then try a build
  PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin${PATH:+:$PATH}
  set -e
  set -x
  mkdir -p ${TEST_SCRATCH_DIR}/chroot/workdir
  mkdir -p ${TEST_SCRATCH_DIR}/chroot/upperdir
  mkdir -p ${TEST_SCRATCH_DIR}/chroot/merged
  mount -t overlay overlay -o upperdir=${TEST_SCRATCH_DIR}/chroot/upperdir,workdir=${TEST_SCRATCH_DIR}/chroot/workdir,lowerdir=/ ${TEST_SCRATCH_DIR}/chroot/merged
  mkdir -p ${TEST_SCRATCH_DIR}/chroot/merged/buildcontext
  echo FROM docker.io/library/busybox > ${TEST_SCRATCH_DIR}/chroot/merged/buildcontext/Dockerfile
  echo RUN cat /proc/mounts >> ${TEST_SCRATCH_DIR}/chroot/merged/buildcontext/Dockerfile
  chown -R 1:1 ${TEST_SCRATCH_DIR}/chroot/merged/buildcontext
  mount -t proc proc ${TEST_SCRATCH_DIR}/chroot/merged/proc
  mount -t sysfs sysfs ${TEST_SCRATCH_DIR}/chroot/merged/sys
  mount --rbind /dev ${TEST_SCRATCH_DIR}/chroot/merged/dev
  mount --bind /etc ${TEST_SCRATCH_DIR}/chroot/merged/etc
  echo build > ${TEST_SCRATCH_DIR}/chroot/hostname
  chmod 644 ${TEST_SCRATCH_DIR}/chroot/hostname
  mount --bind ${TEST_SCRATCH_DIR}/chroot/hostname ${TEST_SCRATCH_DIR}/chroot/merged/etc/hostname
  touch ${TEST_SCRATCH_DIR}/chroot/hosts
  chmod 644 ${TEST_SCRATCH_DIR}/chroot/hosts
  mount --bind ${TEST_SCRATCH_DIR}/chroot/hosts ${TEST_SCRATCH_DIR}/chroot/merged/etc/hosts
  touch ${TEST_SCRATCH_DIR}/chroot/resolv.conf
  chmod 644 ${TEST_SCRATCH_DIR}/chroot/resolv.conf
  mount --bind ${TEST_SCRATCH_DIR}/chroot/resolv.conf ${TEST_SCRATCH_DIR}/chroot/merged/etc/resolv.conf
  mount --bind /tmp ${TEST_SCRATCH_DIR}/chroot/merged/tmp
  mkdir -p ${TEST_SCRATCH_DIR}/chroot/merged/var/tmp
  chmod 1777 ${TEST_SCRATCH_DIR}/chroot/merged/var/tmp
  if test -d /var/tmp; then
    mount --bind /var/tmp ${TEST_SCRATCH_DIR}/chroot/merged/var/tmp
  fi
  mount --bind ${TEST_SCRATCH_DIR} ${TEST_SCRATCH_DIR}/chroot/merged/${TEST_SCRATCH_DIR}
  mkdir -p ${TEST_SCRATCH_DIR}/chroot/merged/usr/local/bin
  touch ${TEST_SCRATCH_DIR}/chroot/merged/usr/local/bin/buildah
  mount --bind ${BUILDAH_BINARY:-$TEST_SOURCES/../bin/buildah} ${TEST_SCRATCH_DIR}/chroot/merged/usr/local/bin/buildah
  cd ${TEST_SCRATCH_DIR}/chroot/merged
  pivot_root . tmp
  mount --make-rslave tmp
  umount -f -l tmp
  mount -o remount,ro --make-rshared /
  grep ' / / ' /proc/self/mountinfo
  # unshare from util-linux 2.39 also accepts INNER:OUTER:SIZE for --map-users
  # and --map-groups, but fedora 37's is too old, so the older OUTER,INNER,SIZE
  # (using commas instead of colons as field separators) will have to do
  unshare --setuid 0 --setgid 0 --map-users=1,0,1024 --map-groups=1,0,1024 -Um bash -x ${TEST_SCRATCH_DIR}/script2
EOF
  cat > ${TEST_SCRATCH_DIR}/script2 <<- EOF
  set -e
  set -x
  export _CONTAINERS_USERNS_CONFIGURED=done
  export CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf
  cat /proc/self/uid_map
  cat /proc/self/gid_map
  mount --make-shared /
  /usr/local/bin/buildah ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} from --name ctrid --pull=never docker.io/library/busybox
  /usr/local/bin/buildah ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} run --isolation=chroot ctrid pwd
  /usr/local/bin/buildah ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} build --isolation=chroot /buildcontext
EOF
  chmod +x ${TEST_SCRATCH_DIR}
  chmod +rx ${TEST_SCRATCH_DIR}/script1 ${TEST_SCRATCH_DIR}/script2
  local unshareflags="-m --setuid 0 --setgid 0"
  if ! have_cap_sys_admin ; then
    unshareflags="$unshareflags -U"
    unshareflags="$unshareflags $(awk 'BEGIN { start=0 }{print "--map-users="start","start","$3; start=start+$3}' /proc/self/uid_map)"
    unshareflags="$unshareflags $(awk 'BEGIN { start=0 }{print "--map-groups="start","start","$3; start=start+$3}' /proc/self/gid_map)"
  fi
  env -i unshare $unshareflags bash -x ${TEST_SCRATCH_DIR}/script1
}
