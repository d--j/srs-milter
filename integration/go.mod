module github.com/d--j/srs-milter/integration

go 1.19

require (
	blitiri.com.ar/go/spf v1.5.1
	github.com/agiledragon/gomonkey/v2 v2.9.0
	github.com/d--j/go-milter v0.8.2
	github.com/d--j/go-milter/integration v0.0.0-20230315192140-b1c7d01972da
	github.com/d--j/srs-milter v0.0.0-00010101000000-000000000000
)

require (
	github.com/emersion/go-message v0.16.0 // indirect
	github.com/emersion/go-sasl v0.0.0-20220912192320-0145f2c60ead // indirect
	github.com/emersion/go-smtp v0.16.0 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/inconshreveable/log15 v2.16.0+incompatible // indirect
	github.com/jellydator/ttlcache/v3 v3.0.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mileusna/srs v0.0.0-20210306010925-501e7d108e91 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/term v0.6.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
)

replace github.com/d--j/srs-milter => ../

replace github.com/mileusna/srs => github.com/d--j/srs v0.0.0-20230317210039-a2adfcc7ffdf
