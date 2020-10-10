#!/usr/bin/env bats

load helpers

@test "already-in-userns" {
  if test "$BUILDAH_ISOLATION" != "rootless" -o $UID == 0 ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi

  _prefetch alpine
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet alpine
  expect_output "alpine-working-container"
  ctr="$output"

  run_buildah unshare buildah run --isolation=oci "$ctr" echo hello
  expect_output "hello"
}

@test "user-and-network-namespace" {
  skip_if_chroot
  skip_if_rootless

  mkdir -p $TESTDIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TESTDIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"
  # Check if we're running in an environment that can even test this.
  run readlink /proc/self/ns/user
  echo "readlink /proc/self/ns/user -> $output"
  [ $status -eq 0 ] || skip "user namespaces not supported"
  run readlink /proc/self/ns/net
  echo "readlink /proc/self/ns/net -> $output"
  [ $status -eq 0 ] || skip "network namespaces not supported"
  mynetns="$output"

  # Generate the mappings to use for using-a-user-namespace cases.
  uidbase=$((${RANDOM}+1024))
  gidbase=$((${RANDOM}+1024))
  uidsize=$((${RANDOM}+1024))
  gidsize=$((${RANDOM}+1024))

  # Create a container that uses that mapping.
  _prefetch alpine
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet --userns-uid-map 0:$uidbase:$uidsize --userns-gid-map 0:$gidbase:$gidsize alpine
  ctr="$output"

  # Check that with settings that require a user namespace, we also get a new network namespace by default.
  run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  if [[ $output == $mynetns ]]; then
      expect_output "[output should not be '$mynetns']"
  fi

  # Check that with settings that require a user namespace, we can still try to use the host's network namespace.
  run_buildah run $RUNOPTS --net=host "$ctr" readlink /proc/self/ns/net
  expect_output "$mynetns"

  # Create a container that doesn't use that mapping.
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet alpine
  ctr="$output"

  # Check that with settings that don't require a user namespace, we don't get a new network namespace by default.
  run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  expect_output "$mynetns"

  # Check that with settings that don't require a user namespace, we can request to use a per-container network namespace.
  run_buildah run $RUNOPTS --net=container "$ctr" readlink /proc/self/ns/net
  if [[ $output == $mynetns ]]; then
      expect_output "[output should not be '$mynetns']"
  fi

  run_buildah run $RUNOPTS --net=private "$ctr" readlink /proc/self/ns/net
  if [[ $output == $mynetns ]]; then
      expect_output "[output should not be '$mynetns']"
  fi
}

@test "idmapping" {
  mkdir -p $TESTDIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TESTDIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"

  # Check if we're running in an environment that can even test this.
  run readlink /proc/self/ns/user
  echo "readlink /proc/self/ns/user -> $output"
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

  for i in $(seq 0 "$((${#uidmaps[*]}-1))") ; do
    # Create a container using these mappings.
    echo "Building container with --signature-policy ${TESTSDIR}/policy.json --quiet ${uidmapargs[$i]} ${gidmapargs[$i]} alpine"
    _prefetch alpine
    run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet ${uidmapargs[$i]} ${gidmapargs[$i]} alpine
    ctr="$output"

    # If we specified mappings, expect to be in a different namespace by default.
    run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/user
    [ "$output" != "" ]
    case x"${uidmapargs[$i]}""${gidmapargs[$i]}" in
    x)
      if test "$BUILDAH_ISOLATION" != "chroot" -a "$BUILDAH_ISOLATION" != "rootless" ; then
        expect_output "$mynamespace"
      fi
      ;;
    *)
      [ "$output" != "$mynamespace" ]
      ;;
    esac
    # Check that we got the mappings that we expected.
    run_buildah run $RUNOPTS "$ctr" cat /proc/self/uid_map
    [ "$output" != "" ]
    uidmap=$(sed -E -e 's, +, ,g' -e 's,^ +,,g' <<< "$output")
    run_buildah run $RUNOPTS "$ctr" cat /proc/self/gid_map
    [ "$output" != "" ]
    gidmap=$(sed -E -e 's, +, ,g' -e 's,^ +,,g' <<< "$output")
    echo With settings "$map", expected UID map "${uidmaps[$i]}", got UID map "${uidmap}", expected GID map "${gidmaps[$i]}", got GID map "${gidmap}".
    expect_output --from="$uidmap" "${uidmaps[$i]}"
    expect_output --from="$gidmap" "${gidmaps[$i]}"
    rootuid=$(sed -E -e 's,^([^ ]*) (.*) ([^ ]*),\2,' <<< "$uidmap")
    rootgid=$(sed -E -e 's,^([^ ]*) (.*) ([^ ]*),\2,' <<< "$gidmap")

    # Check that if we copy a file into the container, it gets the right permissions.
    run_buildah copy --chown 1:1 "$ctr" ${TESTDIR}/somefile /
    run_buildah run $RUNOPTS "$ctr" stat -c '%u:%g' /somefile
    expect_output "1:1"

    # Check that if we copy a directory into the container, its contents get the right permissions.
    run_buildah copy "$ctr" ${TESTDIR}/somedir /somedir
    run_buildah run $RUNOPTS "$ctr" stat -c '%u:%g' /somedir
    expect_output "0:0"
    run_buildah mount "$ctr"
    mnt="$output"
    run stat -c '%u:%g %a' "$mnt"/somedir/someotherfile
    [ $status -eq 0 ]
    expect_output "$rootuid:$rootgid 4700"
    run_buildah run $RUNOPTS "$ctr" stat -c '%u:%g %a' /somedir/someotherfile
    expect_output "0:0 4700"
  done
}

@test "idmapping-syntax" {
  run_buildah 125 from --signature-policy ${TESTSDIR}/policy.json --quiet --userns-uid-map=0:10000:65536 alpine
  expect_output --substring "must be used together"
  run_buildah 125 from --signature-policy ${TESTSDIR}/policy.json --quiet --userns-gid-map=0:10000:65536 alpine
  expect_output --substring "must be used together"
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
  echo "readlink /proc/self/ns/$nstype -> $output"
  [ $status -eq 0 ] || skip "$nstype namespaces not supported"
  mynamespace="$output"

  # Settings to test.
  types[0]=
  types[1]=container
  types[2]=host
  types[3]=/proc/$$/ns/$nstype
  types[4]=private
  types[5]=ns:/proc/$$/ns/$nstype

  _prefetch alpine
  for namespace in "${types[@]}" ; do
    # Specify the setting for this namespace for this container.
    run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet --"$nsflag"=$namespace alpine
    [ "$output" != "" ]
    ctr="$output"

    # Check that, unless we override it, we get that setting in "run".
    run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/"$nstype"
    [ "$output" != "" ]
    case "$namespace" in
    ""|container|private)
      [ "$output" != "$mynamespace" ]
      ;;
    host)
      expect_output "$mynamespace"
      ;;
    /*)
      expect_output "$(readlink $namespace)"
      ;;
    esac

    for different in $types ; do
      # Check that, if we override it, we get what we specify for "run".
      run_buildah run $RUNOPTS --"$nsflag"=$different "$ctr" readlink /proc/self/ns/"$nstype"
      [ "$output" != "" ]
      case "$different" in
      ""|container|private)
        [ "$output" != "$mynamespace" ]
        ;;
      host)
        expect_output "$mynamespace"
        ;;
      /*)
        expect_output "$(readlink $namespace)"
        ;;
      esac
    done

  done
}

@test "ipc-namespace" {
  skip_if_chroot
  skip_if_rootless

  general_namespace ipc
}

@test "net-namespace" {
  skip_if_chroot
  skip_if_rootless

  general_namespace net
}

@test "network-namespace" {
  skip_if_chroot
  skip_if_rootless

  general_namespace net network
}

@test "pid-namespace" {
  skip_if_chroot
  skip_if_rootless

  general_namespace pid
}

@test "user-namespace" {
  skip_if_chroot
  skip_if_rootless

  general_namespace user userns
}

@test "uts-namespace" {
  skip_if_chroot
  skip_if_rootless

  general_namespace uts
}

@test "combination-namespaces" {
  skip_if_chroot
  skip_if_rootless

  _prefetch alpine
  # mnt is always per-container, cgroup isn't a thing OCI runtime lets us configure
  for ipc in host container private ; do
    for net in host container private; do
      for pid in host container private; do
        for userns in host container private; do
          for uts in host container private; do

            if test $userns == private -o $userns == container -a $pid == host ; then
              # We can't mount a fresh /proc, and OCI runtime won't let us bind mount the host's.
              continue
            fi

            echo "buildah from --signature-policy ${TESTSDIR}/policy.json --ipc=$ipc --net=$net --pid=$pid --userns=$userns --uts=$uts alpine"
            run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet --ipc=$ipc --net=$net --pid=$pid --userns=$userns --uts=$uts alpine
            [ "$output" != "" ]
            ctr="$output"
            run_buildah run $ctr pwd
            [ "$output" != "" ]
            run_buildah run --tty=true  $ctr pwd
            [ "$output" != "" ]
            run_buildah run --tty=false $ctr pwd
            [ "$output" != "" ]
          done
        done
      done
    done
  done
}

@test "idmapping-and-squash" {
	createrandom ${TESTDIR}/randomfile
	run_buildah from --userns-uid-map 0:32:16 --userns-gid-map 0:48:16 scratch
	cid=$output
	run_buildah copy "$cid" ${TESTDIR}/randomfile /
	run_buildah copy --chown 1:1 "$cid" ${TESTDIR}/randomfile /randomfile2
	run_buildah commit --squash --signature-policy ${TESTSDIR}/policy.json --rm "$cid" squashed
	run_buildah from --quiet squashed
	cid=$output
	run_buildah mount $cid
	mountpoint=$output
	run stat -c %u:%g $mountpoint/randomfile
	[ "$status" -eq 0 ]
        expect_output "0:0"

	run stat -c %u:%g $mountpoint/randomfile2
	[ "$status" -eq 0 ]
        expect_output "1:1"
}
