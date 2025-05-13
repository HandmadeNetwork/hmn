package buildcss

import (
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/evanw/esbuild/pkg/api"
)

var ActiveServerPort uint16

func RunServer() *jobs.Job {
	job := jobs.New("esbuild CSS server")
	log := job.Logger

	if config.Config.Env != config.Dev {
		return job.Finish()
	}

	esCtx := utils.Must1(BuildContext())
	job.Logger.Info().Msg("Starting esbuild server and watcher")

	err := esCtx.Watch(api.WatchOptions{})
	serverResult, err := esCtx.Serve(api.ServeOptions{
		Port:     config.Config.EsBuild.Port,
		Servedir: "./",
		OnRequest: func(args api.ServeOnRequestArgs) {
			if args.Status != 200 {
				log.Warn().Interface("args", args).Msg("Response from esbuild server")
			}
		},
	})
	if err != nil {
		panic(err)
	}
	ActiveServerPort = serverResult.Port
	go func() {
		<-job.Canceled()
		log.Info().Msg("Shutting down esbuild server and watcher")
		esCtx.Dispose()
		job.Finish()
	}()

	return job
}

func BuildContext() (api.BuildContext, *api.ContextError) {
	return api.Context(api.BuildOptions{
		EntryPoints: []string{
			"src/rawdata/css/style.css",
			"src/rawdata/css/force-light.css",
			"src/rawdata/css/force-dark.css",
		},
		Outbase:  "src/rawdata/css",
		Outdir:   "public",
		External: []string{"/public/*"},
		Bundle:   true,
		Write:    true,
		Engines: []api.Engine{
			{Name: api.EngineChrome, Version: "109"},
			{Name: api.EngineFirefox, Version: "109"},
			{Name: api.EngineSafari, Version: "12"},
		},
	})
}
