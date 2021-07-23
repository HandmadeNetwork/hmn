module git.handmade.network/hmn/hmn

go 1.16

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/alecthomas/chroma v0.9.2
	github.com/frustra/bbcode v0.0.0-20201127003707-6ef347fbe1c8
	github.com/go-stack/stack v1.8.0
	github.com/google/uuid v1.2.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jackc/pgconn v1.8.0
	github.com/jackc/pgtype v1.6.2
	github.com/jackc/pgx/v4 v4.10.1
	github.com/mitchellh/copystructure v1.1.1 // indirect
	github.com/rs/zerolog v1.21.0
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/tcolgate/mp3 v0.0.0-20170426193717-e79c5a46d300
	github.com/teacat/noire v1.1.0
	github.com/wellington/go-libsass v0.9.2
	github.com/yuin/goldmark v1.4.1
	github.com/yuin/goldmark-highlighting v0.0.0-20210516132338-9216f9c5aa01
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d
)

replace (
	github.com/frustra/bbcode v0.0.0-20201127003707-6ef347fbe1c8 => github.com/bvisness/bbcode v0.0.0-20210623031351-ec0e2e2e39d9
	github.com/yuin/goldmark v1.4.1 => github.com/bvisness/goldmark v1.4.1-0.20210707024600-f7e596e26b5e
)
