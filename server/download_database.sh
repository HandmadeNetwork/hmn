#!/bin/bash

# Downloads a database backup from DigitalOcean. Does not restore it on its own; we have the
# seedfile command for that.

set -euo pipefail
source /home/hmn/hmn/server/hmn.conf

s3cmd --config /home/hmn/.s3cfg ls s3://hmn-backup/db/

echo ""
echo "Above is a list of all the available database backups."
echo "Enter the name of the one you would like to download (e.g. \"hmn_pg_dump_live_2021-09-01\"):"
read filename

s3cmd --config /home/hmn/.s3cfg get --force s3://hmn-backup/db/$filename $filename

echo ""
echo "Downloaded $filename to $(pwd)."
