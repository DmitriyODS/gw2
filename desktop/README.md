# Groove Work Desktop

Тонкий Electron-клиент: окно грузит прод-версию (`https://gw.kodass.ru`),
поэтому фронт/бэк не дублируются, UI обновляется обычным `make deploy`,
а десктоп-сборка перевыпускается только при правках самой обёртки.

Что даёт обёртка сверх браузера:
- нативные уведомления ОС (Web Notifications страницы маппятся автоматически;
  на Windows — через AppUserModelID);
- иконка в трее: крестик прячет окно, WS живёт — уведомления продолжают
  приходить; выход — из меню трея;
- звук без «разогрева жестом» (autoplay разрешён);
- своя модалка при закрытии с активным юнитом (`will-prevent-unload`);
- демонстрация экрана в звонках (`setDisplayMediaRequestHandler`,
  на macOS — системный пикер);
- внешние ссылки открываются в системном браузере.

## Запуск и сборка

```bash
cd desktop && npm start     # дев-запуск (URL можно переопределить: GW_DESKTOP_URL=http://localhost:5173 npm start)

make desktop                # из корня: dmg (universal) + NSIS exe + AppImage → apps/desktop/ + version.json
make desktop V=1.0.1        # то же с бампом версии обёртки (desktop/package.json)
make deploy-desktop         # залить apps/desktop/ на сервер (nginx раздаёт /apps/desktop/)
```

Скачивание пользователями — карточка в «Настройки → О приложении»
(ссылки `/apps/desktop/GrooveWork-{mac.dmg,win.exe,linux.AppImage}`).

## Обновления

Два независимых контура (по образцу мобильного приложения):
- **UI** приезжает с сервера при каждом `make deploy` — обёртку перевыпускать
  не нужно;
- **сама обёртка**: при старте (и раз в 6 часов) сверяет свою версию с
  `/apps/desktop/version.json`; новее — предлагает скачать установщик своей
  платформы. `version.json` пишет `make desktop` из `desktop/package.json`.

Иконка — `build/icon.png` (512×512, копия `front/public/icons/icon-512.png`;
electron-builder сам конвертирует в icns/ico).

## Подпись

Сборки не подписаны: macOS Gatekeeper и Windows SmartScreen будут
предупреждать. Для тихой установки нужны Apple Developer ID (mac) и
код-сертификат (win) — задаются стандартными env electron-builder
(`CSC_LINK`/`CSC_KEY_PASSWORD`, для mac ещё notarize).

## Переопределение URL

`GW_DESKTOP_URL=...` (env) или `{"url": "https://…"}` в
`<userData>/config.json` (macOS: `~/Library/Application Support/Groove Work/`).
