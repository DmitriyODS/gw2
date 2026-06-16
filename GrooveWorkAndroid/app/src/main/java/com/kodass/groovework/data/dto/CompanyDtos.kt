package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

// Контракт authsvc раздела «Компании» (см. back-go/auth/internal/dto/companies.go).

@Serializable
data class CompanyCreatorDto(
    val id: Long,
    val fio: String = "",
    val login: String = "",
    @SerialName("avatar_path") val avatarPath: String? = null,
)

@Serializable
data class CompanyDto(
    val id: Long,
    val name: String = "",
    val description: String? = null,
    @SerialName("created_by") val createdBy: Long? = null,
    val creator: CompanyCreatorDto? = null,
    @SerialName("is_active") val isActive: Boolean = true,
    // settings — произвольный объект (uses_yougile/uses_stages/uses_calls/uses_groove,
    // weekend_days); PATCH шлём JsonObject с явными ключами (бэк различает «передан»).
    val settings: JsonObject? = null,
    @SerialName("employees_count") val employeesCount: Int = 0,
    @SerialName("tasks_count") val tasksCount: Int = 0,
    @SerialName("created_at") val createdAt: String? = null,
)

// GET /api/companies/mine.
@Serializable
data class CompanyListDto(
    val items: List<CompanyDto> = emptyList(),
    val total: Int = 0,
)

@Serializable
data class CompanyCreateRequest(
    val name: String,
    val description: String? = null,
)

// ── Участники ────────────────────────────────────────────────────────────────
@Serializable
data class AddMemberRequest(
    @SerialName("user_id") val userId: Long,
    @SerialName("role_id") val roleId: Long,
)

@Serializable
data class RoleIdRequest(@SerialName("role_id") val roleId: Long)

// ── Сотрудники компании (создатель) ──────────────────────────────────────────
@Serializable
data class CreateCompanyUserRequest(
    val fio: String,
    val login: String,
    val post: String? = null,
    @SerialName("role_id") val roleId: Long,
    val phone: String? = null,
    val email: String? = null,
    val password: String? = null,
)

@Serializable
data class UpdateCompanyUserRequest(
    val fio: String? = null,
    val login: String? = null,
    val post: String? = null,
    val phone: String? = null,
    val email: String? = null,
)

// ── Email-приглашения ────────────────────────────────────────────────────────
@Serializable
data class CreateInviteRequest(
    val email: String,
    @SerialName("role_id") val roleId: Long,
)

@Serializable
data class InvitePreviewDto(
    @SerialName("company_name") val companyName: String = "",
    @SerialName("role_name") val roleName: String = "",
    val email: String = "",
)

@Serializable
data class OkMessageDto(val message: String = "")
