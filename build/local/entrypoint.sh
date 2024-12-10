#!/bin/bash
set -o errexit

git config --global --add safe.directory /app

rm -f gitmodules.yml

# Join the dependencies listed in .gitmodules and ci.yml and output the result to gitmodules.yml.
# This script will not fail even if ci.yml is empty.
if ! python ci.py --convert; then
  echo "Error: could not convert the ci.yml config file to gitmodules.yml, exiting..."
  exit 1
fi

DEPS=$(yq -o=json '.' gitmodules.yml | jq -r '[.deps[] | {name: .name, branch: .branch, path: .path, url: .url}]')
echo "$DEPS" > /app/build/build.json
echo -n > /tmp/deps.txt

DISCOVERED_BRANCHES=()

while read -r item; do
  branch=$(echo "$item" | jq -r '.branch')
  path=$(echo "$item" | jq -r '.path')
  name=$(echo "$item" | jq -r '.name')
  url=$(echo "$item" | jq -r '.url')

  if [ -n "${BRANCH_NAME:-}" ]; then
    commit=$(git ls-remote --heads "$url" "$BRANCH_NAME" | cut -f1)
    if [ -n "$commit" ]; then
      echo
      echo "Info: branch '$BRANCH_NAME' found in '$name' submodule, will use it instead of '$branch'"
      branch="$BRANCH_NAME"
      DISCOVERED_BRANCHES+=( "$name|$branch|$path|$url" )
      echo "name=${name}|path=${path}|url=${url}|branch=${branch}|commit=${commit}" >> /tmp/deps.txt
    else
      echo "name=${name}|path=${path}|url=${url}|branch=${branch}|commit=none" >> /tmp/deps.txt
    fi
  else
    echo "name=${name}|path=${path}|url=${url}|branch=${branch}|commit=none" >> /tmp/deps.txt
  fi
done < <(echo "$DEPS" | jq -c '.[]')

cat /tmp/deps.txt > /app/build/deps.txt

DISCOVERED_FILE="/tmp/discovered.yml"

if [[ "${#DISCOVERED_BRANCHES[@]}" -gt 0 ]]; then
  echo "Generating... $DISCOVERED_FILE"
  echo "deps:" > "$DISCOVERED_FILE"
  for item in "${DISCOVERED_BRANCHES[@]}"; do
    echo "$item" | IFS='|' read -r name branch path url
    echo "  - name: $name" >> "$DISCOVERED_FILE"
    echo "    branch: $branch" >> "$DISCOVERED_FILE"
    echo "    path: $path" >> "$DISCOVERED_FILE"
    echo "    url: $url" >> "$DISCOVERED_FILE"
  done
fi

if [ ! -s "ci.yml" ]; then
  echo
  echo "Info: generating ci.yml..."
  cat "$DISCOVERED_FILE" > ci.yml
fi

rm -f gitmodules.yml
