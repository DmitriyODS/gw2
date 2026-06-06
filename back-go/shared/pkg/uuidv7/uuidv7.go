// Package uuidv7 — генерация uuidv7 (RFC 9562) для случаев, когда нужно
// создать ID на стороне Go (синхронные эвенты, тестовые фикстуры).
//
// Основной источник uuidv7 — БД (`DEFAULT uuidv7()` в PostgreSQL 18).
// Здесь только страховка для тех мест, где БД не задействована.
package uuidv7

import "github.com/google/uuid"

// New возвращает свежий uuidv7. Паникует только если в системе нет
// источника энтропии — на этом этапе всё равно работать невозможно.
func New() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic("uuidv7: no entropy source: " + err.Error())
	}
	return id
}

// Parse — тонкая обёртка над uuid.Parse, чтобы не тащить uuid-пакет
// в каждый вызов.
func Parse(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
