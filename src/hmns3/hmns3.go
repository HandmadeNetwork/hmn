package hmns3

import (
	_ "embed"
	"fmt"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

func init() {
	s3Command := &cobra.Command{
		Use:   "hmns3 [storage folder]",
		Short: "Run a local s3 server that stores in the filesystem",
		Run: func(cmd *cobra.Command, args []string) {
			targetFolder := "./tmp"
			if len(args) > 0 {
				targetFolder = args[0]
			}
			err := os.MkdirAll(targetFolder, fs.ModePerm)
			if err != nil {
				panic(err)
			}

			handler := func(w http.ResponseWriter, r *http.Request) {
				bucket, key := bucket_key(r)
			
				fmt.Println("\n\nIncoming request path:", r.URL.Path)
				bodyBytes, err := io.ReadAll(r.Body)
				fmt.Println("Bucket: ", bucket, " key: ", key, " method: ", r.Method, " len(body): ", len(bodyBytes))
				if err != nil {
					panic(err)
				}
				if r.Method == http.MethodPut {
					w.Header().Set("Location", fmt.Sprintf("/%s", bucket))
					err := os.MkdirAll(fmt.Sprintf("%s/%s", targetFolder, bucket), fs.ModePerm)
					if err != nil {
						panic(err)
					}
					if key != "" {
						err = os.WriteFile(fmt.Sprintf("%s/%s/%s",targetFolder, bucket, key),   bodyBytes, fs.ModePerm)
						if err != nil {
							panic(err)
						}
					}
				} else if r.Method == http.MethodGet {
					fileBytes, err := os.ReadFile(fmt.Sprintf("%s/%s/%s",  targetFolder, bucket, key))
					if err != nil {
						panic(err)
					}
					w.Write(fileBytes)
				} else {
					panic("Unimplemented method!")
				}
			}

			http.HandleFunc("/", handler)
			log.Fatal(http.ListenAndServe(":80", nil))
		},
	}

	website.WebsiteCommand.AddCommand(s3Command)
}


func bucket_key(r *http.Request) (string, string) {
	slashIdx := strings.IndexByte(r.URL.Path[1:], '/')
	if slashIdx == -1 {
		return r.URL.Path[1:], ""
	} else {
		return r.URL.Path[1 : 1+slashIdx], strings.Replace(r.URL.Path[2+slashIdx:], "/", "~", -1)
	}
}
