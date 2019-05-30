#!/usr/bin/env bats

load helpers

@test "user-and-network-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  mkdir -p $TESTDIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TESTDIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"
  # Check if we're running in an environment that can even test this.
  run readlink /proc/self/ns/user
  echo "$output"
  [ $status -eq 0 ] || skip "user namespaces not supported"
  run readlink /proc/self/ns/net
  echo "$output"
  [ $status -eq 0 ] || skip "network namespaces not supported"
  mynetns="$output"

  # Generate the mappings to use for using-a-user-namespace cases.
  uidbase=$((${RANDOM}+1024))
  gidbase=$((${RANDOM}+1024))
  uidsize=$((${RANDOM}+1024))
  gidsize=$((${RANDOM}+1024))

  # Create a container that uses that mapping.
  run_buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json --quiet --userns-uid-map 0:$uidbase:$uidsize --userns-gid-map 0:$gidbase:$gidsize alpine
  [ "$output" != "" ]
  ctr="$output"

  # Check that with settings that require a user namespace, we also get a new network namespace by default.
  buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  run_buildah --debug=false run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  [ "$output" != "" ]
  [ "$output" != "$mynetns" ]

  # Check that with settings that require a user namespace, we can still try to use the host's network namespace.
  buildah run $RUNOPTS --net=host "$ctr" readlink /proc/self/ns/net
  run_buildah --debug=false run $RUNOPTS --net=host "$ctr" readlink /proc/self/ns/net
  [ "$output" != "" ]
  [ "$output" == "$mynetns" ]

  # Create a container that doesn't use that mapping.
  run_buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json --quiet alpine
  [ "$output" != "" ]
  ctr="$output"

  # Check that with settings that don't require a user namespace, we don't get a new network namespace by default.
  buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  run_buildah --debug=false run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  [ "$output" != "" ]
  [ "$output" == "$mynetns" ]

  # Check that with settings that don't require a user namespace, we can request to use a per-container network namespace.
  buildah run $RUNOPTS --net=container "$ctr" readlink /proc/self/ns/net
  run_buildah --debug=false run $RUNOPTS --net=container "$ctr" readlink /proc/self/ns/net
  [ "$output" != "" ]
  [ "$output" != "$mynetns" ]
}

@test "idmapping" {
  mkdir -p $TESTDIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TESTDIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"

  # Check if we're running in an environment that can even test this.
  run readlink /proc/self/ns/user
  echo "$output"
  [ $status -eq 0 ] || skip "user namespaces not supported"
  mynamespace="$output"

  # Generate the mappings to use.
  uidbase=$((${RANDOM}+1024))
  gidbase=$((${RANDOM}+1024))
  uidsize=$((${RANDOM}+1024))
  gidsize=$((${RANDOM}+1024))
  # Test with no mappings.
  uidmapargs[0]=
  gidmapargs[0]=
  uidmaps[0]="0 0 4294967295"
  gidmaps[0]="0 0 4294967295"
  # Test with both UID and GID maps specified.
  uidmapargs[1]="--userns-uid-map=0:$uidbase:$uidsize"
  gidmapargs[1]="--userns-gid-map=0:$gidbase:$gidsize"
  uidmaps[1]="0 $uidbase $uidsize"
  gidmaps[1]="0 $gidbase $gidsize"
  # Test with just a UID map specified.
  uidmapargs[2]=--userns-uid-map=0:$uidbase:$uidsize
  uidmaps[2]="0 $uidbase $uidsize"
  gidmaps[2]="0 $uidbase $uidsize"
  # Test with just a GID map specified.
  gidmapargs[3]=--userns-gid-map=0:$gidbase:$gidsize
  uidmaps[3]="0 $gidbase $gidsize"
  gidmaps[3]="0 $gidbase $gidsize"
  # Conditionalize some tests on the subuid and subgid files being present.
  if test -s /etc/subuid ; then
    if test -s /etc/subgid ; then
      # Look for a name that's in both the subuid and subgid files.
      for candidate in $(sed -e 's,:.*,,g' /etc/subuid); do
        if test $(sed -e 's,:.*,,g' -e "/$candidate/!d" /etc/subgid) == "$candidate"; then
          # Read the start of the subuid/subgid ranges.  Assume length=65536.
          userbase=$(sed -e "/^${candidate}:/!d" -e 's,^[^:]*:,,g' -e 's,:[^:]*,,g' /etc/subuid)
          groupbase=$(sed -e "/^${candidate}:/!d" -e 's,^[^:]*:,,g' -e 's,:[^:]*,,g' /etc/subgid)
          # Test specifying both the user and group names.
          uidmapargs[${#uidmaps[*]}]=--userns-uid-map-user=$candidate
          gidmapargs[${#gidmaps[*]}]=--userns-gid-map-group=$candidate
          uidmaps[${#uidmaps[*]}]="0 $userbase 65536"
          gidmaps[${#gidmaps[*]}]="0 $groupbase 65536"
          # Test specifying just the user name.
          uidmapargs[${#uidmaps[*]}]=--userns-uid-map-user=$candidate
          uidmaps[${#uidmaps[*]}]="0 $userbase 65536"
          gidmaps[${#gidmaps[*]}]="0 $groupbase 65536"
          # Test specifying just the group name.
          gidmapargs[${#gidmaps[*]}]=--userns-gid-map-group=$candidate
          uidmaps[${#uidmaps[*]}]="0 $userbase 65536"
          gidmaps[${#gidmaps[*]}]="0 $groupbase 65536"
          break
        fi
      done
      # Choose different names from the files.
      for candidateuser in $(sed -e 's,:.*,,g' /etc/subuid); do
        for candidategroup in $(sed -e 's,:.*,,g' /etc/subgid); do
          if test "$candidateuser" == "$candidate" ; then
            continue
          fi
          if test "$candidategroup" == "$candidate" ; then
            continue
          fi
          if test "$candidateuser" == "$candidategroup" ; then
            continue
          fi
          # Read the start of the ranges.  Assume length=65536.
          userbase=$(sed -e "/^${candidateuser}:/!d" -e 's,^[^:]*:,,g' -e 's,:[^:]*,,g' /etc/subuid)
          groupbase=$(sed -e "/^${candidategroup}:/!d" -e 's,^[^:]*:,,g' -e 's,:[^:]*,,g' /etc/subgid)
          # Test specifying both the user and group names.
          uidmapargs[${#uidmaps[*]}]=--userns-uid-map-user=$candidateuser
          gidmapargs[${#gidmaps[*]}]=--userns-gid-map-group=$candidategroup
          uidmaps[${#uidmaps[*]}]="0 $userbase 65536"
          gidmaps[${#gidmaps[*]}]="0 $groupbase 65536"
          break
        done
      done
    fi
  fi

  touch ${TESTDIR}/somefile
  mkdir ${TESTDIR}/somedir
  touch ${TESTDIR}/somedir/someotherfile
  chmod 700 ${TESTDIR}/somedir/someotherfile
  chmod u+s ${TESTDIR}/somedir/someotherfile

  for i in $(seq 0 "$((${#maps[*]}-1))") ; do
    # Create a container using these mappings.
    echo "Building container with --signature-policy ${TESTSDIR}/policy.json --quiet ${uidmapargs[$i]} ${gidmapargs[$i]} alpine"
    run_buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json --quiet ${uidmapargs[$i]} ${gidmapargs[$i]} alpine
    [ "$output" != "" ]
    ctr="$output"

    # If we specified mappings, expect to be in a different namespace by default.
    buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/user
    run_buildah --debug=false run $RUNOPTS "$ctr" readlink /proc/self/ns/user
    [ "$output" != "" ]
    case x"$map" in
    x)
      if test "$BUILDAH_ISOLATION" != "chroot" -a "$BUILDAH_ISOLATION" != "rootless" ; then
        [ "$output" == "$mynamespace" ]
      fi
      ;;
    *)
      [ "$output" != "$mynamespace" ]
      ;;
    esac
    # Check that we got the mappings that we expected.
    buildah run $RUNOPTS "$ctr" cat /proc/self/uid_map
    run_buildah --debug=false run $RUNOPTS "$ctr" cat /proc/self/uid_map
    [ "$output" != "" ]
    uidmap=$(sed -E -e 's, +, ,g' -e 's,^ +,,g' <<< "$output")
    buildah run $RUNOPTS "$ctr" cat /proc/self/gid_map
    run_buildah --debug=false run $RUNOPTS "$ctr" cat /proc/self/gid_map
    [ "$output" != "" ]
    gidmap=$(sed -E -e 's, +, ,g' -e 's,^ +,,g' <<< "$output")
    echo With settings "$map", expected UID map "${uidmaps[$i]}", got UID map "${uidmap}", expected GID map "${gidmaps[$i]}", got GID map "${gidmap}".
    [ "$uidmap" == "${uidmaps[$i]}" ]
    [ "$gidmap" == "${gidmaps[$i]}" ]
    rootuid=$(sed -E -e 's,^([^ ]*) (.*) ([^ ]*),\2,' <<< "$uidmap")
    rootgid=$(sed -E -e 's,^([^ ]*) (.*) ([^ ]*),\2,' <<< "$gidmap")

    # Check that if we copy a file into the container, it gets the right permissions.
    run_buildah copy --chown 1:1 "$ctr" ${TESTDIR}/somefile /
    buildah run $RUNOPTS "$ctr" stat -c '%u:%g' /somefile
    run_buildah --debug=false run $RUNOPTS "$ctr" stat -c '%u:%g' /somefile
    expect_output "1:1"

    # Check that if we copy a directory into the container, its contents get the right permissions.
    run_buildah copy "$ctr" ${TESTDIR}/somedir /somedir
    buildah run $RUNOPTS "$ctr" stat -c '%u:%g' /somedir
    run_buildah --debug=false run $RUNOPTS "$ctr" stat -c '%u:%g' /somedir
    expect_output "0:0"
    run_buildah --debug=false mount "$ctr"
    mnt="$output"
    run stat -c '%u:%g %a' "$mnt"/somedir/someotherfile
    [ $status -eq 0 ]
    expect_output "$rootuid:$rootgid 4700"
    buildah run $RUNOPTS "$ctr" stat -c '%u:%g %a' /somedir/someotherfile
    run_buildah --debug=false run $RUNOPTS "$ctr" stat -c '%u:%g %a' /somedir/someotherfile
    expect_output "0:0 4700"
  done
}

general_namespace() {
  mkdir -p $TESTDIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TESTDIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"

  # The name of the /proc/self/ns/$link.
  nstype="$1"
  # The flag to use, if it's not the same as the namespace name.
  nsflag="${2:-$1}"

  # Check if we're running in an environment that can even test this.
  run readlink /proc/self/ns/"$nstype"
  echo "$output"
  [ $status -eq 0 ] || skip "$nstype namespaces not supported"
  mynamespace="$output"

  # Settings to test.
  types[0]=
  types[1]=container
  types[2]=host
  types[3]=/proc/$$/ns/$nstype

  for namespace in "${types[@]}" ; do
    # Specify the setting for this namespace for this container.
    run_buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json --quiet --"$nsflag"=$namespace alpine
    [ "$output" != "" ]
    ctr="$output"

    # Check that, unless we override it, we get that setting in "run".
    run_buildah --debug=false run $RUNOPTS "$ctr" readlink /proc/self/ns/"$nstype"
    [ "$output" != "" ]
    case "$namespace" in
    ""|container)
      [ "$output" != "$mynamespace" ]
      ;;
    host)
      [ "$output" == "$mynamespace" ]
      ;;
    /*)
      [ "$output" == $(readlink "$namespace") ]
      ;;
    esac

    for different in $types ; do
      # Check that, if we override it, we get what we specify for "run".
      run_buildah --debug=false run $RUNOPTS --"$nsflag"=$different "$ctr" readlink /proc/self/ns/"$nstype"
      [ "$output" != "" ]
      case "$different" in
      ""|container)
        [ "$output" != "$mynamespace" ]
        ;;
      host)
        [ "$output" == "$mynamespace" ]
        ;;
      /*)
        [ "$output" == $(readlink "$namespace") ]
        ;;
      esac
    done

  done
}

@test "ipc-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  general_namespace ipc
}

@test "net-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  general_namespace net
}

@test "network-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  general_namespace net network
}

@test "pid-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  general_namespace pid
}

@test "user-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  general_namespace user userns
}

@test "uts-namespace" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  general_namespace uts
}

@test "combination-namespaces" {
  if test "$BUILDAH_ISOLATION" = "chroot" -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  # mnt is always per-container, cgroup isn't a thing runc lets us configure
  for ipc in host container ; do
    for net in host container ; do
      for pid in host container ; do
        for userns in host container ; do
          for uts in host container ; do

            if test $userns == container -a $pid == host ; then
              # We can't mount a fresh /proc, and runc won't let us bind mount the host's.
              continue
            fi

            echo "buildah from --signature-policy ${TESTSDIR}/policy.json --ipc=$ipc --net=$net --pid=$pid --userns=$userns --uts=$uts alpine"
            run_buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json --quiet --ipc=$ipc --net=$net --pid=$pid --userns=$userns --uts=$uts alpine
            [ "$output" != "" ]
            ctr="$output"
            buildah run $ctr pwd
            run_buildah --debug=false run $ctr pwd
            [ "$output" != "" ]
            buildah run --tty=true  $ctr pwd
            run_buildah --debug=false run --tty=true  $ctr pwd
            [ "$output" != "" ]
            buildah run --tty=false $ctr pwd
            run_buildah --debug=false run --tty=false $ctr pwd
            [ "$output" != "" ]
          done
        done
      done
    done
  done
}

@test "idmapping-and-squash" {
	createrandom ${TESTDIR}/randomfile
	cid=$(buildah from --userns-uid-map 0:32:16 --userns-gid-map 0:48:16 scratch)
	buildah copy "$cid" ${TESTDIR}/randomfile /
	buildah copy --chown 1:1 "$cid" ${TESTDIR}/randomfile /randomfile2
	buildah commit --squash --signature-policy ${TESTSDIR}/policy.json --rm "$cid" squashed
	cid=$(buildah from squashed)
	mountpoint=$(buildah mount $cid)
	run stat -c %u:%g $mountpoint/randomfile
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" = 0:0 ]
	run stat -c %u:%g $mountpoint/randomfile2
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" = 1:1 ]
}
