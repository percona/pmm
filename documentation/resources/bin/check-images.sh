#!/bin/bash

set -o errexit nounset

declare -r doc_dir=$(git rev-parse --show-toplevel)/documentation
declare -r exceptions="^PMM.png"
declare -r param="${1:-}"
declare file=""

if [ "$param" != "-r" ]; then
  echo "This script will check for unused images in the documentation."
  echo "If you want to remove unused images, please run the script with the -r parameter."
  echo
else
  echo "Removing unused images..."
  echo
fi

for file in $(ls -1 "${doc_dir}/docs/images"); do
  if ! grep -r --include "*.md" --include "mkdocs*.yml" -m 1 -q $file "${doc_dir}"; then
    if [[ "$file" =~ ${exceptions} ]]; then 
      continue
    fi
    if [ "$param" = "-r" ]; then
      echo "Removing ${doc_dir}/docs/images/${file} ..."
      rm -f "${doc_dir}/docs/images/${file}"
    else
      echo "${doc_dir}/docs/images/${file}"
    fi
  fi
done
