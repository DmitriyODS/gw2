package dto

// LinkStartResult — ответ POST /api/auth/link/start (инициатор заводит спаривание).
type LinkStartResult struct {
	Code         string `json:"code"`   // короткий, для показа/ввода и в QR
	Secret       string `json:"secret"` // приватный, держит только инициатор
	Kind         string `json:"kind"`   // login | tv
	ExpiresInSec int    `json:"expires_in_sec"`
}

// LinkInfo — ответ GET /api/auth/link/info?code= (для экрана подтверждения).
type LinkInfo struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

// LinkClaimResult — ответ POST /api/auth/link/claim: pending | expired | ok(+session).
type LinkClaimResult struct {
	Status  string   `json:"status"`
	Session *Session `json:"session,omitempty"`
}
