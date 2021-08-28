#!/bin/bash

cd "$(dirname "$0")"

[ -e "./cinera.conf" ] || exit
. ./cinera.conf

nohup $CINERA_REPO_PATH/cinera/cinera -c cinera_hmn.conf > data/cinera.log 2>&1 &
echo $! > data/cinera.pid

