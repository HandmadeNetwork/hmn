#!/bin/bash

# This script should be called with the name
# of the branch to deploy. ($1 will be the
# branch name.)

set -euo pipefail

sudo -u hmn bash -s <<SCRIPT
set -euo pipefail
pushd /home/hmn/hmn
    git fetch --all
    git reset --hard $1
    go build -o hmn src/main.go
popd
SCRIPT

monit stop hmn
sudo -u hmn bash -s <<'SCRIPT'
set -euo pipefail
/home/hmn/hmn/hmn migrate
SCRIPT
monit start hmn
