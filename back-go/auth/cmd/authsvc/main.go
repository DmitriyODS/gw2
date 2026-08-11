// authsvc — микросервис авторизации, пользователей и компаний Groove Work.
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/auth/*, /api/users/*,
// /api/companies/* (кроме regex ai-settings — он в aisvc), /api/roles и
// /api/backup/* (за nginx, мимо Flask). Выпускает PASETO-токены: access —
// v4.public (Ed25519, PASETO_PRIVATE_KEY), проверяется остальными сервисами
// по публичному ключу; refresh — v4.local (PASETO_REFRESH_KEY) в
// HttpOnly-cookie.
//
// Зависимости: общая PostgreSQL платформы (схему ведёт goose-migrate; пароли —
// pgcrypto), Redis (анти-brute-force), общий uploads-volume (аватарки).
package main

import (
	"net"
	"os"
	"time"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/avatar"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/repository/redisx"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/service"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
	httptransport "github.com/DmitriyODS/gw2/back-go/auth/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour

	// defaultGeoURL — бесплатный гео-сервис без ключа (ip-api.com, ~45 запросов
	// в минуту). Свой провайдер — GEOIP_URL с подстановками {ip} и {token}.
	defaultGeoURL = "http://ip-api.com/json/{ip}?fields=city&lang=ru"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "/app/uploads")
	mailAddr := bootstrap.Env("MAIL_GRPC_ADDR", "localhost:9098")
	// Публичный базовый URL приложения — для ссылок подтверждения email в письмах.
	appBaseURL := bootstrap.Env("APP_PUBLIC_BASE_URL", "http://localhost:5173")

	privateKey := bootstrap.MustEnv(log, "PASETO_PRIVATE_KEY")
	refreshKey := bootstrap.MustEnv(log, "PASETO_REFRESH_KEY")
	issuer, err := token.NewIssuer(privateKey, refreshKey, accessTTL, refreshTTL)
	if err != nil {
		log.Error("paseto.bad_keys", "error", err)
		os.Exit(1)
	}

	ctx, stop := bootstrap.SignalContext()
	defer stop()

	pool := bootstrap.MustPostgres(ctx, log, dbURL)
	defer pool.Close()
	rdb := bootstrap.MustRedis(log, redisURL)
	defer rdb.Close()

	repo := postgres.NewUserRepository(pool)
	companies := postgres.NewCompanyRepository(pool)
	backup := postgres.NewBackupStore(pool)
	verifications := postgres.NewVerificationStore(pool)
	passwordResets := postgres.NewPasswordResetStore(pool)
	companyInvites := postgres.NewCompanyInviteStore(pool)
	throttle := redisx.NewLoginThrottle(rdb, log)
	limiter := redisx.NewRateLimiter(rdb, log)
	deviceLinks := redisx.NewDeviceLinkStore(rdb)
	fileStore := storage.FromEnv(log, uploadFolder)
	avatars := avatar.NewStorage(fileStore)

	mail, err := clients.NewMail(mailAddr, log)
	if err != nil {
		log.Error("mail_grpc.bad_addr", "error", err)
		os.Exit(1)
	}
	defer mail.Close()

	svc := service.New(repo, companies, backup, throttle, issuer, avatars,
		verifications, passwordResets, companyInvites, deviceLinks, mail, appBaseURL, log)
	// REGISTER_IP_LIMIT поднимают испытательные стенды: сотни аккаунтов с
	// одного адреса — их обычный режим, а не злоупотребление.
	svc.WithRateLimiter(limiter, bootstrap.EnvInt("REGISTER_IP_LIMIT", 0))
	// Полный бэкап включает все загруженные файлы (медиа мессенджера, файлы
	// реестров/календарей/заметок/портала, аватарки) — даём доступ к корневому
	// файловому хранилищу.
	svc.WithFiles(fileStore)

	// Лимиты тарифа (сколько компаний создаёт пользователь и сколько человек
	// помещается в компанию) — gRPC billingsvc. Пустой адрес выключает
	// проверки, недоступный биллинг их не блокирует (fail-open).
	billing, err := billingclient.New(bootstrap.Env("BILLING_GRPC_ADDR", ""), log)
	if err != nil {
		log.Error("billing.dial_failed", "error", err)
		os.Exit(1)
	}
	defer billing.Close()
	svc.WithBilling(billing)

	// Перенос компании: архив собирается из кусков владельцев контента. Пустая
	// спецификация выключает перенос, не поднявшийся раздел просто выпадает
	// из архива — вида «tasks=tasks:9095,registry=registry:9099».
	transfer, err := companydata.Dial(bootstrap.Env("COMPANY_DATA_ADDRS", ""))
	if err != nil {
		log.Error("company_data.dial_failed", "error", err)
		os.Exit(1)
	}
	defer transfer.Close()
	svc.WithTransfer(transfer)

	// Реестр входов профиля («Авторизация и сессии»). Город по IP — внешний
	// гео-сервис: шаблон URL с {ip} и {token}, ответ с полем "city" (смена
	// провайдера — правка env, не кода); пустой GEOIP_URL выключает гео, и
	// карточка сеанса показывает IP.
	// Пустой GEOIP_URL — осознанное «гео не нужно», поэтому именно LookupEnv:
	// у bootstrap.Env пустое значение равносильно отсутствию и включило бы
	// провайдера по умолчанию.
	geoURL, ok := os.LookupEnv("GEOIP_URL")
	if !ok {
		geoURL = defaultGeoURL
	}
	var geo domain.GeoResolver
	if geoURL != "" {
		geo = clients.NewGeoIP(geoURL, bootstrap.Env("GEOIP_TOKEN", ""), redisx.NewGeoCache(rdb), log)
	}
	svc.WithSessions(postgres.NewSessionStore(pool), geo)

	// OAuth-провайдер для связки аккаунтов навыка Алисы (пустые креды — выключен).
	if cid, secret := bootstrap.Env("OAUTH_ALICE_CLIENT_ID", ""), bootstrap.Env("OAUTH_ALICE_CLIENT_SECRET", ""); cid != "" && secret != "" {
		svc.WithOAuth(redisx.NewOAuthCodeStore(rdb), cid, secret)
		log.Info("oauth.alice_enabled", "client_id", cid)
	}
	// Вход через Яндекс ID (пустые креды — кнопка скрыта).
	if cid, secret := bootstrap.Env("YANDEX_OAUTH_CLIENT_ID", ""), bootstrap.Env("YANDEX_OAUTH_CLIENT_SECRET", ""); cid != "" && secret != "" {
		svc.WithYandex(clients.NewYandex(cid, secret), cid)
		log.Info("oauth.yandex_login_enabled", "client_id", cid)
	}
	eps := endpoint.New(svc)

	httpAddr := bootstrap.Env("HTTP_ADDR", ":8091")
	httpServer := httptransport.NewServer(eps, token.VerifierFromIssuer(issuer), repo, log)

	// gRPC здесь ровно один и узкий: биллинг спрашивает про фото профиля для
	// раздела «Настройки → Хранилище». Проверка токенов по-прежнему локальная
	// у всех — сюда за ней никто не ходит.
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9091")
	grpcServer := googrpc.NewServer()
	storagefiles.Register(grpcServer, svc)
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	log.Info("listening", "http", httpAddr, "grpc", grpcAddr, "public_key", issuer.PublicKeyHex())
	bootstrap.Run(ctx, log,
		bootstrap.Component{
			Name: "http",
			Run:  func() error { return httpServer.Listen(httpAddr) },
			Stop: func() {
				if err := httpServer.Shutdown(); err != nil {
					log.Warn("http.shutdown_failed", "error", err)
				}
			},
		},
		bootstrap.Component{
			Name: "grpc",
			Run:  func() error { return grpcServer.Serve(listener) },
			Stop: grpcServer.GracefulStop,
		},
	)
}
