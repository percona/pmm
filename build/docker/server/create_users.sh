#!/bin/bash

users=(
  "pmm:1000:/bin/false:/home/pmm:pmm"
  "nginx:1001:/sbin/nologin:/var/cache/nginx:nginx"
)

for user in "${users[@]}"; do
  IFS=: read -r name uid shell home_dir group <<< "$user"
  group_id="$uid"

  # Check if user already exists
  if id "$name" >/dev/null 2>&1; then
    echo "User $name already exists"
    continue
  fi

  # Create user with home directory if it doesn't exist
  if [ ! -d "$home_dir" ]; then
    mkdir -p "$home_dir"
  fi

  # Create user with specified UID, GID, and shell
  groupadd -o -g "$group_id" "$group"
  useradd -o -u "$uid" -g "$group" -G "$group" -s "$shell" -d "$home_dir" -c "$name" -m "$name"
  chown "$uid:$group_id" "$home_dir"

done

