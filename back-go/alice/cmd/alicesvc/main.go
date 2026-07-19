// alicesvc — микросервис голосового навыка Алисы (Яндекс.Диалоги).
//
// Принимает публичный вебхук Диалогов (/api/alice/webhook), авторизует
// пользователя по access-токену связки аккаунтов (session.user.access_token,
// PASETO выпускает authsvc — OAuth-провайдер /api/auth/oauth/*), разбирает
// русские голосовые команды (ИИ через aisvc.Chat ключом активной компании,
// фолбэк — регэксп-парсер) и исполняет их gRPC-вызовами сервисов-владельцев:
// tasksvc (задачи/юниты), diarysvc (ежедневники), notesvc (заметки).
// Своего состояния нет (ни БД, ни Redis): мультиход диалога хранится в
// session_state Диалогов.
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/alice/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/alice/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	httpAddr := bootstrap.Env("HTTP_ADDR", ":8104")
	tasksAddr := bootstrap.Env("TASKS_GRPC_ADDR", "localhost:9095")
	diaryAddr := bootstrap.Env("DIARY_GRPC_ADDR", "localhost:9101")
	notesAddr := bootstrap.Env("NOTES_GRPC_ADDR", "localhost:9103")
	aiAddr := bootstrap.Env("AI_GRPC_ADDR", "localhost:9093")

	// Публичный ключ access-токенов PASETO (v4.public): токены выпускает
	// authsvc, мы только проверяем подпись.
	verifier, err := pasetoauth.NewVerifier(bootstrap.MustEnv(log, "PASETO_PUBLIC_KEY"))
	if err != nil {
		log.Error("paseto.bad_public_key", "error", err)
		os.Exit(1)
	}

	ctx, stop := bootstrap.SignalContext()
	defer stop()

	tasks, err := clients.NewTasks(tasksAddr)
	if err != nil {
		log.Error("tasks.client_failed", "error", err)
		os.Exit(1)
	}
	defer tasks.Close()
	diary, err := clients.NewDiary(diaryAddr)
	if err != nil {
		log.Error("diary.client_failed", "error", err)
		os.Exit(1)
	}
	defer diary.Close()
	notes, err := clients.NewNotes(notesAddr)
	if err != nil {
		log.Error("notes.client_failed", "error", err)
		os.Exit(1)
	}
	defer notes.Close()

	deps := service.Deps{Tasks: tasks, Diary: diary, Notes: notes, Verifier: verifier, Log: log}
	// ИИ-разбор фраз (aisvc): явно пустой AI_GRPC_ADDR — только классический парсер.
	if aiAddr != "" {
		ai, err := clients.NewAI(aiAddr)
		if err != nil {
			log.Warn("ai.client_failed", "error", err)
		} else {
			deps.AI = ai
			defer ai.Close()
			log.Info("ai.intent_parser_enabled", "addr", aiAddr)
		}
	}

	svc := service.New(deps)
	httpServer := httptransport.NewServer(svc, log)

	log.Info("listening", "http", httpAddr)
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
	)
}
