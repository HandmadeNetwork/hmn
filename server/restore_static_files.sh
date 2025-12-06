#!/bin/bash

set -eo pipefail
source /home/hmn/hmn/server/hmn.conf

if [ "$(whoami)" != "hmn" ]; then
  echo "WARNING! You are not running this script as the hmn user. This will probably screw up file permissions."
  echo "Press Ctrl-C to cancel, or press enter to continue."
  read
fi

s3cmd sync s3://hmn-backup/static/media/ /home/hmn/hmn/public/media/
