#!/bin/bash

if [ ! -e "cinera.conf" ]; then
    echo "Can't find cinera.conf"
    exit
fi
. cinera.conf

systemctl stop cinera.service

CMD="cd $CINERA_SCRIPT_PATH; ./user_update_cinera.sh"

su - $ANNOTATIONS_USER -c "$CMD"

systemctl start cinera.service
