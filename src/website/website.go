package website

import (
	"context"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/buildcss"
	"git.handmade.network/hmn/hmn/src/calendar"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmns3"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/twitch"
	"github.com/spf13/cobra"
)

var WebsiteCommand = &cobra.Command{
	Short: "Run the HMN website",
	Run: func(cmd *cobra.Command, args []string) {
		defer logging.LogPanics(nil)
		logging.Info().Msg("Hello, HMN!")

		templates.Init()

		var wg sync.WaitGroup

		conn := db.NewConnPool()
		perfCollector, perfCollectorJob := perf.RunPerfCollector()

		// Start background jobs
		wg.Add(1)
		backgroundJobs := jobs.Jobs{
			auth.PeriodicallyDeleteExpiredStuff(conn),
			auth.PeriodicallyDeleteInactiveUsers(conn),
			perfCollectorJob,
			discord.RunDiscordBot(conn),
			discord.RunHistoryWatcher(conn),
			twitch.MonitorTwitchSubscriptions(conn),
			hmns3.StartServer(),
			assets.BackgroundPreviewGeneration(conn),
			calendar.MonitorCalendars(),
			buildcss.RunServer(),
			email.MonitorBounces(conn),
		}

		// Create HTTP server
		wg.Add(1)
		server := http.Server{
			Addr:    config.Config.Addr,
			Handler: NewWebsiteRoutes(conn, perfCollector),
		}
		go func() {
			logging.Info().Str("addr", config.Config.Addr).Msg("Serving the website")
			serverErr := server.ListenAndServe()
			if !errors.Is(serverErr, http.ErrServerClosed) {
				logging.Error().Err(serverErr).Msg("Server shut down unexpectedly")
			}
			// The wg.Done() happens in the shutdown logic below.
		}()

		// Start up the private HTTP server for pprof. Because it uses the default
		// mux, and we import pprof, it will automatically have all the routes.
		go func() {
			// We don't bother to gracefully shut this down.
			log.Println(http.ListenAndServe(config.Config.PrivateAddr, nil))
		}()

		// Wait for SIGINT in the background and trigger graceful shutdown
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		go func() {
			<-signals // First SIGINT (start shutdown)
			logging.Info().Msg("Shutting down the website")

			const timeout = 10 * time.Second

			go func() {
				logging.Info().Msg("Shutting down background jobs...")
				unfinished := backgroundJobs.CancelAndWait(10 * time.Second)
				if len(unfinished) == 0 {
					logging.Info().Msg("Background jobs closed gracefully")
				} else {
					logging.Warn().Strs("Unfinished", unfinished).Msg("Background jobs did not finish by the deadline")
				}
				wg.Done()
			}()

			// Gracefully shut down the HTTP server
			go func() {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				err := server.Shutdown(timeoutCtx)
				if err != nil {
					logging.Warn().Err(err).Msg("Server did not shut down gracefully")
				}
				wg.Done()
			}()

			<-signals // Second SIGINT (force quit)
			logging.Warn().Strs("Unfinished background jobs", backgroundJobs.ListUnfinished()).Msg("Forcibly killed the website")
			os.Exit(1)
		}()

		// Wait for all of the above to finish, then exit
		wg.Wait()
	},
}
