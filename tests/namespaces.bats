#!/usr/bin/env bats

load helpers

@test "already-in-userns" {
  if test "$BUILDAH_ISOLATION" != "rootless" -o $UID == 0 ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi

  _prefetch alpine
  run_buildah from $WITH_POLICY_JSON --quiet alpine
  expect_output "alpine-working-container"
  ctr="$output"

  run_buildah unshare buildah run --isolation=oci "$ctr" echo hello
  expect_output "hello"
}

@test "user-and-network-namespace" {
  skip_if_rootless_environment
  skip_if_chroot
  skip_if_rootless

  mkdir -p $TEST_SCRATCH_DIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TEST_SCRATCH_DIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"
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
  run_buildah from $WITH_POLICY_JSON --quiet --userns-uid-map 0:$uidbase:$uidsize --userns-gid-map 0:$gidbase:$gidsize alpine
  ctr="$output"

  # Check that with settings that require a user namespace, we also get a new network namespace by default.
  run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  assert "$output" != "$mynetns" "we should get a new network namespace"

  # Check that with settings that require a user namespace, we can still try to use the host's network namespace.
  run_buildah run $RUNOPTS --net=host "$ctr" readlink /proc/self/ns/net
  expect_output "$mynetns"

  # Check that we are not bind mounting /sys from the host with --net=container
  host_sys=$(grep "/sys " /proc/self/mountinfo | cut -d ' ' -f 3)
  run_buildah run $RUNOPTS --net=container "$ctr" sh -c 'grep "/sys " /proc/self/mountinfo | cut -d " " -f 3'
  assert "$output" != "$host_sys"

  # Create a container that doesn't use that mapping.
  run_buildah from $WITH_POLICY_JSON --quiet alpine
  ctr="$output"

  run_buildah run $RUNOPTS --net=host "$ctr" readlink /proc/self/ns/net
  expect_output "$mynetns"

  # Check that with settings that don't require a user namespace, we can request to use a per-container network namespace.
  run_buildah run $RUNOPTS --net=container "$ctr" readlink /proc/self/ns/net
  assert "$output" != "$mynetns" \
         "[/proc/self/ns/net (--net=container) should not be '$mynetns']"

  run_buildah run $RUNOPTS --net=private "$ctr" readlink /proc/self/ns/net
  assert "$output" != "$mynetns" \
         "[/proc/self/ns/net (--net=private) should not be '$mynetns']"

  run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/net
  assert "$output" != "$mynetns" \
         "[/proc/self/ns/net (--net="") should not be '$mynetns']"
}

# Helper for idmapping test: check UID or GID mapping
# NOTE SIDE EFFECT: sets $rootxid for possible use by caller
idmapping_check_map() {
  local _output_idmap=$1
  local _expect_idmap=$2
  local _testname=$3

  assert "$_output_idmap" != "" "Internal error: output_idmap is empty"
  local _idmap=$(sed -E -e 's, +, ,g' -e 's,^ +,,g' <<< "${_output_idmap}")
  expect_output --from="$_idmap" "${_expect_idmap}" "$_testname"

  # SIDE EFFECT: Global: our caller may want this
  rootxid=$(sed -E -e 's,^([^ ]*) (.*) ([^ ]*),\2,' <<< "$_idmap")
}

# Helper for idmapping test: check file permissions
idmapping_check_permission() {
  local _output_file_stat=$1
  local _output_dir_stat=$2

  expect_output --from="${_output_file_stat}" "1:1" "Check if a copied file gets the right permissions"
  expect_output --from="${_output_dir_stat}" "0:0" "Check if a copied directory gets the right permissions"
}

@test "idmapping" {
  skip_if_rootless_environment
  mkdir -p $TEST_SCRATCH_DIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TEST_SCRATCH_DIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"

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

  touch ${TEST_SCRATCH_DIR}/somefile
  mkdir ${TEST_SCRATCH_DIR}/somedir
  touch ${TEST_SCRATCH_DIR}/somedir/someotherfile
  chmod 700 ${TEST_SCRATCH_DIR}/somedir/someotherfile
  chmod u+s ${TEST_SCRATCH_DIR}/somedir/someotherfile

  for i in $(seq 0 "$((${#uidmaps[*]}-1))") ; do
    # local helper function for checking /proc/self/ns/user
    function idmapping_check_namespace() {
      local _output=$1
      local _testname=$2

      assert "$_output" != "" "Internal error: _output is empty"
      if [ -z "${uidmapargs[$i]}${gidmapargs[$i]}" ]; then
        if test "$BUILDAH_ISOLATION" != "chroot" -a "$BUILDAH_ISOLATION" != "rootless" ; then
          expect_output --from="$_output" "$mynamespace" "/proc/self/ns/user ($_testname)"
        fi
      else
        assert "$_output" != "$mynamespace" "_output vs mynamespace"
      fi
    }

    # Create a container using these mappings.
    echo "Building container with $WITH_POLICY_JSON --quiet ${uidmapargs[$i]} ${gidmapargs[$i]} alpine"
    _prefetch alpine
    run_buildah from $WITH_POLICY_JSON --quiet ${uidmapargs[$i]} ${gidmapargs[$i]} alpine
    ctr="$output"

    # If we specified mappings, expect to be in a different namespace by default.
    run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/user
    idmapping_check_namespace "$output" "container"
    # Check that we got the UID and GID mappings that we expected.
    # rootuid/rootgid are obtained (side effect) from helper function
    run_buildah run $RUNOPTS "$ctr" cat /proc/self/uid_map
    idmapping_check_map "$output" "${uidmaps[$i]}" "uid_map"
    rootuid=$rootxid

    run_buildah run $RUNOPTS "$ctr" cat /proc/self/gid_map
    idmapping_check_map "$output" "${gidmaps[$i]}" "gid_map"
    rootgid=$rootxid

    # Check that if we copy a file into the container, it gets the right permissions.
    run_buildah copy --chown 1:1 "$ctr" ${TEST_SCRATCH_DIR}/somefile /
    run_buildah run $RUNOPTS "$ctr" stat -c '%u:%g' /somefile
    output_file_stat="$output"
    # Check that if we copy a directory into the container, its contents get the right permissions.
    run_buildah copy "$ctr" ${TEST_SCRATCH_DIR}/somedir /somedir
    run_buildah run $RUNOPTS "$ctr" stat -c '%u:%g' /somedir
    output_dir_stat="$output"
    idmapping_check_permission "$output_file_stat" "$output_dir_stat"

    run_buildah run $RUNOPTS "$ctr" stat -c '%u:%g %a' /somedir/someotherfile
    expect_output "0:0 4700" "stat(someotherfile), in container test"

    # Check that the copied file has the right permissions on host.
    run_buildah mount "$ctr"
    mnt="$output"
    run stat -c '%u:%g %a' "$mnt"/somedir/someotherfile
    assert "$status" -eq 0 "status of stat $mnt/somedir/someotherfile"
    expect_output "$rootuid:$rootgid 4700"

    # Check that a container with mapped-layer can be committed.
    run_buildah commit "$ctr" localhost/alpine-working:$i


    # Also test bud command
    # Build an image using these mappings.
    echo "Building image with ${uidmapargs[$i]} ${gidmapargs[$i]}"
    run_buildah bud ${uidmapargs[$i]} ${gidmapargs[$i]} $RUNOPTS $WITH_POLICY_JSON \
                    -t localhost/alpine-bud:$i -f $BUDFILES/namespaces/Containerfile $TEST_SCRATCH_DIR
    # If we specified mappings, expect to be in a different namespace by default.
    output_namespace="$(grep -A1 'ReadlinkResult' <<< "$output" | tail -n1)"
    idmapping_check_namespace "${output_namespace}" "bud"
    # Check that we got the mappings that we expected.
    output_uidmap="$(grep -A1 'UidMapResult' <<< "$output" | tail -n1)"
    output_gidmap="$(grep -A1 'GidMapResult' <<< "$output" | tail -n1)"
    idmapping_check_map "$output_uidmap" "${uidmaps[$i]}" "UidMapResult"
    idmapping_check_map "$output_gidmap" "${gidmaps[$i]}" "GidMapResult"

    # Check that if we copy a file into the container, it gets the right permissions.
    output_file_stat="$(grep -A1 'StatSomefileResult' <<< "$output" | tail -n1)"
    # Check that if we copy a directory into the container, its contents get the right permissions.
    output_dir_stat="$(grep -A1 'StatSomedirResult' <<< "$output" | tail -n1)"
    output_otherfile_stat="$(grep -A1 'StatSomeotherfileResult' <<< "$output" | tail -n1)"
    output_workdir_stat="$(grep -A1 'StatNewWorkdir' <<< "$output" | tail -n1)"
    # bud strips suid.
    idmapping_check_permission "$output_file_stat" "$output_dir_stat"
    expect_output --from="${output_otherfile_stat}" "0:0 700" "stat(someotherfile), in bud test"
    expect_output --from="${output_workdir_stat}" "guest:users" "stat(new-workdir), in bud test"
  done
}

general_namespace() {
  mkdir -p $TEST_SCRATCH_DIR/no-cni-configs
  RUNOPTS="--cni-config-dir=${TEST_SCRATCH_DIR}/no-cni-configs ${RUNC_BINARY:+--runtime $RUNC_BINARY}"
  mytmpdir=$TEST_SCRATCH_DIR/my-dir
  mkdir -p ${mytmpdir}

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
    run_buildah from $WITH_POLICY_JSON --quiet --"$nsflag"=$namespace alpine
    assert "$output" != "" "Internal error: buildah-from produced no output"
    ctr="$output"

    # Check that, unless we override it, we get that setting in "run".
    run_buildah run $RUNOPTS "$ctr" readlink /proc/self/ns/"$nstype"
    assert "$output" != "" "readlink /proc/self/ns/$nstype must not be empty"
    case "$namespace" in
    ""|container|private)
      assert "$output" != "$mynamespace" \
             "readlink /proc/self/ns/$nstype, with namespace=$namespace"
      ;;
    host)
      expect_output "$mynamespace"
      ;;
    /*)
      expect_output "$(readlink $namespace)"
      ;;
    esac

    # "run" doesn't have --userns option.
    if [ "$nsflag" != "userns" ]; then
      for different in ${types[@]} ; do
        # Check that, if we override it, we get what we specify for "run".
        run_buildah run $RUNOPTS --"$nsflag"=$different "$ctr" readlink /proc/self/ns/"$nstype"
        assert "$output" != "" "readlink /proc/self/ns/$nstype must not be empty"
        case "$different" in
        ""|container|private)
          assert "$output" != "$mynamespace" \
                 "readlink /proc/self/ns/$nstype, with different=$different"
          ;;
       host)
          expect_output "$mynamespace"
          ;;
        /*)
          expect_output "$(readlink $different)"
          ;;
        esac
     done
    fi

    # Also check "from" command
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo "TargetOutput" && readlink /proc/self/ns/$nstype
_EOF
    run_buildah bud --"$nsflag"=$namespace $RUNOPTS $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
    result=$(grep -A1 "TargetOutput" <<< "$output" | tail -n1)
    case "$namespace" in
    ""|container|private)
      assert "$result" != "$mynamespace" "readlink /proc/self/ns/$nstype"
      ;;
    host)
      expect_output --from="$result" "$mynamespace"
      ;;
    /*)
      expect_output --from="$result" "$(readlink $namespace)"
      ;;
    esac

  done
}

@test "ipc-namespace" {
  skip_if_chroot
  skip_if_rootless
  skip_if_rootless_environment

  general_namespace ipc
}

@test "net-namespace" {
  skip_if_chroot
  skip_if_rootless
  skip_if_rootless_environment

  general_namespace net
}

@test "network-namespace" {
  skip_if_chroot
  skip_if_rootless
  skip_if_rootless_environment

  general_namespace net network
}

@test "pid-namespace" {
  skip_if_chroot
  skip_if_rootless
  skip_if_rootless_environment

  general_namespace pid
}

@test "user-namespace" {
  skip_if_chroot
  skip_if_rootless
  skip_if_rootless_environment

  general_namespace user userns
}

@test "uts-namespace" {
  skip_if_chroot
  skip_if_rootless
  skip_if_rootless_environment

  general_namespace uts
}

@test "combination-namespaces" {
  skip_if_chroot
  skip_if_rootless

  _prefetch alpine
  # mnt is always per-container, cgroup isn't a thing OCI runtime lets us configure
  for ipc in host private; do
    for net in host private; do
      for pid in host private; do
        for userns in host private; do
          for uts in host private; do
            for cgroupns in host private; do

              echo "buildah from $WITH_POLICY_JSON --ipc=$ipc --net=$net --pid=$pid --userns=$userns --uts=$uts --cgroupns=$cgroupns alpine"
              run_buildah from $WITH_POLICY_JSON --quiet --ipc=$ipc --net=$net --pid=$pid --userns=$userns --uts=$uts --cgroupns=$cgroupns alpine
              assert "$output" != "" "output from buildah-from"
              ctr="$output"
              run_buildah run $ctr pwd
              assert "$output" != "" "output from pwd"
              run_buildah run --tty=true  $ctr pwd
              assert "$output" != "" "output from pwd, with --tty=true"
              run_buildah run --terminal=false $ctr pwd
              assert "$output" != "" "output from pwd, with --terminal=false"
            done
          done
        done
      done
    done
  done
}

@test "idmapping-and-squash" {
        skip_if_rootless_environment
	createrandom ${TEST_SCRATCH_DIR}/randomfile
	run_buildah from --userns-uid-map 0:32:16 --userns-gid-map 0:48:16 scratch
	cid=$output
	run_buildah copy "$cid" ${TEST_SCRATCH_DIR}/randomfile /
	run_buildah copy --chown 1:1 "$cid" ${TEST_SCRATCH_DIR}/randomfile /randomfile2
	run_buildah commit --squash $WITH_POLICY_JSON --rm "$cid" squashed
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

@test "invalid userns-uid-map userns-gid-map" {
	run_buildah 125 from --userns-uid-map 16  --userns-gid-map 0:48:16 scratch
	expect_output 'Error: initializing ID mappings: userns-uid-map setting is malformed expected ["uint32:uint32:uint32"]: ["16"]'

	run_buildah 125 from --userns-uid-map 0:32:16  --userns-gid-map 16 scratch
	expect_output 'Error: initializing ID mappings: userns-gid-map setting is malformed expected ["uint32:uint32:uint32"]: ["16"]'

	run_buildah 125 bud --userns-uid-map a  --userns-gid-map bogus bud/from-scratch
	expect_output 'Error: initializing ID mappings: userns-uid-map setting is malformed expected ["uint32:uint32:uint32"]: ["a"]'

	run_buildah 125 bud --userns-uid-map 0:32:16 --userns-gid-map bogus bud/from-scratch
	expect_output 'Error: initializing ID mappings: userns-gid-map setting is malformed expected ["uint32:uint32:uint32"]: ["bogus"]'

	run_buildah from --userns-uid-map 0:32:16  scratch
}

@test "idmapping-syntax" {
  run_buildah from $WITH_POLICY_JSON --quiet --userns-uid-map=0:10000:65536 alpine

  run_buildah 125 from $WITH_POLICY_JSON --quiet --userns-gid-map=0:10000:65536 alpine
  expect_output --substring "userns-gid-map can not be used without --userns-uid-map"
}

@test "use containers.conf namespace settings" {
  skip_if_chroot

  _prefetch alpine
  containers_conf_file="$TEST_SCRATCH_DIR/containers-namespaces.conf"

  for mode in host private; do
    cat > "$containers_conf_file" << EOF
[containers]

cgroupns = "$mode"
netns = "$mode"
pidns = "$mode"
ipcns = "$mode"
utsns = "$mode"
EOF

    CONTAINERS_CONF="$containers_conf_file" run_buildah from $WITH_POLICY_JSON --quiet alpine
    assert "$output" != "" "output from buildah-from"
    ctr="$output"

    local op="=="
    if [[ "$mode" == "private" ]]; then
      op="!="
    fi

    for nstype in cgroup ipc net pid uts; do
      run readlink /proc/self/ns/"$nstype"
      ns="$output"
      run_buildah run $ctr readlink /proc/self/ns/"$nstype"
      assert "$output" $op "$ns" "namespace matches expected ($mode)"
    done
  done

  rm "$containers_conf_file"
}
