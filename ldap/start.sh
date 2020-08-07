#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# restart the database
! [ -z `docker-compose -f $DIR/docker-compose.yml ps -q ldap` ] || [ -z `docker ps -q --no-trunc | grep $(docker-compose -f $DIR/docker-compose.yml ps -q ldap)` ] && docker-compose -f $DIR/docker-compose.yml down
docker-compose -f $DIR/docker-compose.yml up -d

# touch lock file
touch /tmp/ssodav_ldap.lock
