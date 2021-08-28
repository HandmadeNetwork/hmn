#!/bin/bash

if [ ! -e "cinera.conf" ]; then
    echo "Can't find cinera.conf"
    exit
fi
. cinera.conf

./update_cinera.sh
./update_annotations.sh

[ -d "data" ] || mkdir data
