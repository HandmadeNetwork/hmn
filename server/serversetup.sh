#!/bin/bash

set -euxo pipefail

BLUE_BOLD=$'\e[1;34m'
RESET=$'\e[0m'

checkpoint=$(cat ./hmn_setup_checkpoint || echo 0)

savecheckpoint() {
    echo $1 > ./hmn_setup_checkpoint
}

# Add swap space
# https://www.digitalocean.com/community/tutorials/how-to-add-swap-space-on-ubuntu-20-04
if [ $checkpoint -lt 10 ]; then
    savecheckpoint 10

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
fi

# Configure Linux users
if [ $checkpoint -lt 20 ]; then
    savecheckpoint 20

    groupadd --system caddy
    useradd --system \
        --gid caddy \
        --shell /bin/bash \
        --create-home --home-dir /home/caddy \
        caddy
    groupadd --system hmn
    useradd --system \
        --gid hmn \
        --shell /bin/bash \
        --create-home --home-dir /home/hmn \
        hmn
    groupadd --system annotations
    useradd --system \
        --gid annotations \
        --shell /bin/bash \
        --create-home --home-dir /home/annotations \
        annotations
fi

# Install important stuff
if [ $checkpoint -lt 30 ]; then
    savecheckpoint 30

    apt update
    apt install -y \
        build-essential \
        libcurl4-openssl-dev byacc flex
fi

# Install Go
if [ $checkpoint -lt 40 ]; then
    savecheckpoint 40

    wget https://golang.org/dl/go1.17.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
    echo 'PATH=$PATH:/usr/local/go/bin:/root/go/bin' >> ~/.bash_profile
    source ~/.bash_profile
fi

# Install Caddy
# https://www.digitalocean.com/community/tutorials/how-to-host-a-website-with-caddy-on-ubuntu-18-04
if [ $checkpoint -lt 50 ]; then
    savecheckpoint 50

    go install github.com/caddyserver/xcaddy/cmd/xcaddy@v0.1.9
    xcaddy build \
        --with github.com/caddy-dns/cloudflare \
        --with github.com/aksdb/caddy-cgi/v2
    mv caddy /usr/bin
    chown root:root /usr/bin/caddy
    chmod 755 /usr/bin/caddy
fi

# Install Postgres
# (instructions at https://www.postgresql.org/download/linux/ubuntu/)
if [ $checkpoint -lt 60 ]; then
    savecheckpoint 60

    sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
    sudo apt-get update
    sudo apt-get -y install postgresql
fi

# Configure Postgres
# TODO: This was supposed to create a user without a password - why didn't it?
# ...or was it?
if [ $checkpoint -lt 70 ]; then
    savecheckpoint 70
    sudo -u postgres createuser --createdb --login --pwprompt hmn
fi

# Set up the folder structure, clone the repo
if [ $checkpoint -lt 80 ]; then
    savecheckpoint 80

    sudo -u hmn bash -s <<'SCRIPT'
        set -euxo pipefail

        cd ~
        mkdir log
        mkdir bin

        echo 'PATH=$PATH:/usr/local/go/bin:/home/hmn/bin' >> ~/.bash_profile
        source ~/.bash_profile

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

        while true ; do
            ssh -T git@gitssh.handmade.network && break || true
            echo "Failed to connect to GitLab. Fix the issue and then try again. (Press enter when you're done.)"
            read
        done
SCRIPT

    echo 'PATH=$PATH:/home/hmn/bin' >> ~/.bash_profile
    source ~/.bash_profile
fi

if [ $checkpoint -lt 90 ]; then
    savecheckpoint 90

    sudo -u hmn bash -s <<'SCRIPT'
        set -euxo pipefail

        cd ~
        git clone git@gitssh.handmade.network:hmn/hmn.git
SCRIPT
fi

# Copy config files to the right places
if [ $checkpoint -lt 100 ]; then
    savecheckpoint 100

    cp /home/hmn/hmn/server/Caddyfile /home/caddy/Caddyfile

    cp /home/hmn/hmn/server/caddy.service /etc/systemd/system/caddy.service
    cp /home/hmn/hmn/server/hmn.service /etc/systemd/system/hmn.service
    cp /home/hmn/hmn/server/cinera.service /etc/systemd/system/cinera.service
    chmod 644 /etc/systemd/system/caddy.service
    chmod 644 /etc/systemd/system/hmn.service
    chmod 644 /etc/systemd/system/cinera.service

    cp /home/hmn/hmn/server/logrotate /etc/logrotate.d/hmn
    
    cp /home/hmn/hmn/src/config/config.go.example /home/hmn/hmn/src/config/config.go
    cp /home/hmn/hmn/server/deploy.conf.example /home/hmn/hmn/server/deploy.conf
    cp /home/hmn/hmn/cinera/cinera.conf.sample /home/hmn/hmn/cinera/cinera.conf

    systemctl daemon-reload
fi

# Build the site for the first time (despite bad config)
if [ $checkpoint -lt 110 ]; then
    savecheckpoint 110
    
    sudo -u hmn bash -s <<'SCRIPT'
        set -euxo pipefail

        cd /home/hmn/hmn
        echo "Building the site for the first time. This may take a while..."
        go build -o /home/hmn/bin/hmn src/main.go
SCRIPT
fi

cat <<HELP
Everything has been installed, but before you can run the site, you will need
to edit several config files:

${BLUE_BOLD}Caddy${RESET}: /home/caddy/Caddyfile

    Get an API token from Cloudflare and add it to the Caddyfile to allow the
    ACME challenge to succeed. The token must have the Zone / Zone / Read and 
    Zone / DNS / Edit permissions (as laid out in the following links).

        https://github.com/caddy-dns/cloudflare
        https://github.com/libdns/cloudflare

    Add the Cloudflare token to allow the ACME challenge to succeed, and add
    the correct domains. (Don't forget to include both the normal and wildcard
    domains.)

    Also, in the CGI config, add the name of the Git branch you would like to
    use when deploying. For example, a deployment of the beta site should use
    the `beta` branch.

${BLUE_BOLD}Monit${RESET}: ~/.monitrc

    Add the password for the email server.

${BLUE_BOLD}Deploy Secret${RESET}: /home/hmn/hmn/server/deploy.conf

    First, go to GitLab and add a webhook with a secret. Set it to trigger on
    push events for the branch you are using for deploys.

        https://git.handmade.network/hmn/hmn/hooks

    Then, edit the above file and fill in the secret value from the
    GitLab webhook.

${BLUE_BOLD}Website${RESET}: /home/hmn/hmn/src/config/config.go

    Fill out everything :)

    Then rebuild the site:

        su hmn
        cd ~/hmn
        go build -o /home/hmn/bin/hmn src/main.go

${BLUE_BOLD}Cinera${RESET}: /home/hmn/hmn/cinera/cinera.conf

    Add the correct domain.


${BLUE_BOLD}===== Next steps =====${RESET}

Restore a database backup:

    su hmn
    cd ~
    /home/hmn/bin/hmn seedfile <I dunno man figure it out>

Start up Caddy:

    systemctl start caddy

Then run the deploy script:

    /home/hmn/hmn/server/deploy.sh

HELP
