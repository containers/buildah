#!/usr/bin/env bats

load helpers

@test "from-flags-order-verification" {
  run_buildah 125 from scratch -q
  check_options_flag_err "-q"

  run_buildah 125 from scratch --pull
  check_options_flag_err "--pull"

  run_buildah 125 from scratch --ulimit=1024
  check_options_flag_err "--ulimit=1024"

  run_buildah 125 from scratch --name container-name-irrelevant
  check_options_flag_err "--name"

  run_buildah 125 from scratch --cred="fake fake" --name small
  check_options_flag_err "--cred=fake fake"
}

@test "commit-to-from-elsewhere" {
  elsewhere=${TESTDIR}/elsewhere-img
  mkdir -p ${elsewhere}

  run_buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${elsewhere}
  run_buildah rm $cid

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere}
  expect_output "elsewhere-img-working-container"
  run_buildah rm $output

  run_buildah from --quiet --pull-always --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere}
  expect_output "$(basename ${elsewhere})-working-container"
  run_buildah rm $output

  run_buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${elsewhere}
  run_buildah rm $cid

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere}
  expect_output "elsewhere-img-working-container"
  run_buildah rm $output

  run_buildah from --quiet --pull-always --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere}
  expect_output "$(basename ${elsewhere})-working-container"
}

@test "from-tagged-image" {
  # Github #396: Make sure the container name starts with the correct image even when it's tagged.
  run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" scratch2
  run_buildah rm $cid
  run_buildah tag scratch2 scratch3
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch3
  expect_output "scratch3-working-container"
  run_buildah rm $output
  run_buildah rmi scratch2 scratch3

  # Github https://github.com/containers/buildah/issues/396#issuecomment-360949396
  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm $cid
  run_buildah tag alpine alpine2
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json localhost/alpine2
  expect_output "alpine2-working-container"
  run_buildah rm $output
  run_buildah rmi alpine alpine2

  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
  run_buildah rm $output
  run_buildah rmi docker.io/alpine

  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json docker.io/alpine:latest
  run_buildah rm $output
  run_buildah rmi docker.io/alpine:latest

  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json docker.io/centos:7
  run_buildah rm $output
  run_buildah rmi docker.io/centos:7

  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json docker.io/centos:latest
  run_buildah rm $output
  run_buildah rmi docker.io/centos:latest
}

@test "from the following transports: docker-archive, oci-archive, and dir" {
  _prefetch alpine
  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah rm $output

  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json docker:latest
  run_buildah rm $output

  run_buildah push --signature-policy ${TESTSDIR}/policy.json alpine docker-archive:${TESTDIR}/docker-alp.tar:alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json alpine    oci-archive:${TESTDIR}/oci-alp.tar:alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json alpine            dir:${TESTDIR}/alp-dir
  run_buildah rmi alpine

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/docker-alp.tar
  expect_output "alpine-working-container"
  run_buildah rm ${output}
  run_buildah rmi alpine

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/oci-alp.tar
  expect_output "alpine-working-container"
  run_buildah rm ${output}
  run_buildah rmi alpine

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/alp-dir
  expect_output "alp-dir-working-container"
}

@test "from the following transports: docker-archive and oci-archive with no image reference" {
  _prefetch alpine
  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah rm $output

  run_buildah push --signature-policy ${TESTSDIR}/policy.json alpine docker-archive:${TESTDIR}/docker-alp.tar
  run_buildah push --signature-policy ${TESTSDIR}/policy.json alpine    oci-archive:${TESTDIR}/oci-alp.tar
  run_buildah rmi alpine

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/docker-alp.tar
  expect_output "docker-archive-working-container"
  run_buildah rm $output
  run_buildah rmi -a

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/oci-alp.tar
  expect_output "oci-archive-working-container"
  run_buildah rm $output
  run_buildah rmi -a
}

@test "from cpu-period test" {
  skip_if_chroot
  skip_if_rootless
  skip_if_no_runtime
  skip_if_cgroupsv2

  _prefetch alpine
  run_buildah from --quiet --cpu-period=5000 --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid cat /sys/fs/cgroup/cpu/cpu.cfs_period_us
  expect_output "5000"
}

@test "from cpu-quota test" {
  skip_if_chroot
  skip_if_rootless
  skip_if_no_runtime
  skip_if_cgroupsv2

  _prefetch alpine
  run_buildah from --quiet --cpu-quota=5000 --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid cat /sys/fs/cgroup/cpu/cpu.cfs_quota_us
  expect_output "5000"
}

@test "from cpu-shares test" {
  skip_if_chroot
  skip_if_rootless
  skip_if_no_runtime
  skip_if_cgroupsv2

  _prefetch alpine
  run_buildah from --quiet --cpu-shares=2 --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid cat /sys/fs/cgroup/cpu/cpu.shares
  expect_output "2"
}

@test "from cpuset-cpus test" {
  skip_if_chroot
  skip_if_rootless
  skip_if_no_runtime
  skip_if_cgroupsv2 "cgroupsv2: fails with EPERM on writing cpuset.cpus"

  _prefetch alpine
  run_buildah from --quiet --cpuset-cpus=0 --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid cat /sys/fs/cgroup/cpuset/cpuset.cpus
  expect_output "0"
}

@test "from cpuset-mems test" {
  skip_if_chroot
  skip_if_rootless
  skip_if_no_runtime
  skip_if_cgroupsv2 "cgroupsv2: fails with EPERM on writing cpuset.mems"

  _prefetch alpine
  run_buildah from --quiet --cpuset-mems=0 --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid cat /sys/fs/cgroup/cpuset/cpuset.mems
  expect_output "0"
}

@test "from memory test" {
  skip_if_chroot
  skip_if_rootless

  _prefetch alpine
  run_buildah from --quiet --memory=40m --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output

  # Life is much more complicated under cgroups v2
  mpath='/sys/fs/cgroup/memory/memory.limit_in_bytes'
  if is_cgroupsv2; then
      mpath="/sys/fs/cgroup\$(awk -F: '{print \$3}' /proc/self/cgroup)/memory.max"
  fi
  run_buildah run $cid sh -c "cat $mpath"
  expect_output "41943040" "$mpath"
}

@test "from volume test" {
  skip_if_no_runtime

  _prefetch alpine
  run_buildah from --quiet --volume=${TESTDIR}:/myvol --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid -- cat /proc/mounts
  expect_output --substring " /myvol "
}

@test "from volume ro test" {
  skip_if_chroot
  skip_if_no_runtime

  _prefetch alpine
  run_buildah from --quiet --volume=${TESTDIR}:/myvol:ro --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid -- cat /proc/mounts
  expect_output --substring " /myvol "
}

@test "from shm-size test" {
  skip_if_chroot
  skip_if_no_runtime

  _prefetch alpine
  run_buildah from --quiet --shm-size=80m --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run $cid -- df -h /dev/shm
  expect_output --substring " 80.0M "
}

@test "from add-host test" {
  skip_if_no_runtime

  _prefetch alpine
  run_buildah from --quiet --add-host=localhost:127.0.0.1 --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah run --net=container $cid -- cat /etc/hosts
  expect_output --substring "127.0.0.1 +localhost"
}

@test "from name test" {
  _prefetch alpine
  container_name=mycontainer
  run_buildah from --quiet --name=${container_name} --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah inspect --format '{{.Container}}' ${container_name}
}

@test "from cidfile test" {
  _prefetch alpine
  run_buildah from --cidfile output.cid --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$(cat output.cid)
  run_buildah containers -f id=${cid}
}

@test "from pull never" {
  run_buildah 125 from --signature-policy ${TESTSDIR}/policy.json --pull-never busybox
  echo "$output"
  expect_output --substring "no such image"

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --pull=false busybox
  echo "$output"
  expect_output --substring "busybox-working-container"

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --pull-never busybox
  echo "$output"
  expect_output --substring "busybox-working-container"
}

@test "from pull false no local image" {
  _prefetch busybox
  target=my-busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --pull=false busybox
  echo "$output"
  expect_output --substring "busybox-working-container"
}

@test "from with nonexistent authfile: fails" {
  run_buildah 125 from --authfile /no/such/file --pull --signature-policy ${TESTSDIR}/policy.json alpine
  expect_output "error checking authfile path /no/such/file: stat /no/such/file: no such file or directory"
}

@test "from --pull-always: emits 'Getting' even if image is cached" {
  _prefetch docker.io/busybox
  run buildah pull --signature-policy ${TESTSDIR}/policy.json docker.io/busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name busyboxc --pull-always docker.io/busybox
  expect_output --substring "Getting"
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json busyboxc fakename-img
  run_buildah 125 from --signature-policy ${TESTSDIR}/policy.json --pull-always fakename-img
}

@test "from --quiet: should not emit progress messages" {
  # Force a pull. Normally this would say 'Getting image ...' and other
  # progress messages. With --quiet, we should see only the container name.
  run_buildah '?' rmi busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --quiet docker.io/busybox
  expect_output "busybox-working-container"
}

@test "from encrypted local image" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl genrsa -out ${TESTDIR}/tmp/mykey2.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword --encryption-key jwe:${TESTDIR}/tmp/mykey.pub busybox oci:${TESTDIR}/tmp/busybox_enc

  # Try encrypted image without key should fail
  run_buildah 125 from oci:${TESTDIR}/tmp/busybox_enc
  expect_output --substring "Error decrypting layer .* missing private key needed for decryption"

  # Try encrypted image with wrong key should fail
  run_buildah 125 from --decryption-key ${TESTDIR}/tmp/mykey2.pem oci:${TESTDIR}/tmp/busybox_enc
  expect_output --substring "Error decrypting layer .* no suitable key unwrapper found or none of the private keys could be used for decryption"

  # Providing the right key should succeed
  run_buildah from  --decryption-key ${TESTDIR}/tmp/mykey.pem oci:${TESTDIR}/tmp/busybox_enc

  rm -rf ${TESTDIR}/tmp
}

@test "from encrypted registry image" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl genrsa -out ${TESTDIR}/tmp/mykey2.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword --encryption-key jwe:${TESTDIR}/tmp/mykey.pub busybox docker://localhost:5000/buildah/busybox_encrypted:latest

  # Try encrypted image without key should fail
  run_buildah 125 from --tls-verify=false --creds testuser:testpassword docker://localhost:5000/buildah/busybox_encrypted:latest
  expect_output --substring "Error decrypting layer .* missing private key needed for decryption"

  # Try encrypted image with wrong key should fail
  run_buildah 125 from --tls-verify=false --creds testuser:testpassword --decryption-key ${TESTDIR}/tmp/mykey2.pem docker://localhost:5000/buildah/busybox_encrypted:latest
  expect_output --substring "Error decrypting layer .* no suitable key unwrapper found or none of the private keys could be used for decryption"

  # Providing the right key should succeed
  run_buildah from --tls-verify=false --creds testuser:testpassword --decryption-key ${TESTDIR}/tmp/mykey.pem docker://localhost:5000/buildah/busybox_encrypted:latest
  run_buildah rmi localhost:5000/buildah/busybox_encrypted:latest

  rm -rf ${TESTDIR}/tmp
}

@test "from --log-basic: should not emit progress messages" {
  # Force a pull. Normally this would say 'Getting image ...' and other
  # progress messages. With --qlog-basic, we should see only that 
  # the image was pulled, and not progress messages
  run_buildah '?' rmi busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --log-basic docker.io/busybox

  expect_output --substring "Pulled image docker.io/library/busybox:latest"
  expect_line_count 2
}
