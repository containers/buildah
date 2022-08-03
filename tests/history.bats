#!/usr/bin/env bats

load helpers

function testconfighistory() {
  config="$1"
  expected="$2"
  container=$(echo "c$config" | sed -E -e 's|[[:blank:]]|_|g' -e "s,[-=/:'],_,g" | tr '[A-Z]' '[a-z]')
  image=$(echo "i$config" | sed -E -e 's|[[:blank:]]|_|g' -e "s,[-=/:'],_,g" | tr '[A-Z]' '[a-z]')
  run_buildah from --name "$container" --format docker scratch
  run_buildah config $config --add-history "$container"
  run_buildah commit $WITH_POLICY_JSON "$container" "$image"
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' "$image"
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' "$image"
  expect_output --substring "$expected"
  if test "$3" != "not-oci" ; then
      run_buildah inspect --format '{{range .OCIv1.History}}{{println .CreatedBy}}{{end}}' "$image"
      expect_output --substring "$expected"
  fi
}

@test "history-cmd" {
  testconfighistory "--cmd /foo" "CMD /foo"
}

@test "history-entrypoint" {
  testconfighistory "--entrypoint /foo" "ENTRYPOINT /foo"
}

@test "history-env" {
  testconfighistory "--env FOO=BAR" "ENV FOO=BAR"
}

@test "history-healthcheck" {
  run_buildah from --name healthcheckctr --format docker scratch
  run_buildah config --healthcheck "CMD /foo" --healthcheck-timeout=10s --healthcheck-interval=20s --healthcheck-retries=7 --healthcheck-start-period=30s --add-history healthcheckctr
  run_buildah commit $WITH_POLICY_JSON healthcheckctr healthcheckimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' healthcheckimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' healthcheckimg
  expect_output --substring "HEALTHCHECK --interval=20s --retries=7 --start-period=30s --timeout=10s CMD /foo"
}

@test "history-label" {
  testconfighistory "--label FOO=BAR" "LABEL FOO=BAR"
}

@test "history-onbuild" {
  run_buildah from --name onbuildctr --format docker scratch
  run_buildah config --onbuild "CMD /foo" --add-history onbuildctr
  run_buildah commit $WITH_POLICY_JSON onbuildctr onbuildimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' onbuildimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' onbuildimg
  expect_output --substring "ONBUILD CMD /foo"
}

@test "commit-with-omit-history-set-to-true" {
  run_buildah from --name onbuildctr --format docker scratch
  run_buildah config --onbuild "CMD /foo" --add-history onbuildctr
  run_buildah commit --omit-history $WITH_POLICY_JSON onbuildctr onbuildimg
  run_buildah inspect --format "{{index .Docker.History}}" onbuildimg
  expect_output "[]"
}

@test "history-port" {
  testconfighistory "--port 80/tcp" "EXPOSE 80/tcp"
}

@test "history-shell" {
  testconfighistory "--shell /bin/wish" "SHELL /bin/wish"
}

@test "history-stop-signal" {
  testconfighistory "--stop-signal SIGHUP" "STOPSIGNAL SIGHUP" not-oci
}

@test "history-user" {
  testconfighistory "--user 10:10" "USER 10:10"
}

@test "history-volume" {
  testconfighistory "--volume /foo" "VOLUME /foo"
}

@test "history-workingdir" {
  testconfighistory "--workingdir /foo" "WORKDIR /foo"
}

@test "history-add" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --name addctr --format docker scratch
  run_buildah add --add-history addctr ${TEST_SCRATCH_DIR}/randomfile
  digest="$output"
  run_buildah commit $WITH_POLICY_JSON addctr addimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' addimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' addimg
  expect_output --substring "ADD file:$digest"
}

@test "history-copy" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --name copyctr --format docker scratch
  run_buildah copy --add-history copyctr ${TEST_SCRATCH_DIR}/randomfile
  digest="$output"
  run_buildah commit $WITH_POLICY_JSON copyctr copyimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' copyimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' copyimg
  expect_output --substring "COPY file:$digest"
}

@test "history-run" {
  _prefetch busybox
  run_buildah from --name runctr --format docker $WITH_POLICY_JSON busybox
  run_buildah run --add-history runctr -- uname -a
  run_buildah commit $WITH_POLICY_JSON runctr runimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' runimg
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' runimg
  expect_output --substring "/bin/sh -c uname -a"
}

@test "history should not contain vars in allowlist unless set in ARG" {
  _prefetch busybox
  ctxdir=${TEST_SCRATCH_DIR}/bud
  mkdir -p $ctxdir
  cat >$ctxdir/Dockerfile <<EOF
FROM busybox
RUN echo \$HTTP_PROXY
EOF

  run_buildah build $WITH_POLICY_JSON -t test --build-arg HTTP_PROXY="helloworld" ${ctxdir}
  expect_output --substring 'helloworld'
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' test
  # history should not contain value for HTTP_PROXY since it was not in Containerfile
  assert "$output" !~ 'HTTP_PROXY=helloworld'
  assert "$output" !~ 'helloworld'
}

@test "history should contain vars in allowlist when set in ARG" {
  _prefetch busybox
  ctxdir=${TEST_SCRATCH_DIR}/bud
  mkdir -p $ctxdir
  cat >$ctxdir/Dockerfile <<EOF
FROM busybox
ARG HTTP_PROXY
RUN echo \$HTTP_PROXY
EOF

  run_buildah build $WITH_POLICY_JSON -t test --build-arg HTTP_PROXY="helloworld" ${ctxdir}
  expect_output --substring 'helloworld'
  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' test
  # history should not contain value for HTTP_PROXY since it was not in Containerfile
  expect_output --substring 'HTTP_PROXY=helloworld'
}
