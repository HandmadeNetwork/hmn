#!/bin/bash

set -euxo pipefail

BLACK_BOLD=$'\e[1;30m'
RESET=$'\e[0m'

# Add swap space
# https://www.digitalocean.com/community/tutorials/how-to-add-swap-space-on-ubuntu-20-04
fallocate -l 1G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
swapon --show
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
sysctl vm.swappiness=10
sysctl vm.vfs_cache_pressure=50
echo 'vm.swappiness=10' >> /etc/sysctl.conf
echo 'vm.vfs_cache_pressure=50' >> /etc/sysctl.conf

# Configure Linux users
groupadd --system caddy
useradd --system \
    --gid caddy \
    --create-home --home-dir /home/caddy \
    caddy
groupadd --system hmn
useradd --system \
    --gid hmn \
    --create-home --home-dir /home/hmn \
    hmn
groupadd --system annotations
useradd --system \
    --gid annotations \
    --create-home --home-dir /home/annotations \
    annotations

# Install important stuff
apt update
apt install -y \
    build-essential monit \
    libcurl4-openssl-dev byacc flex

# Install Go
wget https://golang.org/dl/go1.17.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
echo 'PATH=$PATH:/usr/local/go/bin:/root/go/bin' >> ~/.profile
source ~/.profile

# Install Caddy
# https://www.digitalocean.com/community/tutorials/how-to-host-a-website-with-caddy-on-ubuntu-18-04
# (with modifications)
go install github.com/caddyserver/xcaddy/cmd/xcaddy@v0.1.9
xcaddy build \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/aksdb/caddy-cgi/v2
mv caddy /usr/bin
chown root:root /usr/bin/caddy
chmod 755 /usr/bin/caddy

# Install Postgres
# (instructions at https://www.postgresql.org/download/linux/ubuntu/)
sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
sudo apt-get update
sudo apt-get -y install postgresql

# Configure Postgres
# TODO: This was supposed to create a user without a password - why didn't it?
# ...or was it?
sudo -u postgres createuser --createdb --login --pwprompt hmn

# Set up the folder structure, clone the repo
sudo -u hmn bash -s <<'SCRIPT'
set -euxo pipefail

cd ~
mkdir log
mkdir bin

echo 'PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile

ssh-keygen -t ed25519 -C "beta-server" -N "" -f ~/.ssh/gitlab
git config --global core.sshCommand "ssh -i ~/.ssh/gitlab"
echo ""
echo ""
echo "Copy the following key and add it as a Deploy Key in the project in GitLab (https://git.handmade.network/hmn/hmn/-/settings/ci_cd#js-deploy-keys-settings):"
echo ""
cat ~/.ssh/gitlab.pub
echo ""
echo "Press enter to continue when you're done."
read

git clone git@gitssh.handmade.network:hmn/hmn.git
pushd hmn
    echo "Building the site for the first time. This may take a while..."
    go build -o hmn src/main.go
popd
SCRIPT

# Copy config files to the right places
cp /home/hmn/hmn/server/Caddyfile /home/caddy/Caddyfile
cp /home/hmn/hmn/server/logrotate /etc/logrotate.d/hmn
cp /home/hmn/hmn/server/monitrc ~/.monitrc
cp /home/hmn/hmn/server/deploy.conf.example /home/hmn/hmn/server/deploy.conf
cp /home/hmn/hmn/src/config/config.go.example /home/hmn/hmn/src/config/config.go
cp /home/hmn/hmn/cinera/cinera.conf.sample /home/hmn/hmn/cinera/cinera.conf
chmod 600 ~/.monitrc

cat <<HELP
Everything has been installed, but before you can run the site, you will need to edit several config files:

${BLACK_BOLD}Caddy${RESET}: /home/caddy/Caddyfile

    Add the Cloudflare key to allow the ACME challenge to succeed, and add the correct domains. (Don't forget to include both the normal and wildcard domains.)

    Also, in the CGI config, add the name of the Git branch you would like to use when deploying.

${BLACK_BOLD}Monit${RESET}: ~/.monitrc

    Add the password for the email server.

${BLACK_BOLD}Deploy Secret${RESET}: /home/hmn/hmn/server/deploy.conf

    First, go to GitLab and add a webhook with a secret. Filter it down to just push events on the branch you care about.

    https://git.handmade.network/hmn/hmn/hooks

    Then, edit the above file and fill in the secret value from the GitLab webhook.

${BLACK_BOLD}Website${RESET}: /home/hmn/hmn/src/config/config.go

    Fill out everything :)

${BLACK_BOLD}Cinera${RESET}: /home/hmn/hmn/cinera/cinera.conf

    Add the correct domain.


${BLACK_BOLD}Next steps:${RESET}

Restore a database backup:

    pg_restore --single-transaction --dbname hmn --host localhost --username hmn ./path/to/dumpfile

Reload the monit config:

    monit reload

Start up Caddy:

    monit start caddy

Then run the deploy script:

    /home/hmn/hmn/server/deploy.sh

HELP
