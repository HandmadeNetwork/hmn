package website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

var WebsiteCommand = &cobra.Command{
	Short: "Run the HMN website",
	Run: func(cmd *cobra.Command, args []string) {
		defer logging.LogPanics()

		logging.Info().Msg("Hello, HMN!")

		conn := db.NewConnPool(4, 8)

		router := httprouter.New()
		router.GET("/", WithRequestLogger(func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
			rw.Write([]byte(fmt.Sprintf("Hello, HMN! The time is: %v\n", time.Now())))
		}))
		router.GET("/project/:id", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
			id := p.ByName("id")
			row := conn.QueryRow(context.Background(), "SELECT name FROM handmade_project WHERE id = $1", p.ByName("id"))
			var name string
			err := row.Scan(&name)
			if err != nil {
				panic(err)
			}

			rw.Write([]byte(fmt.Sprintf("(%s) %s\n", id, name)))
		})

		server := http.Server{
			Addr:    config.Config.Addr,
			Handler: router,
		}

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		go func() {
			<-signals
			logging.Info().Msg("Shutting down the website")
			timeout, _ := context.WithTimeout(context.Background(), 30*time.Second)
			server.Shutdown(timeout)

			<-signals
			logging.Warn().Msg("Forcibly killed the website")
			os.Exit(1)
		}()

		logging.Info().Str("addr", config.Config.Addr).Msg("Serving the website")
		serverErr := server.ListenAndServe()
		if !errors.Is(serverErr, http.ErrServerClosed) {
			logging.Error().Err(serverErr).Msg("Server shut down unexpectedly")
		}
	},
}

func WithRequestLogger(h httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		defer logging.LogPanics()

		start := time.Now()
		defer func() {
			end := time.Now()
			logging.Info().Dur("duration", end.Sub(start)).Msg("Completed request")
		}()

		h(rw, r, p)
	}
}
