#!/bin/bash -e
# This script is a self-sufficient way to run PMM API tests. 
# It will launch a PMM Server instance, build the test image, and execute the tests against the server. 
# The test results will be copied to the host machine for review.

pmm-api() {
  # Check if the resources already exist and clean them up
  if docker container inspect pmm-server &>/dev/null; then
    docker rm -vf pmm-server || :
  fi
  if docker volume inspect pmm-data &>/dev/null; then
    docker volume rm pmm-data
  fi
  docker volume create pmm-data
  if docker container inspect pmm-api-tests &>/dev/null; then
    docker rm -vf pmm-api-tests || :
  fi
  if docker image inspect percona/pmm-api-tests &>/dev/null; then
    docker rmi percona/pmm-api-tests
  fi

  # Launch PMM Server
  docker run -d \
    --platform linux/amd64 \
    --name pmm-server \
    --hostname pmm-server \
    -p 443:8443 \
    -e AWS_ACCESS_KEY \
    -e AWS_SECRET_KEY \
    -e PMM_ENABLE_ACCESS_CONTROL=1 \
    -e PMM_ENABLE_TELEMETRY=0 \
    -v pmm-data:/srv \
    "${PMM_SERVER_IMAGE:-perconalab/pmm-server:3-dev-latest}"

  # Build the test image
  docker buildx build --platform=linux/amd64 --progress=plain -t percona/pmm-api-tests .

  until curl -skf https://127.0.0.1/v1/server/readyz &>/dev/null; do echo "Waiting for pmm-server to come up..." && sleep 2; done

  # Create a test database
  pushd api-tests
  docker compose up test_db # no daemon mode
  popd

  # Run API tests in race mode
  docker run \
    --platform linux/amd64 \
    --name pmm-api-tests \
    -e PMM_SERVER_URl=https://admin:admin@127.0.0.1 \
    -e PMM_RUN_UPDATE_TEST=0 \
    -e PMM_RUN_ADVISOR_TESTS=0 \
    -e PMM_SERVER_INSECURE_TLS=1 \
    -v pmm-gomod:/go/pkg/mod \
    --network host \
    percona/pmm-api-tests

  docker cp pmm-api-tests:/go/pmm/api-tests/pmm-api-tests-output.txt . || :
}

pmm-api "$@"
