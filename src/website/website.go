package website

import (
	"context"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
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
		templates.Init()

		defer logging.LogPanics(nil)

		logging.Info().Msg("Hello, HMN!")

		backgroundJobContext, cancelBackgroundJobs := context.WithCancel(context.Background())
		longRequestContext, cancelLongRequests := context.WithCancel(context.Background())

		conn := db.NewConnPool()
		perfCollector := perf.RunPerfCollector(backgroundJobContext)

		server := http.Server{
			Addr:    config.Config.Addr,
			Handler: NewWebsiteRoutes(longRequestContext, conn),
		}

		backgroundJobsDone := jobs.Zip(
			auth.PeriodicallyDeleteExpiredSessions(backgroundJobContext, conn),
			auth.PeriodicallyDeleteInactiveUsers(backgroundJobContext, conn),
			perfCollector.Job,
			discord.RunDiscordBot(backgroundJobContext, conn),
			discord.RunHistoryWatcher(backgroundJobContext, conn),
			twitch.MonitorTwitchSubscriptions(backgroundJobContext, conn),
			hmns3.StartServer(backgroundJobContext),
		)

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		go func() {
			<-signals
			logging.Info().Msg("Shutting down the website")
			go func() {
				logging.Info().Msg("cancelling long requests")
				cancelLongRequests()
				timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				logging.Info().Msg("shutting down web server")
				server.Shutdown(timeout)
				logging.Info().Msg("cancelling background jobs")
				cancelBackgroundJobs()
			}()

			<-signals
			logging.Warn().Msg("Forcibly killed the website")
			os.Exit(1)
		}()

		go func() {
			log.Println(http.ListenAndServe(config.Config.PrivateAddr, nil))
		}()

		logging.Info().Str("addr", config.Config.Addr).Msg("Serving the website")
		serverErr := server.ListenAndServe()
		if !errors.Is(serverErr, http.ErrServerClosed) {
			logging.Error().Err(serverErr).Msg("Server shut down unexpectedly")
		}

		<-backgroundJobsDone.C
	},
}
