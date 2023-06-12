#!/bin/bash

set -eu
trap "echo FAILED" ERR
sed -i 's^error_page 401 = /auth_request;^\0\nif ($request ~ (\\.\\.|%2[eE]%2[eE])) { return 403; }^' /etc/nginx/conf.d/pmm.conf
grep -Fq 'request ~ (\.\.|%2[eE]%2[eE])' /etc/nginx/conf.d/pmm.conf
nginx -t
nginx -s reload
