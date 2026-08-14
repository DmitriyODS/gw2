/* Правовые документы платформы: действующая редакция, реквизиты оператора и
   каталог документов.

   Работать в приложении можно только приняв действующую редакцию: пока версия
   не принята, access-токен несёт клейм legal_required и ВСЕ сервисы отвечают
   403 LEGAL_CONSENT_REQUIRED. Гейт серверный (back-go/auth/internal/domain/
   legal.go) — здесь только тексты и то, как их показать.

   LEGAL_VERSION обязана совпадать с domain.LegalVersion: сервер не примет
   согласие на чужую редакцию (409 LEGAL_VERSION_MISMATCH). Правите документ —
   поднимайте версию в ОБОИХ местах, и платформа пересоберёт согласия у всех.

   Сами тексты лежат отдельными модулями и грузятся динамически: их читают
   один раз в жизни аккаунта, в первый кадр приложения они не нужны. */

export const LEGAL_VERSION = '1.0'

// Дата редакции — показывается в шапке документов и в карточке настроек.
export const LEGAL_DATE = '14 августа 2026 г.'

/* Реквизиты оператора (правообладателя). ЕДИНСТВЕННОЕ место, где они заданы:
   тексты документов подставляют их отсюда. Плейсхолдеры в фигурных скобках
   заменить на реальные значения — больше править нигде ничего не нужно. */
export const OPERATOR = {
  name: 'Акционерное общество «КОДАСС»',
  shortName: 'АО «КОДАСС»',
  ogrn: '{{ОГРН}}',
  inn: '{{ИНН}}',
  kpp: '{{КПП}}',
  address: '{{ЮРИДИЧЕСКИЙ АДРЕС}}',
  email: '{{EMAIL ДЛЯ ОБРАЩЕНИЙ}}',
  site: 'gw.kodass.ru',
}

// Строка реквизитов одной фразой — для вводных абзацев документов.
export const OPERATOR_LINE =
  `${OPERATOR.name} (ОГРН ${OPERATOR.ogrn}, ИНН ${OPERATOR.inn}, адрес: ${OPERATOR.address})`

/* Документы редакции. Ключи ≡ domain.DocLicense/DocPrivacy/DocConsent.

   consent — согласие в поле `group: 'pdn'`; остальные — в группе 'terms'.
   Группа = одна галочка в плашке: согласие на обработку персональных данных
   не может быть «пакетным» с лицензией (ч.1 ст.9 152-ФЗ требует конкретного и
   предметного согласия), поэтому галочек ровно две. */
export const LEGAL_DOCUMENTS = [
  {
    key: 'license',
    group: 'terms',
    title: 'Лицензионное соглашение',
    short: 'Условия использования приложения',
    load: () => import('@/legal/license.js'),
  },
  {
    key: 'privacy',
    group: 'terms',
    title: 'Политика конфиденциальности',
    short: 'Какие данные обрабатываются и зачем',
    load: () => import('@/legal/privacy.js'),
  },
  {
    key: 'consent',
    group: 'pdn',
    title: 'Согласие на обработку персональных данных',
    short: 'Цели, перечень данных, срок и порядок отзыва',
    load: () => import('@/legal/consent.js'),
  },
]

// Галочки плашки: подпись, к чему относится, и какие документы покрывает.
export const LEGAL_GROUPS = [
  {
    key: 'terms',
    label: 'Я прочитал(а) и принимаю Лицензионное соглашение и Политику конфиденциальности',
  },
  {
    key: 'pdn',
    label: 'Я даю согласие на обработку моих персональных данных на условиях, указанных в согласии',
  },
]

// Все ключи документов редакции — их сервер и ждёт в теле принятия.
export const LEGAL_DOC_KEYS = LEGAL_DOCUMENTS.map((d) => d.key)

export function legalDocument(key) {
  return LEGAL_DOCUMENTS.find((d) => d.key === key) || null
}

// Текст документа (markdown). Загруженное кэшируется — плашка и раздел
// настроек открывают одни и те же документы.
const cache = new Map()

export async function loadLegalText(key) {
  if (cache.has(key)) return cache.get(key)
  const doc = legalDocument(key)
  if (!doc) return ''
  const mod = await doc.load()
  cache.set(key, mod.default)
  return mod.default
}
