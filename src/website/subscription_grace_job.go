package website

import (
	"time"

	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ExpireSubscriptionGracePeriodsJob(dbConn *pgxpool.Pool) *jobs.Job {
	job := jobs.New("subscription grace expiry")
	go func() {
		defer job.Finish()

		t := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-t.C:
				err := func() (err error) {
					defer utils.RecoverPanicAsError(&err)

					n, err := expireDueGracePeriods(job.Ctx, dbConn, SubscriptionNow())
					if err != nil {
						job.Logger.Error().Err(err).Msg("failed to expire subscription grace periods")
						return err
					}
					if n > 0 {
						job.Logger.Info().Int64("num expired", n).Msg("Expired subscription grace periods")
					}
					return nil
				}()
				if err != nil {
					job.Logger.Error().Err(err).Msg("Panicked in subscription grace expiry job")
				}
			case <-job.Canceled():
				return
			}
		}
	}()
	return job
}
