#!/bin/bash
set -o errexit
set -o nounset

git config --add safe.directory /app

rm -f gitmodules.yml /app/build/build.json

if [ -s "ci.yml" ]; then
  echo
  echo "Info: ci.yml was found, we will use it in combination with the defaults to resolve dependencies."
  echo "If you like to (re)discover the dependencies based on '${BRANCH_NAME}' branch, please remove the 'ci.yml' file."
fi

needs_to_pull() {
  local UPSTREAM='@{u}'
  local LOCAL BASE REMOTE
  LOCAL=$(git rev-parse @)
  BASE=$(git merge-base @ "$UPSTREAM")
  REMOTE=$(git rev-parse "$UPSTREAM")

  if [ "$LOCAL" = "$REMOTE" ]; then
    return 1 # false, we are up-to-date
  fi

  if [ "$LOCAL" = "$BASE" ]; then
    return 0 # true, we are behind remote
  fi
}

rewind() {
  local DIR="$1"
  local BRANCH="$2"
  local NAME="$3"
  local CURRENT

  cd "$DIR" > /dev/null
  CURRENT=$(git rev-parse --abbrev-ref HEAD)
  echo
  echo "Rewinding submodule ${NAME}..."
  git fetch

  if [ "$CURRENT" != "$BRANCH" ]; then
    echo "Currently on $CURRENT, checking out $BRANCH..."
    git checkout "$BRANCH"
  fi

  if needs_to_pull; then
    if ! git pull origin; then
      git reset --hard HEAD~30
      git pull origin > /dev/null
    fi
    echo "Submodule ${NAME} has pulled from remote."
    git log --oneline -n 2
    cd - > /dev/null
    git add "$DIR"
  else
    cd - > /dev/null
    echo "Submodule ${NAME} is up-to-date with remote."
  fi
}

# Join the dependencies listed in .gitmodules and ci.yml and output the result to gitmodules.yml.
# This script accepts an empty ci.yml.
if ! python ci.py --convert; then
  echo "Error: could not run '--convert' and generate 'gitmodules.yml', exiting..."
  exit 1
fi

declare -a discovered_branches=()
declare discovered_file="/tmp/ci.yml"
declare updated_deps="[]"
declare branch path name url commit
declare ci_branch deps
deps=$(yq -o=json '.' gitmodules.yml | jq -r '[.deps[] | {name: .name, branch: .branch, path: .path, url: .url}]')
# Note: BRANCHE_NAME is passed via the environment variable

while read -r item; do
  branch=$(echo "$item" | jq -r '.branch')
  path=$(echo "$item" | jq -r '.path')
  name=$(echo "$item" | jq -r '.name')
  url=$(echo "$item" | jq -r '.url'| sed 's:\.git::')

  # Only run this block if we have a branch and if 'ci.yml' is not present or not empty  
  if [ -n "${BRANCH_NAME:-}" ] && [ ! -s "ci.yml" ]; then
    commit=$(git ls-remote --heads "$url" "$BRANCH_NAME" | cut -f1)
    if [ -n "$commit" ]; then
      echo
      echo "Info: branch '$BRANCH_NAME' found in '$name' submodule, will use it instead of the default '$branch' branch."
      branch="$BRANCH_NAME"
      discovered_branches+=( "$name|$branch|$path|$url" )
    fi
  elif [ -s "ci.yml" ]; then
    ci_branch=$(yq -o=json '.' ci.yml | jq -r --arg name "$name" '.deps[] | select(.name == $name) | .branch')
    if [ -n "$ci_branch" ]; then
      branch="$ci_branch"
    fi
  fi

  rewind "$path" "$branch" "$name"
  commit=$(git -C "$path" rev-parse HEAD)
  url="${url}/tree/${commit}"

  updated_deps=$(echo "$updated_deps" | jq ". += [{name: \"$name\", branch: \"$branch\", commit: \"$commit\", path: \"$path\", url: \"$url\"}]")

done < <(echo "$deps" | jq -c '.[]')

echo "$updated_deps" > /app/build/build.json

if [[ "${#discovered_branches[@]}" -gt 0 ]]; then
  echo "deps:" > "$discovered_file"

  for item in "${discovered_branches[@]}"; do
    IFS='|' read -r name branch path url <<< "$item"
    echo "  - name: $name" >> "$discovered_file"
    echo "    branch: $branch" >> "$discovered_file"
    echo "    path: $path" >> "$discovered_file"
    echo "    url: $url" >> "$discovered_file"
  done
fi

if [ ! -s "ci.yml" ]; then
  echo
  echo "Info: generating 'ci.yml'..."
  tee ci.yml < "$discovered_file" 
fi

rm -f gitmodules.yml
