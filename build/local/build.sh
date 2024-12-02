#!/bin/bash -e
set -o errexit
set -o nounset

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
      --release-build         Make it a release, or release candidate build (otherwise it's a feature build)
  -d  --debug                 Turn on a more verbose output mode, useful for troubleshooting
  -h  --help                  Display help

Please note, the script will perform the update of submodules by default on every run unless the '--no-update' option is specified.
EOF
}

parse_params() {
	# All global variables must be defined here
  INITIALIZE=0
	NO_UPDATE=0
	UPDATE_ONLY=0
	NO_CLIENT=0
	NO_CLIENT_DOCKER=0
	NO_SERVER=0
	START_TIME=$(date +%s)
	LOG_FILE="$(dirname $0)/build.log"
	BASE_NAME=$(basename $0)
	PLATFORM=linux/amd64
	SUBMODULES=.modules
  CLONE_BRANCH=v3
	PATH_TO_SCRIPTS="sources/pmm/src/github.com/percona/pmm/build/scripts"
	VAR_PREFIX="__PMM"

	# Exported variables
	export LOCAL_BUILD=1
	export DEBUG_MODE=0
	export ROOT_DIR="$(realpath ./${SUBMODULES})"
	export RPMBUILD_DOCKER_IMAGE=perconalab/rpmbuild:3
  # This replaces the old `RPM_EPOCH=1`, which was used for feature builds
	export RELEASE_BUILD=0
  export BUILD_SUMMARY=""

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

# Function to set a key/value pair
set_value() {
	local key="$1"
	local value="$2"
	validate_key "$key"
	eval "${VAR_PREFIX}_${key}=\"${value}\""
}

# Function to retrieve a value by key
get_value() {
	local key="$1"
	validate_key "$key"
	local value=$(eval "echo -n ${VAR_PREFIX}_${key}")
	if [ -z "$value" ]; then
		echo "Error: variable '${VAR_PREFIX}_${key}' is undefined" >&2
	fi
	echo -n "$value"
}

# Function to unset a key
unset_value() {
	local key="$1"
	validate_key "$key"
	eval "unset ${VAR_PREFIX}_${key}"
}

# Function to validate a key (allow only alphanumeric and underscore characters)
validate_key() {
	local key="$1"
	if ! [[ "$key" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
		echo "The key ${key} is invalid, exiting..." >&2
		exit 1
	fi
}

needs_to_pull() {
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

	if needs_to_pull; then
		if ! git pull origin; then
			git reset --hard HEAD~20
			git pull origin
		fi
		echo "Submodule has pulled from upstream"
		git logs -n 2
		cd - > /dev/null
		git add "$DIR"
	else
		cd - > /dev/null
		echo "Submodule is up-to-date with upstream"
	fi
}

check_files() {
	local DIR="$1"

	# Thouroughly verify the presence of known files, bail out on failure
	if [ ! -d "$DIR" ] ; then
		echo "Error: could not locate the '$SUBMODULES' directory."
    echo "Consider running './build.sh --init' to clone the source code, then try again."
    echo
		exit 1
	fi

	if [ ! -d "$DIR/sources" ] || [ ! -d "$DIR/.git" ] || [ ! -f "$DIR/.gitmodules" ] || [ ! -f "$DIR/ci.py" ]; then
		echo "Error: directory $DIR does not look like a clone of https://github.com/percona-lab/pmm-submodules repository, exiting..."
    echo
		exit 1
	fi

	if [ ! -s "ci.yml" ]; then
		echo "Error: the current directory '$PWD' must contain a non-empty ci.yml file."
		echo "Please refer to the following [README](https://github.com/Percona-Lab/pmm-submodules/blob/v3/README.md#how-to-create-a-feature-build) for more information."
    echo
		exit 1
	fi
}

# Update submodules and PR branches
update() {
	local DEPS=
	local CURDIR="$PWD"

  if [ "$NO_UPDATE" -eq 1 ]; then
    echo "Info: skip refreshing the source code from upstream repositories..."
    return
  fi

	cat <<-'EOF' > entrypoint.sh
		#!/bin/bash -e

		git config --global --add safe.directory /app

		rm -f gitmodules.yml
		python ci.py --convert

		DEPS=$(yq -o=json eval-all '. as $item ireduce ({}; . *d $item )' gitmodules.yml ci.yml | jq '.deps')
		DEPS=$(echo "$DEPS" | jq -r '[.[] | {name: .name, branch: .branch, path: .path, url: .url}]')
		echo "$DEPS"
	EOF

	chmod +x "$CURDIR/entrypoint.sh"
	# Join the dependencies from ci-default.yml and ci.yml
	DEPS=$(
		docker run --rm --platform="$PLATFORM" \
      -v $ROOT_DIR:/app \
			-v $CURDIR/ci.yml:/app/ci.yml \
			-v $CURDIR/entrypoint.sh:/entrypoint.sh \
      -w /app \
			--entrypoint=/entrypoint.sh \
			"$RPMBUILD_DOCKER_IMAGE"
	)

	echo "$DEPS" > sbom.json

	rm -f "$CURDIR/entrypoint.sh"

	echo
	echo "This script rewinds submodule branches as per the joint config of '.gitmodules' and 'ci.yml'"

	cd "$SUBMODULES"

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
	local path=$(git config -f .gitmodules submodule.${module}.path)

	if [ ! -d "$path" ]; then
		echo "Error: could not resolve the path to submodule ${module}"
		exit 1
	fi

	echo $(git -C "$path" branch --show-current)
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
	local CURDIR=$PWD
	local PMM_DIR="build/source/pmm"
	local tmp_files

	cd "$SUBMODULES" > /dev/null
	# Remove stale files and directories
	if [ -d build ]; then
		echo "Removing stale files and directories..."

		if [ -d "build/pmm-server" ]; then
			tmp_files=$(find build/pmm-server | grep -v "RPMS" | grep -Ev "^build/pmm-server$" || :)
			if [ -n "$tmp_files" ]; then
				tmp_files=( $tmp_files )
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
		rm -rvf build/{rpm,srpm,tarball,source_tarball,docker,pmm-client.properties}
	fi
	
	cd "$CURDIR"
}

check_volumes() {
	local CURDIR="$PWD"

	if [ -d "$SUBMODULES" ]; then
		cd "$SUBMODULES" > /dev/null
	fi

	# Create docker volumes to persist package and build cache
	# Read more in the section about `rpmbuild`.
	for volume in pmm-gobuild pmm-gomod pmm-yarn pmm-dnf; do
		if ! docker volume ls | grep "$volume" >/dev/null; then
			docker volume create "$volume"
		else
			echo "Info: docker volume $volume checked."
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

	cd "$CURDIR" > /dev/null
}

initialize() {
  local CURDIR="$PWD"
  local NPROCS=$(getconf _NPROCESSORS_ONLN)

  if [ -d "$SUBMODULES" ]; then
    echo "Info: the source code has already been cloned to '$SUBMODULES', exiting..."
    return
  fi

  git clone --branch "$CLONE_BRANCH" git@github.com:/Percona-Lab/pmm-submodules.git "$SUBMODULES"
  cd "$SUBMODULES" > /dev/null
  git submodule update --init --jobs ${NPROCS:-2}
  git submodule status

  echo "Info: the source code has been cloned to '$SUBMODULES'."
  echo

  echo "Pulling the docker image $RPMBUILD_DOCKER_IMAGE ..."
  docker pull --platform="$PLATFORM" "$RPMBUILD_DOCKER_IMAGE"

  cd "$CURDIR" > /dev/null
}

check_if_installed() {
  local cmd="$1"
  if ! command -v "$cmd" &> /dev/null; then
    echo "Error: $cmd is not installed, exiting..."
    echo
    return 1
  fi

  return 0
}

check_preprequisites() {
  local commands=("docker" "make" "bash" "tar" "git" "curl" "jq")
  echo "Checking pre-requisites..."
  for cmd in "${commands[@]}"; do
    check_if_installed "$cmd"
  done

  if ! docker buildx version &> /dev/null; then
    echo "Error: 'docker buildx plugin is not installed, exiting..."
    echo
    exit 1
  fi
}

cleanup() {
	local CURDIR="$PWD"
	cd "$SUBMODULES" > /dev/null

	# Implement cleanup logic here as/if necessary

	cd "$CURDIR" > /dev/null
}

main() {
	# All global variables are declared in `parse_params`
	parse_params "$@"

	# Capture the build logs in the log file
	exec > >(tee "$LOG_FILE") 2>&1

  check_preprequisites

	if [ "$INITIALIZE" -eq 1 ]; then
    initialize
    exit 0
  fi
  
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

		# Build client binary: ??? from scratch, 0m20s using cache
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
		run_build_script build-server-rpm pmm-update pmm
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
  echo "Total execution time: $(print_duration $(($(date +%s) - $START_TIME)))"
	echo ---

	cleanup
}

main "$@"
