#!/bin/bash

cd "$(dirname "$0")"

[ -e "./cinera.conf" ] || exit
. ./cinera.conf

$CINERA_REPO_PATH/cinera/cinera -c cinera_hmn.conf

