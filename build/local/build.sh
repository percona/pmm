#!/bin/bash -eu

usage() {
  cat <<-EOF
Usage: $BASE_NAME [OPTIONS]
Options:
      --init                  Clone the source, initialize directories, check for pre-requisites and exit
      --platform <platform>   Build for a specific platform (defaults to linux/amd64)
      --no-update             Do not fetch the latest changes from the repo before building
      --update-only           Just fetch the latest changes from the repo and exit
      --client-only           Build only PMM Client (client binaries + docker)
      --no-client             Do not build PMM Client (this will use local cache)
      --no-client-docker      Do not build PMM Client docker image
      --log-file <path>       Save build logs to a file located at <path> (defaults to $PWD/build.log)
                              Note: the log file will get reset on every subsequent run
      --release-build         Mark it as release, or release candidate build (otherwise it's a feature build)
      --clean                 Remove the build state and all related files, except 'ci.yml'
  -d  --debug                 Turn on a more verbose output mode, useful for troubleshooting
  -h  --help                  Display help

Please note, the script will perform the update of submodules by default on every run unless the '--no-update' option is specified.
EOF
}

print_error() {
  echo
  echo "Error: $1" >&2
}

parse_params() {
  # All global variables must be defined here
  INITIALIZE=0
  NO_UPDATE=0
  UPDATE_ONLY=0
  NO_CLIENT=0
  NO_CLIENT_DOCKER=0
  NO_SERVER=0
  CLEAN=0
  START_TIME=$(date +%s)
  LOG_FILE="$(dirname $0)/build.log"
  BASE_NAME=$(basename $0)
  PLATFORM=linux/amd64
  SUBMODULES=""
  CLONE_BRANCH=v3
  BRANCH_NAME=""
  PATH_TO_SCRIPTS="sources/pmm/src/github.com/percona/pmm/build/scripts"

  if ! test -L build.sh; then
    print_error "the script must be run from the root of github.com/percona/pmm repository."
    exit 1
  fi

  if ! cat .modules 2> /dev/null; then
    local TMPDIR
    TMPDIR=$(mktemp -d)
    echo -n "$TMPDIR" > .modules
    SUBMODULES="$TMPDIR"
  else
    SUBMODULES=$(cat .modules)
  fi

  # Exported variables
  export USE_S3_CACHE=0
  export DEBUG_MODE=0
  export RPMBUILD_DOCKER_IMAGE
  RPMBUILD_DOCKER_IMAGE=$([ -n "${CI:-}" ] && echo "public.ecr.aws/e7j3v3n0/rpmbuild:3" || echo "perconalab/rpmbuild:3")
  # This replaces the old `RPM_EPOCH=1`, which was used for feature builds
  export RELEASE_BUILD=0
  export BUILD_SUMMARY=""
  # This is used in "build/scripts/vars"
  export ROOT_DIR="$SUBMODULES"

  while test "$#" -gt 0; do
    case "$1" in
      --init)
        INITIALIZE=1
        ;;
      --update-only)
        UPDATE_ONLY=1; NO_UPDATE=0
        ;;
      --no-update)
        if [ "$UPDATE_ONLY" -eq 1 ]; then
          echo "Error. Mutually exclusive options: --update-only and --no-update"
          exit 1
        fi      
        NO_UPDATE=1
        ;;
      --client-only)
        NO_CLIENT=0; NO_CLIENT_DOCKER=0; NO_SERVER=1
        ;;
      --no-client)
        NO_CLIENT=1; NO_CLIENT_DOCKER=1
        ;;
      --no-client-docker)
        if [ "$NO_CLIENT" -eq 1 ]; then
          echo "Error. Mutually exclusive options: --client-docker and --no-client"
          exit 1
        fi
        NO_CLIENT_DOCKER=1
        ;;
      --platform)
        shift
        if [ -z "$1" ]; then
          echo "Missing argument for --platform"
          exit 1
        fi
        PLATFORM="$1"
        ;;
      --log-file)
        shift
        if [ -z "$1" ]; then
          echo "Missing argument for --log-file"
          exit 1
        fi
        LOG_FILE="$1"
        ;;
      --release-build)
        RELEASE_BUILD=1
        ;;
      --debug | -d)
        DEBUG_MODE=1
        ;;
      --clean)
        CLEAN=1
        ;;
      --help | -h)
        usage
        exit 0
        ;;
      *)
        echo "Unknown argument: $1"
        echo
        usage
        exit 1
        ;;
    esac
    shift
  done
}

check_files() {
  local DIR="$1"

  # Thouroughly verify the presence of known files, bail out on failure
  if [ ! -d "$DIR" ] ; then
    print_error "could not locate the '$SUBMODULES' directory, exiting..."
    exit 1
  fi

  if [ ! -d "$DIR/sources" ] || [ ! -d "$DIR/.git" ] || [ ! -f "$DIR/.gitmodules" ] || [ ! -f "$DIR/ci.py" ]; then
    print_error "the contents of directory $DIR do not look like a clone of https://github.com/percona-lab/pmm-submodules repository, exiting..."
    exit 1
  fi

  # We set this global var here, since git may not be availabe in the `parse_params` function
  # The value must be taken from percona/pmm repository
  BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
  if [ -z "$BRANCH_NAME" ]; then
    print_error "could not determine the current branch name, exiting..."
    exit 1
  fi

  if [[ "$BRANCH_NAME" =~ ^main$|^v3$ ]] && [ "$RELEASE_BUILD" -eq 0 ]; then
    print_error "you are not on a feature branch, but on '$BRANCH_NAME'."
    echo "Please make sure to create a feature branch before proceeding."
    exit 1
  fi

  if [ ! -s "ci.yml" ]; then
    echo
    echo "Info: since the current directory '$PWD' does not contain a 'ci.yml' file with project dependencies,"
    echo "we will create a default configuration by searching for the current branch name in all repositories."
    echo
    if [ -z "${CI:-}" ]; then
      echo "Pausing for 10 seconds to allow you to cancel the operation in case you want to create the file manually..."
      echo
      sleep 10
      echo "To learn more about the file format, please refer to the following [README](https://github.com/Percona-Lab/pmm-submodules/blob/v3/README.md#how-to-create-a-feature-build)."
      echo
    fi

    echo -n > ci.yml
  fi

  mkdir -p "$DIR/build"

  # Get the PR number and the commit hash for feature builds
  if [ "$RELEASE_BUILD" -eq 0 ]; then
    local FB_COMMIT
    FB_COMMIT=$(git rev-parse HEAD)
    local PR_NUMBER
    PR_NUMBER=$(git ls-remote origin 'refs/pull/*/head' | grep "${FB_COMMIT}" | awk -F'/' '{print $3}')
    local TAG
    if [ -n "$PR_NUMBER" ]; then
      TAG="PR-${PR_NUMBER}-${FB_COMMIT:0:7}"
    else
      TAG="FB-${FB_COMMIT:0:7}"
    fi
    echo -n "$PR_NUMBER" > "$DIR/build/PR_NUMBER"
    export DOCKER_CLIENT_TAG=perconalab/pmm-client-fb:${TAG}
    export DOCKER_TAG=perconalab/pmm-server-fb:${TAG}
  fi
}

# Update submodules and PR branches
update() {
  local CURDIR="$PWD"

  if [ "$NO_UPDATE" -eq 1 ]; then
    echo
    echo "Info: not refreshing the source code from upstream repositories."
    return
  fi

  echo
  echo "This script rewinds submodule branches as per the joint config of '.gitmodules' and the user-supplied 'ci.yml'."

  docker run --rm --platform="$PLATFORM" \
    -v "$SUBMODULES:/app" \
    -v "$CURDIR/ci.yml:/app/ci.yml" \
    -v "$CURDIR/build/local/ci.py:/app/ci.py" \
    -v "$CURDIR/build/local/entrypoint.sh:/entrypoint.sh" \
    -w /app \
    -e BRANCH_NAME="$BRANCH_NAME" \
    --entrypoint=/entrypoint.sh \
    "$RPMBUILD_DOCKER_IMAGE"

  if [ ! -s "$SUBMODULES/build/build.json" ]; then
    print_error "could not find '$SUBMODULES/build/build.json' file, which means that the build failed, exiting..."
    exit 1
  fi

  cd "$SUBMODULES"

  echo
  echo "Printing git status..."
  git status --short

  echo
  echo "Printing git submodule status..."
  git submodule status

  cd "$CURDIR" > /dev/null

  if [ "$UPDATE_ONLY" -eq 1 ]; then
    exit 0
  fi
}

get_branch_name() {
  local module="${1:-}"
  local path
  path=$(git config -f ".gitmodules submodule.${module}.path")

  if [ ! -d "$path" ]; then
    print_error "could not resolve the path to submodule ${module}"
    exit 1
  fi

  git -C "$path" branch --show-current
}

print_duration() {
  local sec="$1"
  local min=$((sec / 60))
  local sec=$((sec % 60))
  echo "${min}m${sec}s"
}

run_build_script() {
  local CURDIR="$PWD"
  local script="$PATH_TO_SCRIPTS/$1"
  local script_name="$1"
  local start_time
  local end_time
  local duration

  cd "$SUBMODULES" > /dev/null

  if [ ! -f "$script" ]; then
    echo "Fatal: script $script does not exist."
    cd "$CURDIR" > /dev/null
    exit 1
  fi

  start_time=$(date +%s)
  if [ "$#" -gt 1 ]; then
    shift
    script_name="${script_name}:($1)"
    $script "$@"
  else
    $script
  fi
  end_time=$(date +%s)
  duration=$((end_time - start_time))

  echo ---
  echo "Execution time for $script_name: $(print_duration $duration)"
  echo ---

  cd "$CURDIR" > /dev/null
}

purge_files() {
  local CURDIR="$PWD"
  local PMM_DIR="build/source/pmm"
  local tmp_files

  cd "$SUBMODULES" > /dev/null
  if [ -d build ]; then
    echo
    echo "Removing stale files and directories..."

    if [ -d "build/pmm-server" ]; then
      tmp_files=$(find build/pmm-server | grep -v "RPMS" | grep -Ev "^build/pmm-server$" || :)
      if [ -n "$tmp_files" ]; then
        # Use read to properly split the output into an array
        readarray -t tmp_files <<< "$tmp_files"
        for f in "${tmp_files[@]}"; do
          echo "Removing file or directory $f ..."
          rm -rf "$f"
        done
      fi
    fi

    if [ -d "$PMM_DIR" ]; then
      echo "Removing $PMM_DIR ..."
      rm -rf "$PMM_DIR"
    fi

    echo "Removing build/* ..."
    rm -rvf build/{rpm,srpm,binary,tarball,source_tarball,docker,pmm-client.properties}
  fi
  
  cd "$CURDIR"
}

check_volumes() {
  # Create docker volumes to persist package and build cache
  # Read more in the section about `rpmbuild`.
  echo
  echo "Checking Docker volumes..."
  for volume in pmm-gobuild pmm-gomod pmm-yarn pmm-dnf; do
    if ! docker volume ls | grep "$volume" >/dev/null; then
      docker volume create "$volume" > /dev/null
      echo "Docker volume $volume created."
    else
      echo "Docker volume $volume checked."
    fi
  done

  docker run --rm --platform="$PLATFORM" \
    -v pmm-gobuild:/home/builder/.cache/go-build \
    -v pmm-gomod:/home/builder/go/pkg/mod \
    -v pmm-yarn:/home/builder/.cache/yarn \
    -v pmm-dnf:/var/cache/dnf \
    "$RPMBUILD_DOCKER_IMAGE" sh -c "
      sudo chown builder:builder /home/builder/.cache
      if [ ! -d /home/builder/.cache/go-build ]; then
        mkdir -p /home/builder/.cache/go-build
      fi
      if [ ! -d /home/builder/go ]; then
        mkdir -p /home/builder/go/pkg/mod
      fi        
      if [ ! -w /home/builder/.cache/go-build ]; then
        sudo chown builder:builder /home/builder/.cache/go-build
      fi
      if [ ! -w /home/builder/go/pkg/mod ]; then
        sudo chown builder:builder /home/builder/go/pkg/mod
      fi
      if [ ! -w /home/builder/.cache/yarn ]; then
        sudo chown builder:builder /home/builder/.cache/yarn
      fi
    "
  echo "Docker volumes are ready."
}

initialize() {
  local CURDIR="$PWD"
  local NPROCS
  NPROCS=$(getconf _NPROCESSORS_ONLN 2>/dev/null)

  if [ -d "$SUBMODULES" ] && [ -f "$SUBMODULES/VERSION" ]; then
    echo
    echo "Info: the source code is located in '$SUBMODULES'."
    return
  fi

  echo
  echo "Checking out the source code, it may take a while..."
  git clone --branch "$CLONE_BRANCH" git@github.com:/Percona-Lab/pmm-submodules.git "$SUBMODULES"
  cd "$SUBMODULES" > /dev/null
  git submodule update --init --jobs "${NPROCS:-2}"
  git submodule status

  echo
  echo "Info: the source code has been checked out to '$SUBMODULES'."

  echo
  echo "Pulling the docker image $RPMBUILD_DOCKER_IMAGE ..."
  docker pull --platform="$PLATFORM" "$RPMBUILD_DOCKER_IMAGE"

  cd "$CURDIR" > /dev/null

  if [ "$INITIALIZE" -eq 1 ]; then
    exit 0
  fi
}

check_if_installed() {
  local cmd="$1"
  if ! command -v "$cmd" &> /dev/null; then
    print_error "$cmd is not installed, exiting..."
    return 1
  fi

  return 0
}

check_preprequisites() {
  local commands=("docker" "make" "bash" "tar" "git" "curl" "find")
  echo "Checking pre-requisites..."
  for cmd in "${commands[@]}"; do
    check_if_installed "$cmd"
  done

  if ! docker buildx version &> /dev/null; then
    print_error "docker buildx plugin is not installed, exiting..."
    exit 1
  fi

  echo "Pre-requisites check passed."
}

cleanup() {
  local CURDIR="$PWD"
  cd "$SUBMODULES" > /dev/null

  # Implement cleanup logic here as/if necessary

  cd "$CURDIR" > /dev/null
}

main() {
  # All global variables are declared in `parse_params` for this script,
  # and in `scripts/vars` for the dependent build scripts.
  parse_params "$@"

  if [ "$CLEAN" -eq 1 ]; then
    echo "Cleaning up the build state..."
    rm -rf "$SUBMODULES/build" build.log
    exit 0
  fi

  # Capture the build logs in the log file
  exec > >(tee "$LOG_FILE") 2>&1

  check_preprequisites

  initialize
  
  check_files "$SUBMODULES"

  check_volumes

  update

  purge_files

  if [ "$DEBUG_MODE" -eq 1 ]; then
    set -o xtrace
  fi

  if [ "$NO_CLIENT" -eq 0 ]; then
    # Build client source: 4m39s from scratch, 0m27s using cache
    run_build_script build-client-source

    # Build client binary: 6m40s from scratch, 0m20s using cache
    run_build_script build-client-binary

    # Building client source rpm takes 13s (caching does not apply)
    run_build_script build-client-srpm

    # Building client rpm takes 1m40s
    run_build_script build-client-rpm
  fi

  # Building client docker image takes from 17s (using docker cache) to 43s (no docker cache).
  if [ "$NO_CLIENT_DOCKER" -eq 0 ]; then
    run_build_script build-client-docker
  fi

  if [ "$NO_SERVER" -eq 0 ]; then
    # 1st-party components
    run_build_script build-server-rpm percona-dashboards grafana-dashboards
    run_build_script build-server-rpm pmm-managed pmm
    run_build_script build-server-rpm pmm-ui pmm
    run_build_script build-server-rpm pmm-qan-api pmm
    run_build_script build-server-rpm pmm-dump
    run_build_script build-server-rpm pmm-vmproxy pmm

    # 3rd-party components
    run_build_script build-server-rpm victoriametrics
    run_build_script build-server-rpm grafana

    run_build_script build-server-docker
  fi

  set +o xtrace
  echo
  echo "Done building PMM artifacts."
  echo ---
  echo "Total execution time: $(print_duration $(($(date +%s) - START_TIME)))"
  echo ---

  cleanup
}

main "$@"
