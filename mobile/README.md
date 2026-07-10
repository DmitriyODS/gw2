# Groove Work — мобильный клиент Android (Capacitor)

Тонкая обёртка, брат-близнец десктоп-клиента (`desktop/`): WebView грузит
ПРОД-URL (`https://gw.kodass.ru`, `server.url` в `capacitor.config.json`),
а не локальный бандл — относительные пути `/api`, `/ws`, `/uploads` и
HttpOnly refresh-cookie работают без правок фронта и бэка, обновления UI
прилетают обычным деплоем сервера. Фронт и бэк здесь НЕ дублируются.

## Что нативного

- **Пуш-уведомления (FCM)** — плагин `@capacitor/push-notifications` +
  свой `PushMessagingService` (наследник сервиса плагина, заменяет его в
  манифесте). Сообщения/задачи pushsvc шлёт с notification-payload — их
  системный трей показывает сам даже при убитом приложении; звонки —
  data-only + high priority, уведомление входящего звонка строит
  `PushMessagingService`. Регистрацию токена делает веб-слой
  (`front/src/utils/nativeApp.js` через мост `window.Capacitor`):
  после входа — `POST /api/push/register`, при логауте — unregister.
- **Каналы уведомлений** (`MainActivity`) — id `messages`/`tasks`/
  `calls_incoming` совпадают с `channel_id` пушей pushsvc и с каналами
  прежнего нативного приложения (настройки пользователей переживают
  обновление поверх).
- **Обновление обёртки** (`UpdateChecker`) — аналог `checkShellUpdate`
  Electron: раз в 6 часов сверяет свой `versionCode` с
  `/apps/mobile/version.json`; новее → диалог со скачиванием
  `/apps/mobile/groovework.apk` (ставится поверх — подпись и applicationId
  неизменны).
- **Звонки** — getUserMedia в WebView работает через разрешения приложения
  (CAMERA/RECORD_AUDIO в манифесте, runtime-запросы пробрасывает Capacitor).
  Демонстрация ЭКРАНА в Android WebView недоступна (ограничение платформы) —
  смотреть чужую демонстрацию можно, шарить свой экран — нет.

## Сборка и релиз

```bash
make apk         # npm install + cap sync + gradlew assembleRelease
                 #   → apps/mobile/groovework.apk + зеркало apps/groovework.apk
make deploy-apk  # заливает оба канала на сервер (см. ниже)
```

- **versionCode** (ГГММДДН) Gradle читает из `apps/mobile/version.json` —
  поднять `current_build` перед релизом. **versionName** — из
  `data/changelog.json` (первая запись), как у всего продукта.
- **Подпись** — `android/keystore.properties` + `android/groovework-release.jks`
  (gitignored; шаблон `keystore.properties.example`). Ключ ТОТ ЖЕ, что у
  прежнего нативного приложения: только так APK встаёт обновлением поверх
  установленных. `applicationId com.kodass.groovework` не менять.
- **Firebase** — `android/app/google-services.json` (тот же проект FCM,
  что был у нативного приложения; серверная часть — pushsvc).

## Каналы раздачи (зеркало apps/desktop/)

- `/apps/mobile/groovework.apk` + `/apps/mobile/version.json` — основной канал.
  Сюда смотрят: карточка «Скачать APK» в «О приложении» на сайте и
  автообновление самой обёртки.
- `/apps/groovework.apk` + `/apps/version.json` — ЗЕРКАЛО того же APK
  (пишется автоматически в `make apk`): установленные ранее нативные
  приложения проверяют обновления по этим захардкоженным путям и штатно
  обновляются сразу до текущей обёртки. Каналы всегда синхронны.

## Дев-заметки

- `npx cap sync android` — после правки `capacitor.config.json` или установки
  плагинов (конфиг копируется в assets APK).
- `www/` — только `error.html` (экран «нет соединения», `server.errorPath`);
  реальный UI приезжает с сервера.
- UA обёртки содержит `GrooveWorkApp` (`appendUserAgent`) — по нему фронт
  прячет карточку скачивания APK внутри приложения.
- Мост доступен фронту как `window.Capacitor` (инжектируется в удалённую
  страницу) — фронт НЕ бандлит `@capacitor/*`, все вызовы с guard'ами
  (`front/src/utils/nativeApp.js`).
