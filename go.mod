module github.com/jessepeterson/mysqlscepserver

go 1.16

require (
	github.com/go-kit/kit v0.4.0
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/micromdm/scep/v2 v2.0.1-0.20210515200445-47067d1275f6
)

replace go.mozilla.org/pkcs7 v0.0.0-20200128120323-432b2356ecb1 => github.com/omorsi/pkcs7 v0.0.0-20210217142924-a7b80a2a8568
