package website

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"git.handmade.network/hmn/hmn/db"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

var WebsiteCommand = &cobra.Command{
	Short: "Run the HMN website",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, HMN!")

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
		log.Print("serving stuff")
		http.ListenAndServe(":9001", router)
	},
}
