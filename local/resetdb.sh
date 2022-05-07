#!/bin/bash

set -euxo pipefail

# This script is for use in local development only. It wipes the existing db,
# creates a new empty one, runs the initial migration to create the schema,
# and then imports actual db content on top of that.

# TODO(opensource): We should adapt Asaf's seedfile command and then delete this.

THIS_PATH=$(pwd)
BETA_PATH='/mnt/c/Users/bvisn/Developer/handmade/handmade-beta'
# BETA_PATH='/Users/benvisness/Developer/handmade/handmade-beta'

pushd $BETA_PATH
    docker-compose down -v
    docker-compose up -d postgres s3
    sleep 3

    docker-compose exec postgres bash -c "psql -U postgres -c \"CREATE ROLE hmn CREATEDB LOGIN PASSWORD 'password';\""
popd
go run src/main.go db seedfile local/backups/hmn_pg_dump_live_2021-11-14
# go run src/main.go db seedfile local/backups/hmn_pg_dump_live_2021-10-23
