# Firebase / FCM — настройка пуш-уведомлений

Проект Firebase: `groove-work`, Android-приложение `com.kodass.groovework`.

- [x] **Клиент.** `app/google-services.json` — настоящий (проект `groove-work`).
- [ ] **Сервер.** Нужен ОТДЕЛЬНЫЙ service-account ключ (это НЕ google-services.json):
  Firebase Console → ⚙ Project settings → вкладка **Service accounts** →
  *Generate new private key* → скачается json с полями `type`,
  `private_key`, `client_email`. Его содержимое положить в `deploy/.env`
  на сервере как `FIREBASE_CREDENTIALS_JSON=...` (одной строкой, без кавычек)
  и передеплоить.

Клиентский `google-services.json` не является секретом (его ключ всё равно
зашит в APK), поэтому он коммитится в репозиторий. Серверный service-account
ключ — СЕКРЕТ, в репозиторий не кладётся (только в `deploy/.env`).
