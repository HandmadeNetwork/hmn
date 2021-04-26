#!/bin/bash

set -eou pipefail

# This script is for use in local development only. It wipes the existing db,
# creates a new empty one, runs the initial migration to create the schema,
# and then imports actual db content on top of that.

# TODO(opensource): We should adapt Asaf's seedfile command and then delete this.

THIS_PATH=$(pwd)
#BETA_PATH='/mnt/c/Users/bvisn/Developer/handmade/handmade-beta'
BETA_PATH='/Users/benvisness/Developer/handmade/handmade-beta'

cd $BETA_PATH
docker-compose down -v
docker-compose up -d postgres
sleep 3
./scripts/db_import -d -n hmn_two -c

cd $THIS_PATH
go run src/main.go migrate 2021-03-10T05:16:21Z

cd $BETA_PATH
#./scripts/db_import -d -n hmn_two -a ./dbdumps/hmn_pg_dump_2020-11-10
./scripts/db_import -d -n hmn_two -a ./dbdumps/hmn_pg_dump_2021-04-25
