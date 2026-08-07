package domain

import (
	"net/http"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

// Ошибки диска. Коды стабильны — на них опирается фронт.
var (
	ErrNotFound   = apierror.New("NOT_FOUND", "Не найдено", http.StatusNotFound)
	ErrForbidden  = apierror.New("FORBIDDEN", "Нет доступа", http.StatusForbidden)
	ErrValidation = apierror.New("VALIDATION", "Проверьте заполненные поля", http.StatusUnprocessableEntity)
	// Папку нельзя положить в саму себя или в собственного потомка — иначе
	// поддерево оторвётся от дерева и станет недостижимым.
	ErrFolderCycle = apierror.New("FOLDER_CYCLE", "Папку нельзя перенести в саму себя", http.StatusConflict)
	ErrFileTooBig  = apierror.New("FILE_TOO_BIG", "Файл больше 500 МБ", http.StatusRequestEntityTooLarge)
	ErrEmptyFile   = apierror.New("EMPTY_FILE", "Файл пустой", http.StatusUnprocessableEntity)
	// ErrChunkOffset — кусок пришёл не с той позиции: ответ несёт принятый
	// объём, чтобы клиент продолжил ровно с него, а не начинал заново.
	ErrChunkOffset = apierror.New("CHUNK_OFFSET", "Кусок пришёл не по порядку", http.StatusConflict)
)
