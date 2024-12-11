#!/bin/bash

set -o errexit nounset

declare -r doc_dir=$(git rev-parse --show-toplevel)/documentation
declare -r img_dir="${doc_dir}/docs/images"
declare -r exceptions="^PMM.png"
declare -r action="${ACTION:-}"
declare file=""

if [ "$action" != "remove" ]; then
  echo "Checking for unused images in documentation..."
  echo
else
  echo "Removing unused images from documentation..."
  echo
fi

for file in $(ls -1 "${img_dir}"); do
  if ! grep -r --include "*.md" --include "mkdocs*.yml" -m 1 -q $file "${doc_dir}"; then
    if [[ "$file" =~ ${exceptions} ]]; then 
      continue
    fi
    if [ "$action" = "remove" ]; then
      echo "Removing ${img_dir}/${file} ..."
      rm -f "${img_dir}/${file}"
    else
      echo "${img_dir}/${file}"
    fi
  fi
done
