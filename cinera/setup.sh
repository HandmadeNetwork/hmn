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
chmod 775 $CINERA_OUTPUT_PATH
chmod 775 $CINERA_ASSETS_PATH
chmod 775 $CINERA_SCRIPT_PATH
chmod 775 $CINERA_SCRIPT_PATH/data

./update_cinera.sh

CMD="cd $CINERA_SCRIPT_PATH; ./update_annotations.sh"
su - $ANNOTATIONS_USER -c "$CMD"

mkdir -p data
