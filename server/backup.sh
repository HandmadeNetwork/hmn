#!/bin/bash

set -euxo pipefail
source /home/hmn/hmn/server/hmn.conf

TS=$(date --iso-8601)
FILENAME="hmn_pg_dump_${HMN_ENV}_${TS}"
DUMP="/tmp/$FILENAME"

echo "Dumping database..."
su - postgres -c "pg_dump -Fc hmn > $DUMP"

echo "Uploading database..."
s3cmd --config /home/hmn/.s3cfg put $DUMP s3://hmn-backup/db/$FILENAME

echo "Uploading static assets..."
s3cmd --config /home/hmn/.s3cfg sync /home/hmn/hmn/public/media/ s3://hmn-backup/static/media/

echo "Done."
rm "$DUMP"
