#!/bin/bash
set -o errexit

git config --global --add safe.directory /app

rm -f gitmodules.yml

if [ ! -s ci.yml ]; then
  # Loop through all known repos, find the `BRANCH_NAME` and create a config file
  python ci.py --create-config ${BRANCH_NAME:-}
fi

# Join the dependencies listed in gitmodules.yml and ci.yml and output the result to gitmodules.yml
python ci.py --convert

DEPS=$(yq -o=json '.' gitmodules.yml | jq -r '[.deps[] | {name: .name, branch: .branch, path: .path, url: .url}]')
rm -f gitmodules.yml
echo "$DEPS" > /app/build/build.json
echo -n > /tmp/deps.txt

echo "$DEPS" | jq -c '.[]' | while read -r item; do
  branch=$(echo "$item" | jq -r '.branch')
  path=$(echo "$item" | jq -r '.path')
  name=$(echo "$item" | jq -r '.name')
  url=$(echo "$item" | jq -r '.url')
  echo "name=${name}|path=${path}|url=${url}|branch=${branch}" >> /tmp/deps.txt
done

mv -f /tmp/deps.txt /app/build/deps.txt
