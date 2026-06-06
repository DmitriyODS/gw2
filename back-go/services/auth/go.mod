module github.com/DmitriyODS/gw2/back-go/services/auth

go 1.25.0

require (
	github.com/DmitriyODS/gw2/back-go/gen/proto v0.0.0-00010101000000-000000000000
	github.com/DmitriyODS/gw2/back-go/shared/pkg v0.0.0-00010101000000-000000000000
	github.com/caarlos0/env/v11 v11.4.1
	github.com/gofiber/fiber/v3 v3.3.0
	google.golang.org/grpc v1.69.0
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/gofiber/schema v1.7.1 // indirect
	github.com/gofiber/utils/v2 v2.0.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.71.0 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/protobuf v1.36.0 // indirect
)

replace (
	github.com/DmitriyODS/gw2/back-go/gen/proto => ../../gen/proto
	github.com/DmitriyODS/gw2/back-go/shared/pkg => ../../shared/pkg
)
