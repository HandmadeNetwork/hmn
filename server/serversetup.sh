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
# https://www.digitalocean.com/community/tutorials/how-to-add-swap-space-on-ubuntu-20-04
if [ $checkpoint -lt 10 ]; then
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
    
    savecheckpoint 10
fi

# Configure Linux users
if [ $checkpoint -lt 20 ]; then
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
	usermod -a -G annotations hmn
    
    savecheckpoint 20
fi

# Install important stuff
if [ $checkpoint -lt 30 ]; then
    apt update
    apt install -y \
        build-essential \
        libcurl4-openssl-dev byacc flex \
        s3cmd
    
    savecheckpoint 30
fi

# Install Go
if [ $checkpoint -lt 40 ]; then
	wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
    
    export PATH=$PATH:/usr/local/go/bin:/root/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin:/root/go/bin' >> ~/.bashrc
    go version

    do_as hmn <<'SCRIPT'
        set -euxo pipefail
        echo 'export PATH=$PATH:/usr/local/go/bin:/home/hmn/go/bin' >> ~/.bashrc
        go version
SCRIPT
    
    savecheckpoint 40
fi

export PATH=$PATH:/usr/local/go/bin:/root/go/bin

# Install Caddy
# https://www.digitalocean.com/community/tutorials/how-to-host-a-website-with-caddy-on-ubuntu-18-04
if [ $checkpoint -lt 50 ]; then
    go install github.com/caddyserver/xcaddy/cmd/xcaddy@v0.1.9
    xcaddy build \
        --with github.com/caddy-dns/cloudflare \
        --with github.com/aksdb/caddy-cgi/v2
    mv caddy /usr/bin
    chown root:root /usr/bin/caddy
    chmod 755 /usr/bin/caddy
    
    savecheckpoint 50
fi

# Install Postgres
# (instructions at https://www.postgresql.org/download/linux/ubuntu/)
if [ $checkpoint -lt 60 ]; then
    sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
    sudo apt-get update
    sudo apt-get -y install postgresql
    
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
        ssh-keygen -t ed25519 -C "beta-server" -N "" -f ~/.ssh/gitlab-hmn
        git config --global core.sshCommand "ssh -i ~/.ssh/gitlab-hmn"
SCRIPT

    do_as annotations <<'SCRIPT'
        ssh-keygen -t ed25519 -C "beta-server" -N "" -f ~/.ssh/gitlab-annotation-system
        ssh-keygen -t ed25519 -C "beta-server" -N "" -f ~/.ssh/gitlab-hmml
SCRIPT

    echo ""
    echo "Add the following keys as Deploy Keys to the following projects:"
    echo ""
    cat /home/hmn/.ssh/gitlab-hmn.pub
    echo "https://git.handmade.network/hmn/hmn/-/settings/ci_cd#js-deploy-keys-settings"
    echo ""
    cat /home/annotations/.ssh/gitlab-annotation-system.pub
    echo "https://git.handmade.network/Annotation-Pushers/Annotation-System/-/settings/ci_cd#js-deploy-keys-settings"
    echo ""
    cat /home/annotations/.ssh/gitlab-hmml.pub
    echo "https://git.handmade.network/Annotation-Pushers/cinera_handmade.network/-/settings/ci_cd#js-deploy-keys-settings"
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
        
        if ! ssh -T -i ~/.ssh/gitlab-hmn git@git.handmade.network; then
            set +x
            
            echo "Copy the following key:"
            echo ""
            cat ~/.ssh/gitlab-hmn
            echo ""
            echo "Add it as a Deploy Key to the HMN project in GitLab:"
            echo ""
            echo "    https://git.handmade.network/hmn/hmn/-/settings/ci_cd#js-deploy-keys-settings"
            echo ""
            exit 1
        fi
SCRIPT

    do_as annotations <<'SCRIPT'
        if ! ssh -T -i ~/.ssh/gitlab-annotation-system git@git.handmade.network; then
            set +x

            echo "Copy the following key:"
            echo ""
            cat ~/.ssh/gitlab-annotation-system
            echo ""
            echo "Add it as a Deploy Key to this project in GitLab:"
            echo ""
            echo "    https://git.handmade.network/Annotation-Pushers/Annotation-System/-/settings/ci_cd#js-deploy-keys-settings"
            echo ""
            exit 1
        fi

        if ! ssh -T -i ~/.ssh/gitlab-hmml git@git.handmade.network; then
            set +x

            echo "Copy the following key:"
            echo ""
            cat ~/.ssh/gitlab-hmml
            echo ""
            echo "Add it as a Deploy Key to this project in GitLab:"
            echo ""
            echo "    https://git.handmade.network/Annotation-Pushers/cinera_handmade.network/-/settings/ci_cd#js-deploy-keys-settings"
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
        git clone git@git.handmade.network:hmn/hmn.git
SCRIPT
    
    savecheckpoint 90
fi

# Copy config files to the right places
if [ $checkpoint -lt 100 ]; then
    cp /home/hmn/hmn/server/Caddyfile /home/caddy/Caddyfile

    cp /home/hmn/hmn/server/caddy.service /etc/systemd/system/caddy.service
    cp /home/hmn/hmn/server/hmn.service /etc/systemd/system/hmn.service
    cp /home/hmn/hmn/server/cinera.service /etc/systemd/system/cinera.service
    chmod 644 /etc/systemd/system/caddy.service
    chmod 644 /etc/systemd/system/hmn.service
    chmod 644 /etc/systemd/system/cinera.service

    cp /home/hmn/hmn/server/logrotate /etc/logrotate.d/hmn
    
    cp /home/hmn/hmn/src/config/config.go.example /home/hmn/hmn/src/config/config.go
    cp /home/hmn/hmn/server/hmn.conf.example /home/hmn/hmn/server/hmn.conf
    cp /home/hmn/hmn/adminmailer/config.go.example /home/hmn/hmn/adminmailer/config.go
    cp /home/hmn/hmn/cinera/cinera.conf.sample /home/hmn/hmn/cinera/cinera.conf
    chown hmn:hmn /home/hmn/hmn/src/config/config.go
    chown hmn:hmn /home/hmn/hmn/server/hmn.conf
    chown hmn:hmn /home/hmn/hmn/cinera/cinera.conf

    cp /home/hmn/hmn/server/.s3cfg /home/hmn/.s3cfg
    chown hmn:hmn /home/hmn/.s3cfg
    chmod 600 /home/hmn/.s3cfg

    cp /home/hmn/hmn/server/root.Makefile /root/Makefile

    systemctl daemon-reload
    systemctl enable caddy
    systemctl enable hmn
    systemctl enable cinera
    
    savecheckpoint 100
fi

# Set up crons
if [ $checkpoint -lt 105 ]; then
    # See https://stackoverflow.com/a/9625233/1177139
    (crontab -l 2>/dev/null; echo "50 4 * * * /home/hmn/hmn/server/backup.sh") | crontab -

    # TODO: This seems to fail the first time you run it? But then works fine afterward, thanks
    # to checkpoints. Probably should fix this someday.

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

# Install ffmpeg and cpulimit
if [ $checkpoint -lt 120 ]; then    
    apt update
    apt install -y \
        ffmpeg \
        cpulimit

    savecheckpoint 120
fi

cat <<HELP
Everything has been successfully installed!

${BLUE_BOLD}===== Next steps =====${RESET}

First, make sure you have everything on your path:

    source ~/.bashrc

${BLUE_BOLD}Edit the Caddy config${RESET}

Get an API token from Cloudflare. The token must have the Zone / Zone / Read and
Zone / DNS / Edit permissions (as laid out in the following links).

    https://github.com/caddy-dns/cloudflare
    https://github.com/libdns/cloudflare

Then edit the Caddyfile:

    vim /home/caddy/Caddyfile

Add the Cloudflare token to allow the ACME challenge to succeed, and add
the correct domains. (Don't forget to include both the normal and wildcard
domains.)

Also, in the CGI config, add the name of the Git branch you would like to
use when deploying. For example, a deployment of the beta site should use
the 'beta' branch.

${BLUE_BOLD}Edit the website config${RESET}

Edit the config file using a special make task:

    make edit-config

Fill out everything, then rebuild the site:

    make build

You don't need to deploy the site yet; wait until you've
configured everything.

${BLUE_BOLD}Edit HMN environment vars${RESET}

First, go to GitLab and add a webhook with a secret. Set it to trigger on
push events for the branch you are using for deploys.

    https://git.handmade.network/hmn/hmn/hooks

Then, edit the following file and fill in all the environment vars, including
the secret value from the GitLab webhook:

    /home/hmn/hmn/server/hmn.conf

${BLUE_BOLD}Configure s3cmd${RESET}

Edit the following file:

    /home/hmn/.s3cfg

Add the DigitalOcean Spaces credentials, and ensure that the bucket info is correct.

${BLUE_BOLD}Configure Cinera${RESET}

Edit the following file, adding the correct domain:

    /home/hmn/hmn/cinera/cinera.conf

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

${BLUE_BOLD}Set up Cinera${RESET}

    cd /home/hmn/hmn/cinera
    ./setup.sh

${BLUE_BOLD}Start up Caddy${RESET}

    systemctl start caddy

${BLUE_BOLD}Deploy the site!${RESET}

    make deploy

Run 'make' on its own to see all the other tasks available to you!

HELP
