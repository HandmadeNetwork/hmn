#!/bin/bash

set -eo pipefail

branch=$1
if [ -z "$branch" ]; then
    echo "Type the name of the branch you would like to deploy: "
    read branch
fi

echo "Deploying branch $branch..."

do_as() {
    sudo -u $1 --preserve-env=PATH bash -s
}

do_as hmn <<SCRIPT
set -euo pipefail
cd /home/hmn/hmn
git fetch --all
git reset --hard origin/$branch
go build -o /home/hmn/bin/hmn src/main.go
SCRIPT

echo "Running migrations..."
systemctl stop hmn
do_as hmn <<'SCRIPT'
set -euo pipefail
/home/hmn/bin/hmn migrate
SCRIPT
systemctl start hmn
