#!/bin/bash

set -eu
trap "echo FAILED" ERR
sed -i 's^error_page 401 = /auth_request;^\0\nif ($request ~ (\\.\\.|%2e%2e)) { return 403; }^' /etc/nginx/conf.d/pmm.conf
grep -Fq 'request ~ (\.\.|%2e%2e)' /etc/nginx/conf.d/pmm.conf
nginx -t
nginx -s reload
