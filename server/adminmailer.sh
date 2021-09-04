#!/bin/bash

/home/hmn/hmn/adminmailer/adminmailer "[$1] Status changed" <<ERRMAIL
$(service "$1" status)
ERRMAIL
