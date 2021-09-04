#!/bin/bash

if [ $SERVICE_RESULT == "success" ]; then
	exit
fi

adminmailer "[$1] Status changed" <<ERRMAIL
$(systemctl status --full "$1")
ERRMAIL
