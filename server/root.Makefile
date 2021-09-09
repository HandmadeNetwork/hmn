# from https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Print this help.
	@#                                                                  wrap text here -> |
	@#                                                                                    |
	@echo 'This Makefile contains tasks for the root user only. Most website management'
	@echo 'commands must be run as a different user account, e.g. hmn or caddy. To become'
	@echo 'another user, run "su <username>" and then "cd ~".'
	@echo ''

	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort \
		| sed 's/^.*\/\(.*\)/\1/' \
		| awk 'BEGIN {FS = ":[^:]*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deploy: ## Manually build and deploy a branch of the website.
	/home/hmn/hmn/server/deploy.sh

build: ## Rebuild the website binary
	sudo -u hmn --preserve-env=PATH bash -c "cd ~/hmn && go build -o /home/hmn/bin/hmn src/main.go"

edit-config: ## Edit the website config
	vim /home/hmn/hmn/src/config/config.go
	@echo 'Now that you have edited the config, you probably want to re-deploy the site:'
	@echo ''
	@echo '    make deploy'
	@echo ''

edit-caddyfile: ## Edit the Caddyfile
	vim /home/caddy/Caddyfile
	@echo 'Now that you have edited the Caddyfile, you probably want to restart Caddy:'
	@echo ''
	@echo '    systemctl restart caddy'
	@echo ''
	@echo "Don't forget to copy your changes back to the repo when you're done."

logs: ## View logs for the website
	journalctl -u hmn.service -n 100 -f

logs-caddy: ## View logs for Caddy
	journalctl -u caddy.service -n 100 -f

download-database: ## Download a database backup
	sudo -u hmn bash -c "cd ~ && ~/hmn/server/download_database.sh"

restore-static-files: ## Download static files from the backup.
	sudo -u hmn bash -c "cd ~/hmn && /home/hmn/hmn/server/restore_static_files.sh"

update-makefile: ## Update this Makefile with the latest from the repo.
	cp /home/hmn/hmn/server/root.Makefile /root/Makefile
