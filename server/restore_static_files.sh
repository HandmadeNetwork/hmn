#!/bin/bash

set -eo pipefail
source /home/hmn/hmn/server/hmn.conf

if [ "$(whoami)" != "hmn" ]; then
  echo "WARNING! You are not running this script as the hmn user. This will probably screw up file permissions."
  echo "Press Ctrl-C to cancel, or press enter to continue."
  read
fi

# The --no-preserve flag prevents it from attempting to restore user/group, which we don't care
# about as these all should be owned by hmn/hmn anyway.
s3cmd sync --no-preserve s3://hmn-backup/static/media/ /home/hmn/hmn/public/media/
