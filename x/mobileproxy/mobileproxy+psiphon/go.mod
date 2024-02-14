// Fake module, just to pin the dependencies.
module mobileproxy.local

go 1.20

// Use checked out code.
replace github.com/Jigsaw-Code/outline-sdk/x => ../..

require (
	github.com/Jigsaw-Code/outline-sdk/x v0.0.0-00010101000000-000000000000
	github.com/Psiphon-Labs/psiphon-tunnel-core v0.0.14-beta-ios.0.20240130163824-f406d7f78492
)

require (
	filippo.io/bigmod v0.0.1 // indirect
	filippo.io/keygen v0.0.0-20230306160926-5201437acf8e // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/Jigsaw-Code/outline-sdk v0.0.13-0.20240212211908-de32e39b12e5 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Psiphon-Labs/bolt v0.0.0-20200624191537-23cedaef7ad7 // indirect
	github.com/Psiphon-Labs/goptlib v0.0.0-20200406165125-c0e32a7a3464 // indirect
	github.com/Psiphon-Labs/qtls-go1-19 v0.0.0-20230608213623-d58aa73e519a // indirect
	github.com/Psiphon-Labs/qtls-go1-20 v0.0.0-20230608214729-dd57d6787acf // indirect
	github.com/Psiphon-Labs/quic-go v0.0.0-20230626192210-73f29effc9da // indirect
	github.com/Psiphon-Labs/tls-tris v0.0.0-20230824155421-58bf6d336a9a // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/armon/go-proxyproto v0.1.0 // indirect
	github.com/bifurcation/mint v0.0.0-20210616192047-fd18df995463 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/cognusion/go-cache-lru v0.0.0-20170419142635-f73e2280ecea // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.0.2 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/elazarl/goproxy v0.0.0-20231117061959-7cc037d33fb5 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20231117061959-7cc037d33fb5 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/florianl/go-nfqueue v1.3.1 // indirect
	github.com/flynn/noise v1.0.0 // indirect
	github.com/gaukas/godicttls v0.0.4 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-task/task/v3 v3.34.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20211214055906-6f57359322fd // indirect
	github.com/grafov/m3u8 v0.12.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/juju/ratelimit v1.0.2 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/marusama/semaphore v0.0.0-20190110074507-6952cef993b2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-zglob v0.0.4 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mroth/weightedrand v1.0.0 // indirect
	github.com/onsi/ginkgo/v2 v2.15.0 // indirect
	github.com/onsi/gomega v1.31.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pion/dtls/v2 v2.2.7 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/sctp v1.8.8 // indirect
	github.com/pion/stun v0.6.1 // indirect
	github.com/pion/transport/v2 v2.2.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/radovskyb/watcher v1.0.7 // indirect
	github.com/refraction-networking/conjure v0.7.11-0.20240130155008-c8df96195ab2 // indirect
	github.com/refraction-networking/ed25519 v0.1.2 // indirect
	github.com/refraction-networking/gotapdance v1.7.10 // indirect
	github.com/refraction-networking/obfs4 v0.1.2 // indirect
	github.com/refraction-networking/utls v1.3.3 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sajari/fuzzy v1.0.0 // indirect
	github.com/sergeyfrolov/bsbuffer v0.0.0-20180903213811-94e85abb8507 // indirect
	github.com/shadowsocks/go-shadowsocks2 v0.1.5 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/wader/filtertransport v0.0.0-20200316221534-bdd9e61eee78 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/goptlib v1.5.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/exp v0.0.0-20240213143201-ec583247a57a // indirect
	golang.org/x/mobile v0.0.0-20240213143359-d1f7d3436075 // indirect
	golang.org/x/mod v0.15.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/term v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.18.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/sh/v3 v3.7.0 // indirect
)
