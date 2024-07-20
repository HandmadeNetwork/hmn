package buildcss

import (
	"context"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"github.com/evanw/esbuild/pkg/api"
)

var ActiveServerPort uint16

func RunServer(ctx context.Context) jobs.Job {
	job := jobs.New()
	if !config.Config.DevConfig.LiveTemplates {
		job.Done()
		return job
	}
	logger := logging.ExtractLogger(ctx).With().Str("module", "EsBuild").Logger()
	esCtx, ctxErr := BuildContext()
	if ctxErr != nil {
		panic(ctxErr)
	}
	logger.Info().Msg("Starting esbuild server and watcher")
	err := esCtx.Watch(api.WatchOptions{})
	serverResult, err := esCtx.Serve(api.ServeOptions{
		Port:     config.Config.EsBuild.Port,
		Servedir: "./",
		OnRequest: func(args api.ServeOnRequestArgs) {
			if args.Status != 200 {
				logger.Warn().Interface("args", args).Msg("Response from esbuild server")
			}
		},
	})
	if err != nil {
		panic(err)
	}
	ActiveServerPort = serverResult.Port
	go func() {
		<-ctx.Done()
		logger.Info().Msg("Shutting down esbuild server and watcher")
		esCtx.Dispose()
		job.Done()
	}()

	return job
}

func BuildContext() (api.BuildContext, *api.ContextError) {
	return api.Context(api.BuildOptions{
		EntryPoints: []string{
			"src/rawdata/css/style.css",
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
