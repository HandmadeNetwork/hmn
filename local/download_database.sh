#!/bin/bash

set -euo pipefail

s3cmd ls s3://hmn-backup/db/

echo ""
echo "Above is a list of all the available database backups."
echo "Enter the name of the one you would like to download (e.g. \"hmn_pg_dump_live_2021-09-01\"):"
read filename

s3cmd get --force s3://hmn-backup/db/$filename ./local/backups/$filename

echo ""
echo "Downloaded $filename to local/backups."
