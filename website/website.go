package website

import (
	"context"
	"fmt"
	"net/http"

	"git.handmade.network/hmn/hmn/config"
	"git.handmade.network/hmn/hmn/db"
	"git.handmade.network/hmn/hmn/logging"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

var WebsiteCommand = &cobra.Command{
	Short: "Run the HMN website",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					logging.Error().Err(err).Msg("recovered from panic")
				} else {
					logging.Error().Interface("recovered", r).Msg("recovered from panic")
				}
			}
		}()

		logging.Info().Msg("Hello, HMN!")

		conn := db.NewConnPool(4, 8)

		router := httprouter.New()
		router.GET("/", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
			rw.Write([]byte("Hello, HMN!"))
		})
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
		logging.Info().Str("addr", config.Config.Addr).Msg("Serving the website")
		http.ListenAndServe(config.Config.Addr, router)
	},
}
