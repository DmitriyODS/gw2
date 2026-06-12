// Package dto — JSON-формы REST-ответов, байт-в-байт совместимые с прежними
// Flask-схемами (back/app/schemas/ai_settings.py и api/ai_settings.py).
package dto

// AiSettings — что отдаём наружу. Сырого ключа НЕТ — только маска key_hint.
type AiSettings struct {
	Enabled        bool    `json:"enabled"`
	KeyHint        *string `json:"key_hint"`
	HasKey         bool    `json:"has_key"`
	ModelChat      string  `json:"model_chat"`
	ModelEmbedding string  `json:"model_embedding"`
}

// AiSettingsUpdate — распарсенный PUT-боди (nil — поле не передано).
// api_key пустой/None — «не менять»; удаление ключа — флаг clear_key.
type AiSettingsUpdate struct {
	Enabled        *bool
	APIKey         *string
	ClearKey       bool
	ModelChat      *string
	ModelEmbedding *string
}

// AiTestResult — POST .../ai-settings/test: реальная проверка связи с моделью.
type AiTestResult struct {
	Chat      bool    `json:"chat"`
	Embedding bool    `json:"embedding"`
	Error     *string `json:"error"`
	LatencyMS int64   `json:"latency_ms"`
}

// IndexingStatus — GET .../ai-settings/indexing.
type IndexingStatus struct {
	TotalTasks int    `json:"total_tasks"`
	Indexed    int    `json:"indexed"`
	Pending    int    `json:"pending"`
	Model      string `json:"model"`
	AiEnabled  bool   `json:"ai_enabled"`
}

// ReindexQueued — POST .../ai-settings/reindex-tasks (202 Accepted).
type ReindexQueued struct {
	Queued  bool `json:"queued"`
	Pending int  `json:"pending"`
}
