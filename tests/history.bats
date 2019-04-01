#!/usr/bin/env bats

load helpers

function testconfighistory() {
  config="$1"
  expected="$2"
  container=$(echo "c$config" | sed -E -e 's|[[:blank:]]|_|g' -e "s,[-=/:'],_,g" | tr '[A-Z]' '[a-z]')
  image=$(echo "i$config" | sed -E -e 's|[[:blank:]]|_|g' -e "s,[-=/:'],_,g" | tr '[A-Z]' '[a-z]')
  buildah from --name "$container" --format docker scratch
  buildah config $config --add-history "$container"
  buildah commit --signature-policy ${TESTSDIR}/policy.json "$container" "$image"
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' "$image"
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' "$image" | grep "$expected"
  if test "$3" != "not-oci" ; then
    buildah inspect --format '{{range .OCIv1.History}}{{println .CreatedBy}}{{end}}' "$image" | grep "$expected"
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
  buildah from --name healthcheckctr --format docker scratch
  buildah config --healthcheck "CMD /foo" --healthcheck-timeout=10s --healthcheck-interval=20s --healthcheck-retries=7 --healthcheck-start-period=30s --add-history healthcheckctr
  buildah commit --signature-policy ${TESTSDIR}/policy.json healthcheckctr healthcheckimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' healthcheckimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' healthcheckimg | grep "HEALTHCHECK --interval=20s --retries=7 --start-period=30s --timeout=10s CMD /foo"
}

@test "history-label" {
  testconfighistory "--label FOO=BAR" "LABEL FOO=BAR"
}

@test "history-onbuild" {
  buildah from --name onbuildctr --format docker scratch
  buildah config --onbuild "CMD /foo" --add-history onbuildctr
  buildah commit --signature-policy ${TESTSDIR}/policy.json onbuildctr onbuildimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' onbuildimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' onbuildimg | grep "ONBUILD CMD /foo"
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
  createrandom ${TESTDIR}/randomfile
  buildah from --name addctr --format docker scratch
  run_buildah add --add-history addctr ${TESTDIR}/randomfile
  digest="$output"
  buildah commit --signature-policy ${TESTSDIR}/policy.json addctr addimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' addimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' addimg | grep "ADD file:$digest"
}

@test "history-copy" {
  createrandom ${TESTDIR}/randomfile
  buildah from --name copyctr --format docker scratch
  run_buildah --debug=false copy --add-history copyctr ${TESTDIR}/randomfile
  digest="$output"
  buildah commit --signature-policy ${TESTSDIR}/policy.json copyctr copyimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' copyimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' copyimg | grep "COPY file:$digest"
}

@test "history-run" {
  buildah from --name runctr --format docker --signature-policy ${TESTSDIR}/policy.json busybox
  buildah run --add-history runctr -- uname -a
  buildah commit --signature-policy ${TESTSDIR}/policy.json runctr runimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' runimg
  buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' runimg | grep "/bin/sh -c uname -a"
}
