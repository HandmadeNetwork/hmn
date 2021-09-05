#!/bin/bash

set -eo pipefail
source /home/hmn/hmn/server/hmn.conf

branch=$1
if [ -z "$branch" ]; then
    echo "Type the name of the branch you would like to deploy: "
    read branch
fi

echo "Deploying branch $branch..."

do_as() {
    sudo -u $1 --preserve-env=PATH bash -s
}

cd /home/hmn/hmn
old_rev=$(git log -1 --pretty=format:%H)

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

cd /home/hmn/hmn
new_rev=$(git log -1 --pretty=format:%H)
if [ $old_rev != $new_rev ]; then
	adminmailer "[$HMN_ENV] Deployed new version" <<DEPLOYMAIL
$(git --no-pager log --no-color $old_rev..)
DEPLOYMAIL
fi


