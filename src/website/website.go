package website

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/spf13/cobra"
)

var WebsiteCommand = &cobra.Command{
	Short: "Run the HMN website",
	Run: func(cmd *cobra.Command, args []string) {
		templates.Init()

		defer logging.LogPanics(nil)

		logging.Info().Msg("Hello, HMN!")

		conn := db.NewConnPool(4, 128)

		server := http.Server{
			Addr:    config.Config.Addr,
			Handler: NewWebsiteRoutes(conn),
		}

		backgroundJobContext, cancelBackgroundJobs := context.WithCancel(context.Background())
		backgroundJobsDone := zipJobs(
			auth.PeriodicallyDeleteExpiredSessions(backgroundJobContext, conn),
		)

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		go func() {
			<-signals
			logging.Info().Msg("Shutting down the website")
			go func() {
				timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				server.Shutdown(timeout)
				cancelBackgroundJobs()
			}()

			<-signals
			logging.Warn().Msg("Forcibly killed the website")
			os.Exit(1)
		}()

		logging.Info().Str("addr", config.Config.Addr).Msg("Serving the website")
		serverErr := server.ListenAndServe()
		if !errors.Is(serverErr, http.ErrServerClosed) {
			logging.Error().Err(serverErr).Msg("Server shut down unexpectedly")
		}

		<-backgroundJobsDone
	},
}

func zipJobs(cs ...<-chan struct{}) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		for _, c := range cs {
			<-c
		}
		close(out)
	}()
	return out
}
