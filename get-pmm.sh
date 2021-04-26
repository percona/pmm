#!/usr/bin/env bash
#
# ###############################
# Script to run PMM 2.
# If docker is not installed - this script try to install it as root user.
#
# Usage example:
# curl -fsSL https://raw.githubusercontent.com/percona/pmm/PMM-2.0/get-pmm.sh -o get-pmm2.sh; chmod +x get-pmm2.sh; ./get-pmm2.sh
#
#################################

set -Eeuo pipefail
trap cleanup SIGINT SIGTERM ERR EXIT

# Set defaults.
tag=${PMM_TAG:-"latest"}
repo=${PMM_REPO:-"percona/pmm-server"}
port=${PMM_PORT:-443}
container_name=${CONTAINER_NAME:-"pmm-server"}
interactive=0
root_is_needed='no'

#######################################
# Show script usage info.
#######################################
usage() {
  cat <<EOF
Usage: $(basename "${BASH_SOURCE[0]}") [-h] [-v] [-i] [-t] [-n] [-p]

Script description here.

Available options:

-h, --help          Print this help and exit
-v, --verbose       Print script debug info
-i, --interactive   Run script in inteactive mode

-t, --tag="$tag"
       PMM server container tag (default: latest - PMM server version)

-r, --repo="$repo"
       PMM server container repo (default percona/pmm-server)

-p, --port=${port}
      Port number to start PMM server on (default: 443)

-n, --name=${container_name}
      PMM server container name (default: pmm-server)
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
  local msg=$1
  local code=${2-1} # default exit status 1
  msg "$msg"
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
    --no-color) NO_COLOR=1 ;;
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
  msg "\tPort Number to start PMM Server on (default: 443): "
  read port
  msg "\tPMM Server Container Name (default: pmm-server): "
  read container_name
  msg "\tOverride specific version (container tag) (default: latest in 2.x series) format: 2.x.y: "
  read tag
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
# Installs docker if needed.
#######################################
install_docker() {
  msg "Checking if docker is installed..."
  if ! check_command docker; then
    msg "Installing docker..."
    curl -fsSL get.docker.com -o /tmp/get-docker.sh ||
      wget -qO /tmp/get-docker.sh get.docker.com
    sh /tmp/get-docker.sh
    run_root 'service docker start' || :
  fi
  if ! docker ps &>/dev/null; then
    root_is_needed='yes'
    if ! run_root 'docker ps &> /dev/null'; then
      die "${RED}ERROR: cannot run "docker ps" command${NOFORMAT}"
    fi
  fi
  msg "Docker is ready."
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
# Starts PMM server container with give repo, tag, name and port.
# If any PMM server instance is run - stop and backup it.
#######################################
start_pmm() {
  msg "Starting PMM server..."
  run_docker "pull $repo:$tag 1> /dev/null"

  if ! run_docker "inspect pmm-data &> /dev/null"; then
    run_docker "create -v /srv/ --name pmm-data percona/pmm-server:$tag /bin/true 1> /dev/null"
    msg "Created PMM Data Volume: pmm-data"
  fi

  if run_docker "inspect pmm-server &> /dev/null"; then
    pmm_archive="pmm-server-$(date "+%F-%H%M%S")"
    msg "\tExisting PMM Server found, renaming to $pmm_archive"
    run_docker 'stop pmm-server' || :
    run_docker "rename pmm-server $pmm_archive"
  fi
  run_pmm="run -d -p $port:443 --volumes-from pmm-data --name $container_name --restart always $repo:$tag"

  run_docker "$run_pmm &> /dev/null"
  msg "Created PMM Server: $container_name"
  msg "\tUse the following command if you ever need to update your container by hand:"
  msg "\tdocker $run_pmm"
}

#######################################
# Shows final message.
# Shows a list of addresses on which PMM server available.
#######################################
show_message() {
  msg "PMM Server has been successfully setup on this system!"
  msg "You can access your new server using the one of following web addresses:"
  for ip in $(ifconfig | grep "inet " | awk '{print $2}'); do
    msg "\t${BLUE}https://$ip:$port/${NOFORMAT}"
  done
  msg "The default username is '${PURPLE}admin${NOFORMAT}' and password is '${PURPLE}admin${NOFORMAT}'"
  msg "**Note** Browser may not trust the default SSL certificate on first load."
  msg "So type '${PURPLE}thisisunsafe${NOFORMAT}' to bypass their warning"
}

main() {
  parse_params "$@"
  setup_colors
  if [[ $interactive == 1 ]]; then
    gather_info
  fi
  msg "Gathering/Downloading required components, this may take a moment"
  install_docker
  start_pmm
  show_message
}

main
die "Done!" 0
