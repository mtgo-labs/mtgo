module github.com/mtgo-labs/mtgo

go 1.26.2

require (
	github.com/klauspost/compress v1.18.6
	github.com/mtgo-labs/storage v0.2.0
	github.com/mtgo-labs/storage/sqlite v0.0.0
	golang.org/x/crypto v0.51.0
	golang.org/x/sync v0.20.0
)

require go.uber.org/goleak v1.3.0

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.44.0 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.34.5 // indirect
)

replace github.com/mtgo-labs/storage => ../storage

replace github.com/mtgo-labs/storage/sqlite => ../storage/sqlite
