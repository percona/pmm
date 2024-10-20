#!/bin/bash -e
set -o errexit
set -o nounset

usage() {
  cat <<-EOF
Usage: $BASE_NAME [--no-update | --update-only] [--no-client] [--no-client-docker] [--no-server-rpm] [--no-server-docker] [--log-file <path>] [--help | -h]
--no-update              Do not fetch the latest changes from the repo
--update-only            Only fetch the latest changes from the repo
--no-client              Do not build PMM Client
--client-docker          Build PMM Client docker image
--no-server-rpm          Do not build Server RPM packages
--no-server-docker       Do not build PMM Server docker image
--log-file <path>        Save build logs to a file located at <path>
--help | -h              Display help
EOF
}

parse-params() {
  # Define global variables
  NO_UPDATE=0
  UPDATE_ONLY=0
  NO_CLIENT=0
  NO_CLIENT_DOCKER=1
  NO_SERVER_RPM=0
  NO_SERVER_DOCKER=0
  START_TIME=$(date +%s)
  LOG_FILE="$(dirname $0)/build.log"
  BASE_NAME=$(basename $0)
  SUBMODULES=pmm-submodules
  PATH_TO_SCRIPTS="sources/pmm/src/github.com/percona/pmm/build/scripts"

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
      --client-docker)
        NO_CLIENT_DOCKER=0
        ;;
      --no-server-rpm)
        if [ "$NO_SERVER_DOCKER" -eq 1 ]; then
          echo "Error: cannot disable both server RPM and server Docker"
          exit 1
        fi
        NO_SERVER_RPM=1
        ;;
      --no-server-docker)
        if [ "$NO_SERVER_RPM" -eq 1 ]; then
          echo "Error: cannot disable both server RPM and server Docker"
          exit 1
        fi
        NO_SERVER_DOCKER=1
        ;;
      --log-file)
        shift
        if [ -z "$1" ]; then
          echo "Missing argument for --log-file"
          exit 1
        fi
        LOG_FILE="$1"
        ;;
      --help | -h)
        shift
        usage
        exit 0
        ;;
      *)
        echo "Unknown argument: $1"
        usage
        exit 1
        ;;
    esac
    shift
  done
}

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

  cd "$DIR" > /dev/null
  local CURRENT=$(git branch --show-current)
  git fetch

  if [ "$CURRENT" != "$BRANCH" ]; then
    echo "Currently on $CURRENT, checking out $BRANCH"
    git checkout "$BRANCH"
  fi

  if needs-to-pull; then
    git pull origin
    echo "Submodule has pulled from upstream"
    git logs -n 2
    cd - > /dev/null
    git add "$DIR"
  else
    cd - > /dev/null
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

  # Thouroughly verify the presence of known files, otherwise bail out
  if [ ! -d "$SUBMODULES" ] ; then # pwd must outside of pmm-submodules
    echo "Warn: the current working directory must be outside of pmm-submodules"
    echo "cd .."
    cd .. > /dev/null
  fi

  if [ -d "$SUBMODULES" ]; then # pwd is outside pmm-submodules
    if ! check-files "$SUBMODULES"; then
      echo "Fatal: could not locate known files in ${PWD}/${SUBMODULES}"
      exit 1
    fi
  else
    echo "Fatal: could not locate known files in $PWD"
    exit 1
  fi

  cd "$SUBMODULES"

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
  local CURDIR="$PWD"
  local script="$PATH_TO_SCRIPTS/$1"
  local start_time
  local end_time
  local script_name="$1"

  cd "$SUBMODULES" > /dev/null

  if [ ! -f "$script" ]; then
    echo "Fatal: script $script does not exist"
    cd "$CURDIR" > /dev/null
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

  cd "$CURDIR" > /dev/null
}

purge_files() {
  local CURDIR=$PWD
  local tmp_files

  cd "$SUBMODULES" > /dev/null
  # Remove stale files and directories
  if [ -d tmp ]; then
    echo "Removing stale files and directories..."

    if [ -d "tmp/pmm-server" ]; then
      tmp_files=$(find tmp/pmm-server | grep -v "RPMS" | grep -Ev "^tmp/pmm-server$" || :)
      if [ -n "$tmp_files" ]; then
        tmp_files=( $tmp_files )
        for f in "${tmp_files[@]}"; do
          rm -rf "$f"
        done
      fi
    fi

    if [ -d "tmp/source/pmm" ]; then
      rm -rf tmp/source/pmm
    fi
  fi

  if [ -f "$LOG_FILE" ]; then
    echo "Removing the log file..."
    rm -f $LOG_FILE
  fi

  cd "$CURDIR"
}

init() {
  local CURDIR="$PWD"

  export RPMBUILD_DOCKER_IMAGE=perconalab/rpmbuild:3

  if [ -d "$SUBMODULES" ]; then
    cd "$SUBMODULES" > /dev/null
  fi

  GIT_COMMIT=$(git rev-parse HEAD | head -c 8)

  pmm_commit=$(git submodule status | grep 'sources/pmm/src' | awk -F ' ' '{print $1}')
  pmm_branch=$(git config -f .gitmodules submodule.pmm.branch)
  pmm_url=$(git config -f .gitmodules submodule.pmm.url)
  pmm_qa_branch=$(git config -f .gitmodules submodule.pmm-qa.branch)
  pmm_qa_commit=$(git submodule status | grep 'pmm-qa' | awk -F ' ' '{print $1}')
  pmm_ui_tests_branch=$(git config -f .gitmodules submodule.pmm-ui-tests.branch)
  pmm_ui_tests_commit=$(git submodule status | grep 'pmm-ui-tests' | awk -F ' ' '{print $1}')
  fb_commit_sha=$(git rev-parse HEAD)

  echo $fb_commit_sha > fbCommitSha
  echo $pmm_commit > apiCommitSha
  echo $pmm_branch > apiBranch
  echo $pmm_url > apiURL
  echo $pmm_qa_branch > pmmQABranch
  echo $pmm_qa_commit > pmmQACommitSha
  echo $pmm_ui_tests_branch > pmmUITestBranch
  echo $pmm_ui_tests_commit > pmmUITestsCommitSha

  # Create cache directories. Read more in the section about `rpmbuild`.
  test -d "go-path" || mkdir -p "go-path"
  test -d "go-build" || mkdir -p "go-build"
  test -d "yarn-cache" || mkdir -p "yarn-cache"
  test -d "yum-cache" || mkdir -p "yum-cache"

  cd "$CURDIR" > /dev/null
}

cleanup() {
  local CURDIR="$PWD"
  cd "$SUBMODULES" > /dev/null

  # Clean up temporary files
  rm -f apiBranch \
    apiCommitSha \
    apiURL \
    fbCommitSha \
    pmmQABranch \
    pmmQACommitSha \
    pmmUITestBranch \
    pmmUITestsCommitSha || :

  cd "$CURDIR" > /dev/null
}

main() {
  if [ "$NO_UPDATE" -eq 0 ]; then
    local UPDATED_SCRIPT="$SUBMODULES/$PATH_TO_SCRIPTS/build/local/build.sh"
    MD5SUM=$(md5sum $(dirname $0)/build.sh)

    # Update submodules and PR branches
    update

    test "$UPDATE_ONLY" -eq 1 && return

    if [ -f "$UPDATED_SCRIPT" ] && [ "$MD5SUM" != "$(md5sum $UPDATED_SCRIPT)" ]; then
      echo "The local copy of this script differs from the one fetched from the repo." 
      echo "Apparently, that version is newer. We will halt to give you the change to run a fresh version."
      echo "You can copy it over and run it again, i.e. '/bin/bash $(dirname $0)/build.sh --no-update'"
      return
    fi
  fi

  init

  purge_files

  if [ "$NO_CLIENT" -eq 0 ]; then
    # Build client source: 4m39s from scratch, 0m27s using cache
    build_with_logs build-client-source

    # Build client binary: ??? from scratch, 0m20s using cache
    build_with_logs build-client-binary

    # Building client source rpm takes 13s (caching does not apply)
    # build_with_logs build-client-srpm

    # Building client rpm takes 1m40s
    # build_with_logs build-client-rpm
  fi

  # Building client docker image takes 17s
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

  if [ "$NO_SERVER_DOCKER" -eq 0 ]; then
    export DOCKER_TAG=percona/pmm-server:${GIT_COMMIT}
    export DOCKERFILE=Dockerfile.el9
    build_with_logs build-server-docker
  fi

  echo
  echo "Done building PMM artifacts."
  echo ---
  echo "Total execution time, sec: $(($(date +%s) - $START_TIME))" | tee -a $LOG_FILE
  echo ---

  cleanup
}

# Reference test environment
# CPU: 4 cores
# RAM: 16 GB
# OS: Ubuntu 22.04.1 LTS

parse-params "$@"

main
