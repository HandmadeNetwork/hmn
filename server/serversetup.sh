#!/bin/bash

set -exo pipefail

BLUE_BOLD=$'\e[1;34m'
RESET=$'\e[0m'

checkpoint=$(cat ./hmn_setup_checkpoint || echo 0)

savecheckpoint() {
    echo $1 > ./hmn_setup_checkpoint
}

do_as() {
    sudo -u $1 --preserve-env=PATH bash -s
}

# Add swap space
# https://www.digitalocean.com/community/tutorials/how-to-add-swap-space-on-ubuntu-22-04
if [ $checkpoint -lt 10 ]; then
    fallocate -l 1G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    swapon --show
    echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
    sysctl vm.swappiness=10
    sysctl vm.vfs_cache_pressure=50
    echo 'vm.swappiness=40' >> /etc/sysctl.conf
    echo 'vm.vfs_cache_pressure=50' >> /etc/sysctl.conf
    
    savecheckpoint 10
fi

# Configure Linux users
if [ $checkpoint -lt 20 ]; then
    groupadd --system caddy
    useradd --system \
        --gid caddy \
        --shell /usr/sbin/nologin \
        --create-home --home-dir /home/caddy \
        caddy
    groupadd --system hmn
    useradd --system \
        --gid hmn \
        --shell /bin/bash \
        --create-home --home-dir /home/hmn \
        hmn
    
    savecheckpoint 20
fi

# Install important stuff
if [ $checkpoint -lt 30 ]; then
    apt update
    apt install -y \
        build-essential \
        s3cmd \
        ffmpeg \
        cpulimit
    
    savecheckpoint 30
fi

# Install Go
if [ $checkpoint -lt 40 ]; then
	wget https://go.dev/dl/go1.25.5.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.25.5.linux-amd64.tar.gz
    
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    go version

    do_as hmn <<'SCRIPT'
        set -euxo pipefail
        cd ~
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        go version
SCRIPT
    
    savecheckpoint 40
fi

export PATH=$PATH:/usr/local/go/bin

# Install Caddy
# https://caddyserver.com/docs/install#debian-ubuntu-raspbian
if [ $checkpoint -lt 50 ]; then
    apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
    chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    chmod o+r /etc/apt/sources.list.d/caddy-stable.list
    apt update
    apt install -y caddy

    caddy --version

    # This is currently marked "experimental", so it may eventually stop
    # working. If so, I will be sad. But for now it works and is much easier
    # and faster than xcaddy, and allows us to use the default apt approach for
    # installing caddy.
    caddy add-package github.com/caddy-dns/cloudflare
    caddy list-modules --packages --skip-standard
    
    savecheckpoint 50
fi

# Install Postgres
# (instructions at https://www.postgresql.org/download/linux/ubuntu/)
if [ $checkpoint -lt 60 ]; then
    apt install -y postgresql-common
    /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y

    apt update
    apt -y install postgresql
    
    savecheckpoint 60
fi

# Configure Postgres
if [ $checkpoint -lt 70 ]; then
    echo "Enter the password for the HMN postgres user:"
    sudo -u postgres createuser --createdb --login --pwprompt hmn
    
    savecheckpoint 70
fi

# Set up the folder structure
if [ $checkpoint -lt 80 ]; then
    set +x

    do_as hmn <<'SCRIPT'
        cd ~
        mkdir log
        mkdir bin

        echo 'export PATH=$PATH:/home/hmn/bin' >> ~/.bashrc
SCRIPT

    savecheckpoint 80
fi

# Set up SSH for hmn
if [ $checkpoint -lt 81 ]; then
    set +x

    do_as hmn <<'SCRIPT'
        ssh-keygen -t ed25519 -C "hmn-server" -N "" -f ~/.ssh/github-hmn
        git config --global core.sshCommand "ssh -i ~/.ssh/github-hmn"
SCRIPT

    echo ""
    echo "Add the following key as a deploy key to the GitHub repo:"
    echo ""
    cat /home/hmn/.ssh/github-hmn.pub
    echo "https://github.com/HandmadeNetwork/hmn/settings/keys/new"
    echo ""
    echo "Run this script again when you're done - it will continue where it left off."

    savecheckpoint 81

    # This is a special case, where we want to halt the script and allow the
    # user to perform an action before moving on.
    exit 0
fi

# Test SSH for hmn
if [ $checkpoint -lt 82 ]; then
    do_as hmn <<'SCRIPT'
        set -euxo pipefail

        output=$(ssh -T -i ~/.ssh/github-hmn git@github.com 2>&1 || true)
        
        if ! echo "$output" | grep -q "successfully authenticated"; then
            set +x
            
            echo "Copy the following key:"
            echo ""
            cat ~/.ssh/github-hmn.pub
            echo ""
            echo "Add it as a deploy key in the GitHub repo:"
            echo ""
            echo "    https://github.com/HandmadeNetwork/hmn/settings/keys/new"
            echo ""
            exit 1
        fi
SCRIPT
    
    savecheckpoint 82
fi

# Clone the repo
if [ $checkpoint -lt 90 ]; then
    do_as hmn <<'SCRIPT'
        set -euxo pipefail

        cd ~
        git clone git@github.com:HandmadeNetwork/hmn.git
SCRIPT
    
    savecheckpoint 90
fi

# Copy config files to the right places
if [ $checkpoint -lt 100 ]; then
    cp /home/hmn/hmn/server/Caddyfile /etc/caddy/Caddyfile

    mkdir -p /etc/systemd/system/caddy.service.d
    cp /home/hmn/hmn/server/caddy.service.override /etc/systemd/system/caddy.service.d/override.conf
    cp /home/hmn/hmn/server/hmn.service /etc/systemd/system/hmn.service
    chmod 644 /etc/systemd/system/caddy.service.d/override.conf
    chmod 644 /etc/systemd/system/hmn.service

    cp /home/hmn/hmn/server/logrotate /etc/logrotate.d/hmn
    
    cp /home/hmn/hmn/src/config/config.go.example /home/hmn/hmn/src/config/config.go
    cp /home/hmn/hmn/server/hmn.conf.example /home/hmn/hmn/server/hmn.conf
    cp /home/hmn/hmn/adminmailer/config.go.example /home/hmn/hmn/adminmailer/config.go
    chown hmn:hmn /home/hmn/hmn/src/config/config.go
    chown hmn:hmn /home/hmn/hmn/server/hmn.conf

    cp /home/hmn/hmn/server/.s3cfg /home/hmn/.s3cfg
    chown hmn:hmn /home/hmn/.s3cfg
    chmod 600 /home/hmn/.s3cfg

    cp /home/hmn/hmn/server/root.Makefile /root/Makefile

    systemctl daemon-reload
    systemctl enable caddy
    systemctl enable hmn
    
    savecheckpoint 100
fi

# Set up crons
if [ $checkpoint -lt 105 ]; then
    # See https://stackoverflow.com/a/9625233/1177139
    (crontab -l 2>/dev/null || true; echo "50 4 * * * /home/hmn/hmn/server/backup.sh") | crontab -

    savecheckpoint 105
fi

# Build the site for the first time (despite bad config)
if [ $checkpoint -lt 110 ]; then    
    do_as hmn <<'SCRIPT'
        set -euxo pipefail

        cd /home/hmn/hmn
        echo "Building the site for the first time. This may take a while..."
        go build -v -o /home/hmn/bin/hmn .
SCRIPT

    echo 'PATH=$PATH:/home/hmn/bin' >> ~/.bashrc
    source ~/.bashrc
    
    savecheckpoint 110
fi

cat <<HELP
Everything has been successfully installed!

${BLUE_BOLD}===== Next steps =====${RESET}

First, make sure you have everything on your path:

    source ~/.bashrc

${BLUE_BOLD}Edit the Caddy config${RESET}

Get an API token from Cloudflare. The token must have the Zone / Zone / Read
and Zone / DNS / Edit permissions (as laid out in the following links).

    https://github.com/caddy-dns/cloudflare
    https://github.com/libdns/cloudflare

Then edit the Caddyfile:

    vim /etc/caddy/Caddyfile

Add the Cloudflare token to allow the ACME challenge to succeed, and add
the correct domains. (Don't forget to include both the normal and wildcard
domains.)

${BLUE_BOLD}Edit the website config${RESET}

Edit the config file using a special make task:

    make edit-config

Fill out everything, then rebuild the site:

    make build

You don't need to deploy the site yet; wait until you've
configured everything.

${BLUE_BOLD}Edit HMN environment vars${RESET}

Edit the following file and fill in all the environment vars:

    /home/hmn/hmn/server/hmn.conf

${BLUE_BOLD}Configure s3cmd${RESET}

Edit the following file:

    /home/hmn/.s3cfg

Add the DigitalOcean Spaces credentials, and ensure that the bucket info is
correct.

${BLUE_BOLD}Configure the admin mailer${RESET}

Fill in the config file and build the mailer:

    cd /home/hmn/hmn/adminmailer
    vim config.go
    go build -o /usr/bin/adminmailer .

${BLUE_BOLD}Download and restore a database backup${RESET}

    make download-database

    su hmn
    cd ~
    hmn db seedfile <your backup file>
    hmn db migrate

${BLUE_BOLD}Restore static files${RESET}

    make restore-static-files

${BLUE_BOLD}Restart Caddy${RESET}

    systemctl restart caddy

${BLUE_BOLD}Deploy the site!${RESET}

    make deploy

Run 'make' on its own to see all the other tasks available to you!

HELP
