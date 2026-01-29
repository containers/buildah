#!/usr/bin/env bats

load helpers

@test "source-policy: DENY rule blocks base image" {
  # Create a policy that denies alpine
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "DENY",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine:latest"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
RUN echo hello
EOF

  # Build should fail with source policy denial
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file $policyfile -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "denied by source policy"
}

@test "source-policy: DENY rule with WILDCARD match" {
  # Create a policy that denies all ubuntu images
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "DENY",
      "selector": {
        "identifier": "docker-image://docker.io/library/ubuntu:*",
        "matchType": "WILDCARD"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile using ubuntu
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM ubuntu:22.04
RUN echo hello
EOF

  # Build should fail with source policy denial
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file $policyfile -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "denied by source policy"
}

@test "source-policy: CONVERT rule rewrites base image to pinned digest" {
  _prefetch alpine

  # Get the digest of the alpine image
  run_buildah inspect --format '{{.FromImageDigest}}' alpine
  alpine_digest="$output"

  # Create a policy that converts alpine:latest to the pinned digest
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << EOF
{
  "rules": [
    {
      "action": "CONVERT",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine:latest"
      },
      "updates": {
        "identifier": "docker-image://docker.io/library/alpine@${alpine_digest}"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
RUN echo converted
EOF

  imgname="img-$(safename)"

  # Build should succeed with the converted reference
  run_buildah build $WITH_POLICY_JSON --source-policy-file $policyfile -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: ALLOW rule explicitly allows source" {
  _prefetch alpine

  # Create a policy that explicitly allows alpine
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "ALLOW",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine:latest"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
RUN echo allowed
EOF

  imgname="img-$(safename)"

  # Build should succeed
  run_buildah build $WITH_POLICY_JSON --source-policy-file $policyfile -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: first matching rule wins" {
  _prefetch alpine

  # Create a policy where ALLOW comes before DENY for the same image
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "ALLOW",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine:latest"
      }
    },
    {
      "action": "DENY",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine:latest"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
RUN echo first-match-wins
EOF

  imgname="img-$(safename)"

  # Build should succeed because ALLOW matches first
  run_buildah build $WITH_POLICY_JSON --source-policy-file $policyfile -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: invalid policy file fails build" {
  # Create an invalid policy file (bad JSON)
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{invalid json}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
EOF

  # Build should fail with parsing error
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file $policyfile -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "loading source policy"
}

@test "source-policy: missing policy file fails build" {
  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
EOF

  # Build should fail with file not found error
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file /nonexistent/policy.json -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "loading source policy"
}

@test "source-policy: scratch base image is not evaluated" {
  # Create a policy that would deny everything
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "DENY",
      "selector": {
        "identifier": "docker-image://*",
        "matchType": "WILDCARD"
      }
    }
  ]
}
EOF

  # Create a simple file to copy into the image
  echo "test content" > ${TEST_SCRATCH_DIR}/testfile.txt

  # Create a Dockerfile using scratch that doesn't reference external images
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM scratch
COPY testfile.txt /testfile.txt
EOF

  imgname="img-$(safename)"

  # Build from scratch should succeed - scratch is not evaluated against policy
  run_buildah build $WITH_POLICY_JSON --source-policy-file $policyfile -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: multi-stage build with stage reference not evaluated" {
  _prefetch alpine

  # Create a policy that denies everything except alpine
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "ALLOW",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine:latest"
      }
    },
    {
      "action": "DENY",
      "selector": {
        "identifier": "docker-image://*",
        "matchType": "WILDCARD"
      }
    }
  ]
}
EOF

  # Create a multi-stage Dockerfile where stage references another stage by name
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest AS builder
RUN echo "building"

FROM builder AS final
RUN echo "final"
EOF

  imgname="img-$(safename)"

  # Build should succeed - stage reference "builder" should not be evaluated against policy
  run_buildah build $WITH_POLICY_JSON --source-policy-file $policyfile -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: no policy file means no restrictions" {
  _prefetch alpine

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
RUN echo "no policy"
EOF

  imgname="img-$(safename)"

  # Build without --source-policy-file should work normally
  run_buildah build $WITH_POLICY_JSON -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: CONVERT with normalized image references" {
  _prefetch alpine

  # Get the digest of the alpine image
  run_buildah inspect --format '{{.FromImageDigest}}' alpine
  alpine_digest="$output"

  # Create a policy that converts short name "alpine" to full reference with digest
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << EOF
{
  "rules": [
    {
      "action": "CONVERT",
      "selector": {
        "identifier": "docker-image://docker.io/library/alpine"
      },
      "updates": {
        "identifier": "docker-image://docker.io/library/alpine@${alpine_digest}"
      }
    }
  ]
}
EOF

  # Create a Dockerfile with short name (no tag)
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine
RUN echo converted
EOF

  imgname="img-$(safename)"

  # Build should succeed with the converted reference
  run_buildah build $WITH_POLICY_JSON --source-policy-file $policyfile -t $imgname -f $dockerfile ${TEST_SCRATCH_DIR}

  # Verify the image was built
  run_buildah inspect $imgname
}

@test "source-policy: policy validation - missing action" {
  # Create a policy missing required action field
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "selector": {
        "identifier": "docker-image://test"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
EOF

  # Build should fail with validation error
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file $policyfile -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "action is required"
}

@test "source-policy: policy validation - missing selector identifier" {
  # Create a policy missing selector identifier
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "DENY",
      "selector": {}
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
EOF

  # Build should fail with validation error
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file $policyfile -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "selector.identifier is required"
}

@test "source-policy: policy validation - CONVERT without updates" {
  # Create a CONVERT policy without updates field
  policyfile=${TEST_SCRATCH_DIR}/policy.json
  cat > $policyfile << 'EOF'
{
  "rules": [
    {
      "action": "CONVERT",
      "selector": {
        "identifier": "docker-image://test"
      }
    }
  ]
}
EOF

  # Create a simple Dockerfile
  dockerfile=${TEST_SCRATCH_DIR}/Dockerfile
  cat > $dockerfile << 'EOF'
FROM alpine:latest
EOF

  # Build should fail with validation error
  run_buildah 125 build $WITH_POLICY_JSON --source-policy-file $policyfile -f $dockerfile ${TEST_SCRATCH_DIR}
  expect_output --substring "updates.identifier is required for CONVERT"
}
