#!/usr/bin/env bash
#
# ###############################
# Script to run PMM 2.
# If docker is not installed, this script will try to install it as root user.
#
# Usage example:
# curl -fsSL https://raw.githubusercontent.com/percona/pmm/main/get-pmm.sh -o get-pmm.sh; chmod +x get-pmm.sh; ./get-pmm.sh
#
#################################

set -Eeuo pipefail
trap cleanup SIGINT SIGTERM ERR EXIT

# Set defaults.
tag=${PMM_TAG:-"2"}
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
  default_port=$port
  default_container_name=$container_name
  default_tag=$tag
  read -p "  Port Number to start PMM Server on (default: $default_port): " port
  : ${port:=$default_port}
  read -p "  PMM Server Container Name (default: $default_container_name): " container_name
  : ${container_name:="$default_container_name"}
  read -p "  Override specific version (container tag) (default: $default_tag in 2.x series) format: 2.x.y: " tag
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
      echo "ERROR: Cannot auto-install components on macOS"
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
        echo "Giving docker desktop time to start"
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
# Starts PMM server container with give repo, tag, name and port.
# If any PMM server instance is run - stop and backup it.
#######################################
start_pmm() {
  msg "Starting PMM server..."
  run_docker "pull $repo:$tag 1> /dev/null"

  if ! run_docker "inspect pmm-data 1> /dev/null 2> /dev/null"; then
    run_docker "create -v /srv/ --name pmm-data $repo:$tag /bin/true 1> /dev/null"
    msg "Created PMM Data Volume: pmm-data"
  fi

  if run_docker "inspect pmm-server 1> /dev/null 2> /dev/null"; then
    pmm_archive="pmm-server-$(date "+%F-%H%M%S")"
    msg "\tExisting PMM Server found, renaming to $pmm_archive\n"
    run_docker 'stop pmm-server' || :
    run_docker "rename pmm-server $pmm_archive\n"
  fi
  run_pmm="run -d -p $port:8443 --volumes-from pmm-data --name $container_name --restart always $repo:$tag"

  run_docker "$run_pmm 1> /dev/null"
  msg "Created PMM Server: $container_name"
  msg "\tUse the following command if you ever need to update your container by hand:"
  msg "\tdocker $run_pmm \n"
}

#######################################
# Shows final message.
# Shows a list of addresses on which PMM server available.
#######################################
show_message() {
  msg "PMM Server has been successfully setup on this system!\n"

  if check_command ifconfig; then
    ips=$(ifconfig | awk '/inet / {print $2}' | sed 's/addr://')
  elif check_command ip; then
    ips=$(ip -f inet a | awk -F"[/ ]+" '/inet / {print $3}')
  else
    die "${RED}ERROR: cannot detect PMM server address${NOFORMAT}"
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
  if [[ $interactive == 1 ]]; then
    gather_info
  fi
  msg "Gathering/downloading required components, this may take a moment\n"
  install_docker
  start_pmm
  show_message
}

parse_params "$@"

main
die "Enjoy Percona Monitoring and Management!" 0
