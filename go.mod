module git.handmade.network/hmn/hmn

go 1.22

require (
	github.com/HandmadeNetwork/golorem v0.0.0-20220507185207-414965a3a817
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/alecthomas/chroma v0.9.2
	github.com/aws/aws-sdk-go-v2 v1.8.1
	github.com/aws/aws-sdk-go-v2/config v1.6.1
	github.com/aws/aws-sdk-go-v2/credentials v1.3.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.13.0
	github.com/aws/smithy-go v1.7.0
	github.com/emersion/go-ical v0.0.0-20220601085725-0864dccc089f
	github.com/frustra/bbcode v0.0.0-20201127003707-6ef347fbe1c8
	github.com/go-stack/stack v1.8.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/pgx-zerolog v0.0.0-20220923130014-7856b90a65ae
	github.com/jackc/pgx/v5 v5.6.0
	github.com/jpillora/backoff v1.0.0
	github.com/rs/zerolog v1.28.0
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.8.1
	github.com/tcolgate/mp3 v0.0.0-20170426193717-e79c5a46d300
	github.com/teacat/noire v1.1.0
	github.com/yuin/goldmark v1.4.13
	github.com/yuin/goldmark-highlighting v0.0.0-20210516132338-9216f9c5aa01
	golang.org/x/crypto v0.31.0
	golang.org/x/image v0.23.0
	mvdan.cc/xurls/v2 v2.4.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.2.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.2.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.5.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.3.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.6.2 // indirect
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/evanw/esbuild v0.21.4
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.1.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/teambition/rrule-go v1.7.2 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/frustra/bbcode v0.0.0-20201127003707-6ef347fbe1c8 => github.com/HandmadeNetwork/bbcode v0.0.0-20210623031351-ec0e2e2e39d9
	github.com/yuin/goldmark v1.4.1 => github.com/HandmadeNetwork/goldmark v1.4.1-0.20210707024600-f7e596e26b5e
)
