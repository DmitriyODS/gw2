package domain

import "github.com/DmitriyODS/gw2/back-go/pkg/records"

// Структурная логика полей (нормализация, config-хелперы, search/export) —
// общая с календарями, живёт в pkg/records; здесь — тонкие делегаты на
// доменном типе Field.

// GridCols — ширина сетки карточки реестра: поле занимает от четверти строки
// до всей строки.
const GridCols = 4

const (
	// MaxFileSize — потолок поля «Файл». Такой файл едет частями
	// (pkg/chunkupload), одним запросом его не принять.
	MaxFileSize = 1 << 30
	// MaxImageSize — потолок обложки. Клиент ужимает картинку до этого размера
	// перед отправкой; проверка на сервере — на случай, если пришли мимо него.
	MaxImageSize = 2 << 20
)

// Normalize — привести span'ы к допустимым границам (col 1..GridCols, row ≥1).
func (f *Field) Normalize() { records.NormalizeSpans(&f.ColSpan, &f.RowSpan, &f.Config, GridCols) }

// FieldOptions — варианты select-поля из config (пустой срез, если нет).
func (f Field) FieldOptions() []string { return records.Options(f.Config) }

// SelectMultiple — допускает ли select несколько значений.
func (f Field) SelectMultiple() bool { return records.Multiple(f.Config) }

// NumberPattern — опциональная regex-маска числового поля ("" — без маски).
func (f Field) NumberPattern() string { return records.NumberPattern(f.Config) }

// FieldID — строковый ключ поля в Record.Data.
func FieldID(id int64) string { return records.FieldID(id) }

// SearchContribution — текстовое представление значения поля для search_text.
func SearchContribution(fieldType string, value any) string {
	return records.SearchContribution(fieldType, value)
}

// Exportable — можно ли выгружать поле в xlsx.
func Exportable(fieldType string) bool { return records.Exportable(fieldType) }
