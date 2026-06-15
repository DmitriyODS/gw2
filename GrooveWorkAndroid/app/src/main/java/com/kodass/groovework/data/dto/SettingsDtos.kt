package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// ── Компания: выходные / Groove / приглашение ───────────────────────────────
@Serializable
data class WeekendSettingsDto(
    @SerialName("weekend_days") val weekendDays: List<Int> = emptyList(),
)

@Serializable
data class GrooveSettingsDto(val enabled: Boolean = false)

@Serializable
data class InviteCodeDto(val code: String = "")

// ── Нейро-функции (AI / ProxyAPI) ───────────────────────────────────────────
@Serializable
data class AiSettingsDto(
    val enabled: Boolean = false,
    @SerialName("model_chat") val modelChat: String = "gpt-4o-mini",
    @SerialName("model_embedding") val modelEmbedding: String = "text-embedding-3-small",
    @SerialName("has_key") val hasKey: Boolean = false,
    @SerialName("key_hint") val keyHint: String? = null,
)

// explicitNulls=false: непереданные поля не сериализуются (частичное обновление).
@Serializable
data class AiSettingsUpdate(
    val enabled: Boolean? = null,
    @SerialName("model_chat") val modelChat: String? = null,
    @SerialName("model_embedding") val modelEmbedding: String? = null,
    @SerialName("api_key") val apiKey: String? = null,
    @SerialName("clear_key") val clearKey: Boolean? = null,
)

@Serializable
data class AiTestDto(
    val chat: Boolean = false,
    val embedding: Boolean = false,
    @SerialName("latency_ms") val latencyMs: Long? = null,
    val error: String? = null,
)

@Serializable
data class AiIndexingDto(
    @SerialName("total_tasks") val totalTasks: Int = 0,
    val indexed: Int = 0,
    val pending: Int = 0,
    @SerialName("ai_enabled") val aiEnabled: Boolean = false,
)

@Serializable
data class AiReindexDto(val pending: Int = 0)

// ── YouGile ─────────────────────────────────────────────────────────────────
@Serializable
data class YougileStatusDto(
    val connected: Boolean = false,
    @SerialName("company_enabled") val companyEnabled: Boolean = false,
    @SerialName("yg_login") val ygLogin: String? = null,
    @SerialName("key_fingerprint") val keyFingerprint: String? = null,
    @SerialName("last_validated_at") val lastValidatedAt: String? = null,
    @SerialName("yg_company_id") val ygCompanyId: String? = null,
)

@Serializable
data class YougileRefDto(val id: String = "", val name: String = "")

@Serializable
data class YougileNamedDto(val id: String = "", val title: String = "")

@Serializable
data class YougileSettingsDto(
    val enabled: Boolean = false,
    @SerialName("webhook_registered") val webhookRegistered: Boolean = false,
    @SerialName("yg_company_id") val ygCompanyId: String? = null,
    @SerialName("yg_company_name") val ygCompanyName: String? = null,
    @SerialName("yg_project_id") val ygProjectId: String? = null,
    @SerialName("yg_project_title") val ygProjectTitle: String? = null,
    @SerialName("yg_board_id") val ygBoardId: String? = null,
    @SerialName("yg_board_title") val ygBoardTitle: String? = null,
    @SerialName("yg_completed_column_id") val ygCompletedColumnId: String? = null,
)

@Serializable
data class YougileLoginRequest(val login: String, val password: String)

@Serializable
data class YougileConnectRequest(
    val login: String,
    val password: String,
    @SerialName("yg_company_id") val ygCompanyId: String? = null,
)

@Serializable
data class YougileRotateRequest(val password: String)
