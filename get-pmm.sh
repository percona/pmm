#!/usr/bin/env bash
#
# ###############################
# Script to run PMM 3.
# If docker is not installed, this script will try to install it as root user.
# if PMM 2 is running, it will be stopped and data will be migrated to PMM 3.
#
# Usage example:
# curl -fsSL https://raw.githubusercontent.com/percona/pmm/main/get-pmm.sh -o get-pmm.sh; chmod +x get-pmm.sh; ./get-pmm.sh -b
#
#################################

set -Eeuo pipefail
trap cleanup SIGINT SIGTERM ERR EXIT

# Set defaults.
network_name=${NETWORK_NAME:-pmm-net}
tag=${PMM_TAG:-3}
repo=${PMM_REPO:-percona/pmm-server}
port=${PMM_PORT:-443}
container_name=${CONTAINER_NAME:-pmm-server}
docker_socket_path=${DOCKER_SOCKET_PATH:-/var/run/docker.sock}
watchtower_token=${WATCHTOWER_TOKEN:-}
backup_data=0
interactive=0
root_is_needed=no

#######################################
# Show script usage info.
#######################################
usage() {
  cat <<EOF
Usage: $(basename "${BASH_SOURCE[0]}") [-h] [-v] [-i] [-t] [-n] [-p] [-r] [-b] [-wt] [-dsp] [-nc] [-net] [--] [args]

Script description here.

Available options:

-h, --help          Print this help and exit
-v, --verbose       Print script debug info
-i, --interactive   Run script in inteactive mode
-nc, --no-color     Disable colors

-t, --tag="$tag"
       PMM server container tag (default: latest - PMM server version)

-r, --repo="$repo"
       PMM server container repo (default percona/pmm-server)

-p, --port=${port}
      Port number to start PMM server on (default: 443)

-n, --name=${container_name}
      PMM server container name (default: pmm-server)

-b, --backup
      Backup data from existing PMM Server instance

-wt, --watchtower-token
      Watchtower token to use for PMM Server updates

-dsp, --docker-socket-path
      Path to docker socket (default: /var/run/docker.sock)

-net, --network-name
      Name of the network to create (default: pmm-net)
EOF
  exit
}

#######################################
# Clean up setup if interrupt.
#######################################
cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
}

#######################################
# Defines colours for output messages.
#######################################
setup_colors() {
  if [[ -t 2 ]] && [[ -z "${NO_COLOR-}" ]] && [[ "${TERM-}" != "dumb" ]]; then
    NOFORMAT='\033[0m' RED='\033[0;31m' GREEN='\033[0;32m' ORANGE='\033[0;33m'
    BLUE='\033[0;34m' PURPLE='\033[0;35m' CYAN='\033[0;36m' YELLOW='\033[1;33m'
  else
    NOFORMAT='' RED='' GREEN='' ORANGE='' BLUE='' PURPLE='' CYAN='' YELLOW=''
  fi
}

#######################################
# Prints message to stderr with new line at the end.
#######################################
msg() {
  echo >&2 -e "${1-}"
}

#######################################
# Prints message and exit with code.
# Arguments:
#   message string;
#   exit code.
# Outputs:
#   writes message to stderr.
#######################################
die() {
  local message=$1
  local code=${2-1} # default exit status 1
  msg "$message"
  exit "$code"
}

#######################################
# Accept and parse script's params.
#######################################
parse_params() {
  while :; do
    case "${1-}" in
    -h | --help) usage ;;
    -v | --verbose) set -x ;;
    -nc | --no-color) NO_COLOR=1 ;;
    -i | --interactive) interactive=1 ;;
    -t | --tag)
      tag="${2-}"
      shift
      ;;
    -r | --repo)
      repo="${2-}"
      shift
      ;;
    -p | --port)
      port="${2-}"
      shift
      ;;
    -n | --name)
      container_name="${2-}"
      shift
      ;;
    -b | --backup) backup_data=1 ;;
    -wt | --watchtower-token)
      watchtower_token="${2-}"
      shift
      ;;
    -dsp | --docker-socket-path)
      docker_socket_path="${2-}"
      shift
      ;;
    -net | --network-name)
      network_name="${2-}"
      shift
      ;;
    -?*) die "Unknown option: $1" ;;
    *) break ;;
    esac
    shift
  done

  args=("$@")

  return 0
}

#######################################
# Gathers PMM setup param in interactive mode.
#######################################
gather_info() {
  msg "${GREEN}PMM Server Wizard Install${NOFORMAT}"
  default_port=$port
  default_container_name=$container_name
  default_tag=$tag
  read -p "  Port Number to start PMM Server on (default: $default_port): " port
  : ${port:=$default_port}
  read -p "  PMM Server Container Name (default: $default_container_name): " container_name
  : ${container_name:="$default_container_name"}
  read -p "  Override specific version (container tag) (default: $default_tag in 3.x series) format: 3.x.y: " tag
  : ${tag:=$default_tag}
}

check_command() {
  command -v "$@" 1>/dev/null
}

#######################################
# Runs command as root.
#######################################
run_root() {
  sh='sh -c'
  if [ "$(id -un)" != 'root' ]; then
    if check_command sudo; then
      sh='sudo -E sh -c'
    elif check_command su; then
      sh='su -c'
    else
      die "${RED}ERROR: root rights needed to run "$*" command${NOFORMAT}"
    fi
  fi
  ${sh} "$@"
}

#######################################
# Check if MacOS
#######################################
is_darwin() {
   case "$(uname -s)" in
     *darwin* | *Darwin* ) true ;;
     * ) false;;
   esac
}

#######################################
# Installs docker if needed.
#######################################
install_docker() {
  printf "Checking docker installation"
  if ! check_command docker; then
    if is_darwin; then
      echo
      echo "ERROR: Cannot auto-install components on MacOS"
      echo "Please get Docker Desktop from https://www.docker.com/products/docker-desktop and rerun installer after starting"
      echo
      exit 1
    fi
    printf " - not installed. Installing...\n\n"
    curl -fsSL get.docker.com -o /tmp/get-docker.sh ||
      wget -qO /tmp/get-docker.sh get.docker.com
    sh /tmp/get-docker.sh
    run_root 'service docker start' || :
  else
    printf " - installed.\n\n"
  fi

  if ! docker ps 1>/dev/null; then
    root_is_needed='yes'
    if ! run_root 'docker ps > /dev/null'; then
      if is_darwin; then
        run_root 'open --background -a Docker'
        echo "Giving Docker Desktop time to start"
        sleep 30
      else
        die "${RED}ERROR: cannot run "docker ps" command${NOFORMAT}"
      fi
    fi
  fi
}

#######################################
# Runs docker command as root if required.
#######################################
run_docker() {
  if [ "${root_is_needed}" = 'yes' ]; then
    run_root "docker $*"
  else
    sh -c "docker $*"
  fi
}

#######################################
# Generates Watchtower token if needed, or reuses existing one.
#######################################
generate_watchtower_token() {
  if [ !  "$watchtower_token" == "" ]; then
    msg "Using provided Watchtower token"
    return 0
  fi
  if run_docker "inspect watchtower 1> /dev/null 2> /dev/null"; then
    watchtower_token=$(run_docker "inspect --format='{{range .Config.Env}}{{println .}}{{end}}' watchtower | grep WATCHTOWER_HTTP_API_TOKEN | cut -d'=' -f2")
    msg "Found Watchtower Token: $watchtower_token"
    return 0
  fi
  if run_docker "inspect pmm-server 1> /dev/null 2> /dev/null"; then
    watchtower_token=$(run_docker "inspect --format='{{range .Config.Env}}{{println .}}{{end}}' pmm-server | grep PMM_WATCHTOWER_TOKEN | cut -d'=' -f2")
    msg "Found Watchtower Token: $watchtower_token"
    # we don't return here, as we want to generate a new token if it's not found
  fi
  if [ "$watchtower_token" == "" ]; then
    watchtower_token=random-$(date "+%F-%H%M%S")
    msg "Generated Watchtower Token: $watchtower_token"
  fi
}

#######################################
# Creates PMM Network if needed.
#######################################
create_pmm_network() {
  if ! run_docker "network inspect $network_name 1> /dev/null 2> /dev/null"; then
    run_docker "network create $network_name 1> /dev/null"
    msg "Created PMM Network: $network_name"
  fi
}

#######################################
# Starts Watchtower container if needed.
#######################################
start_watchtower() {
  if ! run_docker "inspect watchtower 1> /dev/null 2> /dev/null"; then
    run_docker "run -d --name watchtower --restart always --network $network_name -e WATCHTOWER_HTTP_API_TOKEN=$watchtower_token -e WATCHTOWER_HTTP_LISTEN_PORT=8080 -e WATCHTOWER_HTTP_API_UPDATE=1 -v $docker_socket_path:/var/run/docker.sock perconalab/watchtower --cleanup"
    msg "Created Watchtower container"
  fi
}

#######################################
# Migrates PMM Server data from 2.x to 3.x
#######################################
migrate_pmm_data() {
  msg "Migrating PMM Server data from 2.x to 3.x"
  run_docker "start $container_name"
  sleep 5
  run_docker "exec -t $container_name supervisorctl stop all"
  sleep 5
  run_docker "exec -t $container_name chown -R pmm:pmm /srv"
  run_docker "stop $container_name"
}

#######################################
# Backs up existing PMM Data Volume.
#######################################
backup_pmm_data() {
  pmm_volume_archive="pmm-data-$(date "+%F-%H%M%S")"
  msg "Backing up existing PMM Data Volume to $pmm_volume_archive"
  run_docker "volume create $pmm_volume_archive 1> /dev/null"
  run_docker "run --rm -v pmm-data:/from -v $pmm_volume_archive:/to alpine ash -c 'cd /from ; cp -av . /to'"
  msg "Successfully backed up existing PMM Data Volume to $pmm_volume_archive"
}

#######################################
# Starts PMM Server container with given repo, tag, name and port.
# If a PMM Server instance is running - stop and back it up.
#######################################
start_pmm() {
  msg "Starting PMM Server..."
  run_docker "pull $repo:$tag 1> /dev/null"

  if ! run_docker "inspect pmm-data 1> /dev/null 2> /dev/null"; then
    if ! run_docker "volume create pmm-data 1> /dev/null"; then
      die "${RED}ERROR: cannot create PMM Data Volume${NOFORMAT}"
    fi
    msg "Created PMM Data Volume: pmm-data"
  fi

  if run_docker "inspect $container_name 1> /dev/null 2> /dev/null"; then
    pmm_archive="$container_name-$(date "+%F-%H%M%S")"
    msg "\tExisting PMM Server found, renaming to $pmm_archive\n"
    run_docker "stop $container_name" || :
    if [[ "$backup_data" == 1 ]]; then
      backup_pmm_data
    fi
    # get container tag from inspect
    old_version=$(run_docker "inspect --format='{{.Config.Image}}' $container_name | cut -d':' -f2")
    # if tag starts with 2.x, we need to migrate data
    if [[ "$old_version" == "2" || "$old_version" == 2.* || "$old_version" == "dev-latest" ]]; then
      migrate_pmm_data
    fi
    run_docker "rename $container_name $pmm_archive\n"
  fi
  run_pmm="run -d -p $port:8443 --volume pmm-data:/srv --name $container_name --network $network_name -e PMM_WATCHTOWER_HOST=http://watchtower:8080 -e PMM_WATCHTOWER_TOKEN=$watchtower_token --restart always $repo:$tag"

  run_docker "$run_pmm 1> /dev/null"
  msg "Created PMM Server: $container_name"
  msg "\nUse the following command if you ever need to update your container manually:"
  msg "\tdocker pull $repo:$tag \n"
  msg "\tdocker $run_pmm \n"
}

#######################################
# Shows final message.
# Shows a list of addresses on which PMM Server is available.
#######################################
show_message() {
  msg "PMM Server has been successfully setup on this system!\n"

  if check_command ifconfig; then
    ips=$(ifconfig | awk '/inet / {print $2}' | sed 's/addr://')
  elif check_command ip; then
    ips=$(ip -f inet a | awk -F"[/ ]+" '/inet / {print $3}')
  else
    die "${RED}ERROR: cannot detect PMM Server address${NOFORMAT}"
  fi

  msg "You can access your new server using one of the following web addresses:"
  for ip in $ips; do
    msg "\t${GREEN}https://$ip:$port/${NOFORMAT}"
  done
  msg "\nThe default username is '${PURPLE}admin${NOFORMAT}' and the password is '${PURPLE}admin${NOFORMAT}' :)"
  msg "Note: Some browsers may not trust the default SSL certificate when you first open one of the urls above."
  msg "If this is the case, Chrome users may want to type '${PURPLE}thisisunsafe${NOFORMAT}' to bypass the warning.\n"
}

main() {
  setup_colors
  if [[ "$interactive" == 1 ]]; then
    gather_info
  fi
  msg "Gathering/downloading required components, this may take a moment\n"
  install_docker
  create_pmm_network
  generate_watchtower_token
  start_pmm
  start_watchtower
  show_message
}

parse_params "$@"

main
die "Enjoy Percona Monitoring and Management!" 0
# TODO: Update script from PMM 2 to PMM 3

}# TODO: reuse watchtower token from older pmm instance or watchtower