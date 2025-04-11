module github.com/d--j/srs-milter/integration

go 1.24.0

require (
	blitiri.com.ar/go/spf v1.5.1
	github.com/agiledragon/gomonkey/v2 v2.13.0
	github.com/d--j/go-milter v0.10.0
	github.com/d--j/go-milter/integration v0.0.0-20250410163938-ab271d1872c6
	github.com/d--j/srs-milter v0.3.3
)

require (
	github.com/emersion/go-message v0.18.2 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/emersion/go-smtp v0.21.3 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/inconshreveable/log15 v2.16.0+incompatible // indirect
	github.com/jellydator/ttlcache/v3 v3.3.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mileusna/srs v0.0.0-20210306010925-501e7d108e91 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/term v0.31.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/tools v0.32.0 // indirect
)

replace github.com/d--j/srs-milter => ../

replace github.com/mileusna/srs => github.com/d--j/srs v0.0.0-20230317210039-a2adfcc7ffdf
