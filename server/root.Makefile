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

deploy:
	/home/hmn/hmn/server/deploy.sh $1

edit-config:
	vim /home/hmn/hmn/src/config/config.go
	@echo 'Now that you have edited the config, you probably want to re-deploy the site:'
	@echo ''
	@echo '    make deploy'
	@echo ''

logs-hmn: ## View logs for the website
	journalctl -u hmn.service -f

logs-caddy: ## View logs for Caddy
	journalctl -u caddy.service -f
