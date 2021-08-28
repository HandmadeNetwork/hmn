# How to run Cinera locally

0. Install prerequisites:
	a. `libcurl4-openssl-dev`
	b. `byacc`
	c. `flex`

1. Copy `cinera/cinera.conf.sample` to `cinera/cinera.conf` and edit to match your local system
2. Run `cinera/user_update_cinera.sh`
3. Run `cinera/update_annotations.sh`
4. From the cinera dir run: `./run_local.sh`
5. Once it's done processing everything (you should see "Monitoring
   file system for new, edited and deleted .hmml and asset files")
   You can shut it down with ctrl+c.

