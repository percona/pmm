#!/bin/bash

############################################################
# Help                                                     #
############################################################
Help()
{
   echo "Custom PMM 3 Client installation script. To handle custom installations path and"
   echo "To handle custom installation path and port listening tests"
   echo
   echo "Syntax: pmm3_client_install_tarball.sh [-v X.XX.X] [-p /my/path] [-h|l]"
   echo "options:"
   echo "h     Print this Help."
   echo "v     Installing specified version 3.XX.X or feature build, ex: PR-2734-6fe2553"
   echo "      Also full s3 url to FB tarball is supported"
   echo "p     Installation path. Default: /usr/local/percona/pmm."
   echo "      Sets default version to 3.0.0 if no version specified"
   echo "l     listening custom port mode. Sets default version to 3.0.0 if no version specified"
   echo "u     PMM-Agent can be updated from tarball: run ./install_tarball script with the “-u” flag."
   echo "      The configuration file will not be overwritten with “-u” flag while the pmm-agent is updated."
}

### Defaults
default_path=/usr/local/percona/pmm

# custom path flag and installation are supported from 3.0.0
default_version=3.0.0
min_port_listening_version=3.0.0
port_listening=0
update_flag=""

############################################################
# Process the input options.                               #
############################################################
while getopts "v:p:hlu" option; do
   case $option in
      h) # display Help
        Help
        exit 0
        ;;
      v) # Enter a version
        version=$OPTARG
        ;;
      p) # Enter a custom path
        path=$OPTARG
        ;;
      l) # listening custom port starts from 3.0.0
        port_listening=1
        ;;
      u) # update mode
        update_flag="-u"
        ;;
     \?) # Invalid option
        echo "Error: Invalid option"
        exit 1
        ;;
   esac
done

############################################################
# Post-process args, to ignore the order of the flags      #
############################################################
if [[ -z "${version}" ]]; then
  echo "version is '${version}'"
  if [[ ${port_listening} == 1 ]]; then
    echo ${min_port_listening_version}
    version=${min_port_listening_version}
  else
    version=${default_version}
  fi
else
  ### detect Feature Build code and skip this until folder is extracted
  if [[ "${version}" == *"PR-"* ]]; then
    fb=${version}
  else
    min_ver=$(echo $version | awk -F'.' '{print $2}')
  fi
fi
if [ -z "${path}" ]; then
  path=$default_path
fi
client_tar=pmm-client-${version}.tar.gz
architecture=$(uname -m)
echo $architecture

if [[ "$architecture" == "aarch64" ]]; then
    architecture="arm64"
elif [[ "$architecture" == "x86_64" ]]; then
    architecture="amd64"
fi

if [[ "$architecture" == "arm64" ]]; then
    tarball_url=https://downloads.percona.com/downloads/TESTING/pmm-arm/${client_tar}
elif [[ "$architecture" == "amd64" ]]; then
    tarball_url=https://downloads.percona.com/downloads/TESTING/pmm/${client_tar}
fi

if [ -n "${fb}" ]; then
  client_tar=pmm-client-${fb}.tar.gz
  ### Handle FB url
  if [[ "${fb}" == *"http"* ]]; then
    tarball_url="${fb}"
  else
    tarball_url=https://s3.us-east-2.amazonaws.com/pmm-build-cache/PR-BUILDS/pmm-client/${client_tar}
  fi
fi
### Main program
echo "Downloading ${tarball_url}"
mkdir -p ./tmp/
wget -O ./tmp/pmm-client.tar.gz --progress=dot:giga "${tarball_url}" || exit 1
tar -xvf "./tmp/pmm-client.tar.gz" -C ./tmp/
cd ./tmp
extracted_folder_name=`ls -1td pmm-client* 2>/dev/null | grep -v ".tar" | grep -v ".sh" | head -n1`
echo ${extracted_folder_name}
## for FB extract minor version from folder and check flags are compatible
if [ -n "${fb}" ]; then
  min_ver=$(echo $extracted_folder_name | awk -F'.' '{print $2}')
fi
cd ..
echo "Installing tarball to ${path}"
mkdir -p ${path}
export PMM_DIR=${path}
if [[ $min_ver -lt 30 ]]; then
  cd ./tmp/${extracted_folder_name}
  ./install_tarball ${update_flag}
  cd ../../
else
  ./tmp/${extracted_folder_name}/install_tarball ${update_flag}
fi
ln -sf ${path}/bin/pmm-admin /usr/bin/pmm-admin
ln -sf ${path}/bin/pmm-agent /usr/bin/pmm-agent
rm -rf ./tmp/
echo 'Done!'
