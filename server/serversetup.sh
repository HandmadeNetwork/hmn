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
    wget https://golang.org/dl/go1.17.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
    
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

# Set up SSH
if [ $checkpoint -lt 81 ]; then
    set +x

    do_as hmn <<'SCRIPT'
        ssh-keygen -t ed25519 -C "beta-server" -N "" -f ~/.ssh/gitlab
        git config --global core.sshCommand "ssh -i ~/.ssh/gitlab"
        echo ""
        echo ""
        echo "Copy the following key and add it as a Deploy Key in the project in GitLab (https://git.handmade.network/hmn/hmn/-/settings/ci_cd#js-deploy-keys-settings):"
        echo ""
        cat ~/.ssh/gitlab.pub
        echo ""
        echo "Run this script again when you're done - it will continue where it left off."
        exit 0
SCRIPT

    savecheckpoint 81

    # This is a special case, where we want to halt the script and allow the
    # user to perform an action before moving on.
    exit 0
fi

# Test SSH
if [ $checkpoint -lt 82 ]; then
    do_as hmn <<'SCRIPT'
        set -euxo pipefail
        
        if ! ssh -T -i ~/.ssh/gitlab git@gitssh.handmade.network; then
            set +x
            
            echo "Failed to connect to GitLab. Fix the issue and then run this script again."
            echo ""
            echo "Copy the following key and add it as a Deploy Key in the project in GitLab (https://git.handmade.network/hmn/hmn/-/settings/ci_cd#js-deploy-keys-settings):"
            echo ""
            cat ~/.ssh/gitlab.pub
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
        git clone git@gitssh.handmade.network:hmn/hmn.git
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
    cp /home/hmn/hmn/server/deploy.conf.example /home/hmn/hmn/server/deploy.conf
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
    
    savecheckpoint 100
fi

# Set up crons
if [ $checkpoint -lt 105 ]; then
  # See https://stackoverflow.com/a/9625233/1177139
  (crontab -l 2>/dev/null; echo "50 4 * * * /home/hmn/hmn/server/backup.sh") | crontab -

  savecheckpoint 105
fi

# Build the site for the first time (despite bad config)
if [ $checkpoint -lt 110 ]; then    
    do_as hmn <<'SCRIPT'
        set -euxo pipefail

        cd /home/hmn/hmn
        echo "Building the site for the first time. This may take a while..."
        go build -v -o /home/hmn/bin/hmn src/main.go
SCRIPT

    echo 'PATH=$PATH:/home/hmn/bin' >> ~/.bashrc
    source ~/.bashrc
    
    savecheckpoint 110
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
    the 'beta' branch.

${BLUE_BOLD}Website${RESET}: /home/hmn/hmn/src/config/config.go

    First make sure you have Go on your path:

        source ~/.bashrc

    Then edit the config file using a special make task:

        make edit-config

    Fill out everything, then rebuild the site:

        make build

    You don't need to deploy the site yet; wait until you've
    configured everything.

${BLUE_BOLD}HMN Environment Vars${RESET}: /home/hmn/hmn/server/hmn.conf

    First, go to GitLab and add a webhook with a secret. Set it to trigger on
    push events for the branch you are using for deploys.

        https://git.handmade.network/hmn/hmn/hooks

    Then, edit the above file and fill in all the environment vars, including
    the secret value from the GitLab webhook.

${BLUE_BOLD}Cinera${RESET}: /home/hmn/hmn/cinera/cinera.conf

    Add the correct domain.

${BLUE_BOLD}s3cmd${RESET}: /home/hmn/.s3cfg

    Add the DigitalOcean Spaces credentials, and ensure that the bucket info is correct.

${BLUE_BOLD}Admin mailer${RESET}: /home/hmn/hmn/adminmailer/config.go

    First make sure you have Go on your path:

        source ~/.bashrc

	Fill in the config file and build the mailer:

		cd /home/hmn/hmn/adminmailer
		go build .

${BLUE_BOLD}===== Next steps =====${RESET}

Make sure you have everything on your path:

    source ~/.bashrc

Download and restore a database backup:

    make download-database

    su hmn
    cd ~
    hmn migrate --list
    hmn seedfile <your backup file> <ID of initial migration>

Restore static files:

    make restore-static-files

Start up Caddy:

    systemctl start caddy

Then deploy the site:

    make deploy

Run 'make' on its own to see all the other tasks available to you!

HELP
