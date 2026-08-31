package config

import (
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
)

func init() {
	hmnurl.SetGlobalBaseUrl(Config.BaseUrl)
	hmnurl.SetCacheBustVersion(fmt.Sprint(time.Now().Unix()))
	hmnurl.SetS3BaseUrl(Config.DigitalOcean.AssetsPublicUrlRoot)
}
