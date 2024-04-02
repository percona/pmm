#!/bin/bash -e

# Define global variables
NO_UPDATE=0
UPDATE_ONLY=0
NO_CLIENT=0
NO_CLIENT_DOCKER=0
NO_SERVER_RPM=0
START_TIME=$(date +%s)
LOG_FILE="/tmp/build.log"

while test "$#" -gt 0; do
  case "$1" in
    --update-only)
      UPDATE_ONLY=1; NO_UPDATE=0
      ;;
    --no-update)
      NO_UPDATE=1
      ;;
    --no-client)
      NO_CLIENT=1; NO_CLIENT_DOCKER=1
      ;;
    --no-client-docker)
      NO_CLIENT_DOCKER=1
      ;;
    --no-server-rpm)
      NO_SERVER_RPM=1
      ;;
    --log-file)
      shift
      if [ -z "$1" ]; then
        echo "Missing argument for --log-file"
        exit 1
      fi
      LOG_FILE="$1"
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 [--no-update | --update-only] [--no-client] [--no-client-docker] [--no-server-rpm] [--log-file <path>]"
      echo
      exit 1
      ;;
  esac
  shift
done

needs-to-pull() {
  local UPSTREAM=${1:-'@{u}'}
  local LOCAL=$(git rev-parse @)
  local BASE=$(git merge-base @ "$UPSTREAM")
  local REMOTE=$(git rev-parse "$UPSTREAM")

  if [ "$LOCAL" = "$REMOTE" ]; then
    return 1 # false, we are up-to-date
  fi

  if [ "$LOCAL" = "$BASE" ]; then
    return 0 # true, we are behind upstream
  fi
}

rewind() {
  local DIR="$1"
  local BRANCH="$2"

  cd "$DIR"
  CURRENT=$(git branch --show-current)
  git fetch

  if [ "$CURRENT" != "$BRANCH" ]; then
    echo "Currently on $CURRENT, checking out $BRANCH"
    git checkout "$BRANCH"
  fi

  if needs-to-pull; then
    git pull origin
    echo "Submodule has pulled from upstream"
    git logs -n 2
    cd - >/dev/null
    git add "$DIR"
  else
    cd - >/dev/null
    echo "Submodule is up-to-date with upstream"
  fi
}

check-files() {
  local DIR="$1"

  test -z "DIR" && exit 1

  if [ -d "$DIR/sources" ] && [ -f "$DIR/ci-default.yml" ] && [ -f "$DIR/ci.yml" ]; then
    return 0
  fi

  return 1
}

update() {
  local DEPS=
  local CURDIR="$PWD"
  local DIR=pmm-submodules

  # Thouroughly verify the presence of known files, otherwise bail out
  if check-files "."; then # pwd is pmm-submodules
    DIR="."
  elif [ -d "$DIR" ]; then # pwd is outside pmm-submodules
    if ! check-files "$DIR"; then
      echo "Fatal: could not locate known files in ${PWD}/${DIR}"
      exit 1
    fi
  else
    echo "Fatal: could not locate known files in $PWD"
    exit 1
  fi

  cd "$DIR"

  # Join the dependencies from ci-default.yml and ci.yml
  DEPS=$(yq -o=json eval-all '. as $item ireduce ({}; . *d $item )' ci-default.yml ci.yml | jq '.deps')

  echo "This script rewinds submodule branches as per the joint config of 'ci-default.yml' and 'ci.yml'"

  echo "$DEPS" | jq -c '.[]' | while read -r item; do
    branch=$(echo "$item" | jq -r '.branch')
    path=$(echo "$item" | jq -r '.path')
    name=$(echo "$item" | jq -r '.name')
    echo
    echo "Rewinding submodule '$name' ..."
    echo "path: ${path}, branch: ${branch}"

    rewind "$path" "$branch"
  done

  echo
  echo "Printing git status..."
  git status --short
  echo
  echo "Printing git submodule status..."
  git submodule status

  cd "$CURDIR" > /dev/null
}

get_branch_name() {
  local module="${1:-}"
  local branch_name
  local path

  path=$(git config -f .gitmodules submodule.${module}.path)
  cd "$path" || exit 1
  branch_name=$(git branch --show-current)
  cd - > /dev/null
  echo $branch_name
}

build_with_logs() {
  local script="$PATH_TO_SCRIPTS/$1"
  local start_time
  local end_time
  local script_name="$1"

  if [ ! -f "$script" ]; then
    echo "Fatal: script $script does not exist"
    exit 1
  fi

  start_time=$(date +%s)
  if [ "$#" -gt 1 ]; then
    shift
    script_name="${script_name}:($1)"
    $script "$@" | tee -a $LOG_FILE
  else
    $script | tee -a $LOG_FILE
  fi
  end_time=$(date +%s)

  echo ---
  echo "Execution time (in sec) for $script_name: $((end_time - start_time))" | tee -a $LOG_FILE
  echo ---
}

init() {
  local tmp_files
  # Remove stale files and directories
  if [ -d tmp ]; then
    echo "Removing stale files and directories..."
    if [ -d "tmp/pmm-server" ]; then
      tmp_files=$(find tmp/pmm-server | grep -v "RPMS")
      tmp_files=($tmp_files)
      for f in "${tmp_files[@]}"; do
        rm -rf "$f"
      done
    fi
  fi
  if [ -f "$LOG_FILE" ]; then
    echo "Removing the log file..."
    rm -f $LOG_FILE
  fi

  # Define global variables
  pmm_commit=$(git submodule status | grep 'sources/pmm/src' | awk -F ' ' '{print $1}')
  echo $pmm_commit > apiCommitSha
  pmm_branch=$(get_branch_name pmm)
  echo $pmm_branch > apiBranch
  pmm_url=$(git config -f .gitmodules submodule.pmm.url)
  echo $pmm_url > apiURL
  pmm_qa_branch=$(get_branch_name pmm-qa)
  echo $pmm_qa_branch > pmmQABranch
  pmm_qa_commit=$(git submodule status | grep 'pmm-qa' | awk -F ' ' '{print $1}')
  echo $pmm_qa_commit > pmmQACommitSha
  pmm_ui_tests_branch=$(get_branch_name pmm-ui-tests)
  echo $pmm_ui_tests_branch > pmmUITestBranch
  pmm_ui_tests_commit=$(git submodule status | grep 'pmm-ui-tests' | awk -F ' ' '{print $1}')
  echo $pmm_ui_tests_commit > pmmUITestsCommitSha
  fb_commit_sha=$(git rev-parse HEAD)
  echo $fb_commit_sha > fbCommitSha

  PATH_TO_SCRIPTS="sources/pmm/src/github.com/percona/pmm/build/scripts"
  export RPMBUILD_DOCKER_IMAGE=perconalab/rpmbuild:3

  # We use a special docker image to build various PMM artifacts - `perconalab/rpmbuild:3`.
  # Important: the docker container's user needs to be able to write to these directories.
  # The docker container's user is `builder` with uid 1000 and gid 1000. You need to make sure
  # that the directories we create on the host are owned by a user with the same uid and gid.

  # Create cache directories.
  test -d "${root_dir}/go-path" || mkdir -p "go-path"
  test -d "${root_dir}/go-build" || mkdir -p "go-build"
  test -d "${root_dir}/yarn-cache" || mkdir -p "yarn-cache"
  test -d "${root_dir}/yum-cache" || mkdir -p "yum-cache"
}

cleanup() {
  # Clean up temporary files
  rm -f apiBranch \
    apiCommitSha \
    apiURL \
    fbCommitSha \
    pmmQABranch \
    pmmQACommitSha \
    pmmUITestBranch \
    pmmUITestsCommitSha
}

main() {
  if [ "$NO_UPDATE" -eq 0 ]; then
    MD5SUM=$(md5sum $(dirname $0)/build.sh)
    
    # Update submodules and PR branches
    update

    test "$UPDATE_ONLY" -eq 1 && return

    if [ "$MD5SUM" != "$(md5sum $(dirname $0)/build.sh)" ]; then
      echo "The updated version of this script has been fetched from the repository, exiting..."
      echo "Please run it again, i.e. '/bin/bash $(dirname $0)/build.sh --no-update'"
      return
    fi
  fi

  if [ "$NO_CLIENT" -eq 0 ]; then
    # Build client source: 4m39s from scratch, 0m27s using cache
    build_with_logs build-client-source

    # Build client binary: ??? from scratch, 0m20s using cache
    build_with_logs build-client-binary

    # Building client source rpm takes 13s (caching does not apply)
    build_with_logs build-client-srpm

    # Building client rpm takes 1m40s
    build_with_logs build-client-rpm
  fi

  # Building client docker image takes 17s
  GIT_COMMIT=$(git rev-parse HEAD | head -c 8)
  export DOCKER_CLIENT_TAG=local/pmm-client:${GIT_COMMIT}
  if [ "$NO_CLIENT_DOCKER" -eq 0 ]; then
    build_with_logs build-client-docker
  fi

  # Building PMM CLient locally (non-CI, i.e. non-Jenkins)
  # total time: 6m26s - build from scratch, no initial cache
  # total time: 2m49s - subsequent build, using cache from prior builds


  # Building PMM CLient in a CI environment, i.e. Jenkins running on AWS
  # total time: 8m45s - build from scratch, no initial cache
  # total time: ??? - subsequent build, using cache from prior builds

  export RPM_EPOCH=1
  if [ "$NO_SERVER_RPM" -eq 0 ]; then
    build_with_logs build-server-rpm percona-dashboards grafana-dashboards
    build_with_logs build-server-rpm pmm-managed pmm
    build_with_logs build-server-rpm percona-qan-api2 pmm
    build_with_logs build-server-rpm pmm-update pmm
    build_with_logs build-server-rpm pmm-dump
    build_with_logs build-server-rpm vmproxy pmm

    # 3rd-party
    build_with_logs build-server-rpm victoriametrics
    build_with_logs build-server-rpm grafana
  fi

  export DOCKER_TAG=local/pmm-server:${GIT_COMMIT}
  export DOCKERFILE=Dockerfile.el9
  build_with_logs build-server-docker

  echo
  echo "Done building PMM artifacts."
  echo ---
  echo "Total execution time, sec: $(($(date +%s) - $START_TIME))" | tee -a $LOG_FILE
  echo ---
}

# Local reference test environment
# CPU: 4 cores
# RAM: 16GB
# OS: Ubuntu 22.04.1 LTS

init
main
cleanup
