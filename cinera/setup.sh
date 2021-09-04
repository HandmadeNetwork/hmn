#!/bin/bash

if [ ! -e "cinera.conf" ]; then
    echo "Can't find cinera.conf"
    exit
fi
. cinera.conf

mkdir -p $CINERA_OUTPUT_PATH
mkdir -p $CINERA_ASSETS_PATH
mkdir -p $CINERA_SCRIPT_PATH/data

chgrp $ANNOTATIONS_USER_GROUP $CINERA_OUTPUT_PATH
chgrp $ANNOTATIONS_USER_GROUP $CINERA_ASSETS_PATH
chgrp $ANNOTATIONS_USER_GROUP $CINERA_SCRIPT_PATH
chgrp $ANNOTATIONS_USER_GROUP $CINERA_SCRIPT_PATH/data

./update_cinera.sh
./update_annotations.sh

mkdir -p data
