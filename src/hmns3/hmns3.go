package hmns3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/rs/zerolog"
)

const dir = "./tmp/s3"

type server struct {
	log zerolog.Logger
}

func StartServer(ctx context.Context) jobs.Job {
	if !config.Config.DigitalOcean.RunFakeServer {
		return jobs.Noop()
	}

	utils.Must(os.MkdirAll(dir, fs.ModePerm))

	s := server{
		log: logging.ExtractLogger(ctx).With().
			Str("module", "S3 server").
			Logger(),
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.getObject(w, r)
		} else if r.Method == http.MethodPut {
			s.putObject(w, r)
		} else {
			panic("Unimplemented method!")
		}
	})

	job := jobs.New()
	srv := http.Server{
		Addr: config.Config.DigitalOcean.FakeAddr,
	}

	s.log.Info().Msg("Starting local S3 server")
	go func() {
		defer job.Done()
		err := srv.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				// This is normal and fine
			} else {
				panic(err)
			}
		}
	}()

	go func() {
		<-ctx.Done()
		s.log.Info().Msg("Shutting down local S3 server")
		srv.Shutdown(context.Background())
	}()

	return job
}

func (s *server) getObject(w http.ResponseWriter, r *http.Request) {
	bucket, key := bucketKey(r)

	file := utils.Must1(os.Open(filepath.Join(dir, bucket, key)))
	io.Copy(w, file)
}

func (s *server) putObject(w http.ResponseWriter, r *http.Request) {
	bucket, key := bucketKey(r)

	w.Header().Set("Location", fmt.Sprintf("/%s", bucket))
	utils.Must(os.MkdirAll(filepath.Join(dir, bucket), fs.ModePerm))
	if key != "" {
		file := utils.Must1(os.Create(filepath.Join(dir, bucket, key)))
		io.Copy(file, r.Body)
	}
}

func bucketKey(r *http.Request) (string, string) {
	slashIdx := strings.IndexByte(r.URL.Path[1:], '/')
	if slashIdx == -1 {
		return r.URL.Path[1:], ""
	} else {
		return r.URL.Path[1 : 1+slashIdx], strings.ReplaceAll(r.URL.Path[2+slashIdx:], "/", "~")
	}
}
