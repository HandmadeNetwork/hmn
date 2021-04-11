#!/bin/bash

THIS_PATH=$(pwd)
BETA_PATH='/mnt/c/Users/bvisn/Developer/handmade/handmade-beta'

cd $BETA_PATH
docker-compose down -v
docker-compose up -d postgres
sleep 3
./scripts/db_import -d -n hmn_two -c

cd $THIS_PATH
go run src/main.go migrate 2021-03-10T05:16:21Z

cd $BETA_PATH
./scripts/db_import -d -n hmn_two -a ./dbdumps/hmn_pg_dump_2020-11-10
