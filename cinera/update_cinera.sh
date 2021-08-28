#!/bin/bash

if [ ! -e "cinera.conf" ]; then
    echo "Can't find cinera.conf"
    exit
fi
. cinera.conf

monit -g $CINERA_MONIT_GROUP stop

CMD="cd $CINERA_SCRIPT_PATH; ./user_update_cinera.sh"

su - $ANNOTATIONS_USER -c "$CMD"

monit -g $CINERA_MONIT_GROUP start
