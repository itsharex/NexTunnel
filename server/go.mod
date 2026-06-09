module github.com/nextunnel/server

go 1.25.0

require (
	github.com/cilium/ebpf v0.11.0
	github.com/google/uuid v1.6.0
	github.com/nextunnel/pkg v0.0.0
	github.com/oschwald/maxminddb-golang v1.13.1
	github.com/pion/logging v0.2.4
	github.com/pion/stun/v2 v2.0.0
	github.com/pion/turn/v4 v4.1.4
	github.com/quic-go/quic-go v0.59.1
	golang.org/x/crypto v0.52.0
	modernc.org/sqlite v1.52.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pion/dtls/v2 v2.2.7 // indirect
	github.com/pion/dtls/v3 v3.0.7 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/stun/v3 v3.0.1 // indirect
	github.com/pion/transport/v2 v2.2.1 // indirect
	github.com/pion/transport/v3 v3.0.8 // indirect
	github.com/pion/transport/v4 v4.0.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	golang.org/x/exp v0.0.0-20230224173230-c95f2b4c22f2 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/nextunnel/pkg => ../pkg
