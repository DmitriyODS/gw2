package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class LoginRequest(
    val login: String,
    val password: String,
)

// Публичная регистрация: сессию НЕ выдаёт (см. RegisterResultDto). Логин может
// прийти пустым — сервер сгенерит из ФИО; пароль генерируется на клиенте.
@Serializable
data class RegisterRequest(
    val fio: String,
    val email: String,
    val login: String,
    val password: String,
)

// Ответ register: верификация email обязательна перед входом.
@Serializable
data class RegisterResultDto(
    val status: String = "",
    val email: String = "",
)

@Serializable
data class SuggestLoginDto(val login: String = "")

// Подтверждение email: по ссылке ({token}) или вводом кода ({email, code}).
// explicitNulls=false (общий Json) вырежет непереданные поля.
@Serializable
data class VerifyEmailRequest(
    val token: String? = null,
    val email: String? = null,
    val code: String? = null,
)

// Универсальный ответ {status:"ok"} (resend-verification, forgot-password).
@Serializable
data class StatusDto(val status: String = "")

@Serializable
data class ForgotPasswordRequest(val email: String)

@Serializable
data class ResetPasswordRequest(
    val token: String,
    @SerialName("new_password") val newPassword: String,
)

// Ответ reset-password: логин для префилла на экране входа.
@Serializable
data class ResetPasswordResultDto(val login: String = "")

@Serializable
data class ChangeDefaultRequest(
    @SerialName("new_login") val newLogin: String,
    @SerialName("new_password") val newPassword: String,
    @SerialName("confirm_password") val confirmPassword: String,
)

// Тело login/refresh/change-default/select-company/switch-company: токен + клеймы
// сессии (PASETO на клиенте не декодируется). При многокомпанийном логине access_token
// пуст, а needs_company_selection=true — сначала нужен выбор компании (см. select-company).
@Serializable
data class SessionResponse(
    @SerialName("access_token") val accessToken: String = "",
    @SerialName("user_id") val userId: Long,
    @SerialName("force_change") val forceChange: Boolean = false,
    @SerialName("company_id") val companyId: Long? = null,
    @SerialName("company_name") val companyName: String? = null,
    @SerialName("role_level") val roleLevel: Int? = null,
    @SerialName("is_super_admin") val isRootAdmin: Boolean = false,
    @SerialName("needs_company_selection") val needsCompanySelection: Boolean = false,
    @SerialName("select_token") val selectToken: String? = null,
    val companies: List<MembershipDto> = emptyList(),
)

// Компания пользователя и его роль в ней — для пикера при многокомпанийном входе.
@Serializable
data class MembershipDto(
    @SerialName("company_id") val companyId: Long,
    @SerialName("company_name") val companyName: String,
    @SerialName("is_active") val isActive: Boolean = true,
    @SerialName("role_level") val roleLevel: Int = 0,
    @SerialName("role_name") val roleName: String = "",
)

@Serializable
data class SelectCompanyRequest(
    @SerialName("select_token") val selectToken: String,
    @SerialName("company_id") val companyId: Long,
)

@Serializable
data class SwitchCompanyRequest(
    @SerialName("company_id") val companyId: Long,
)

@Serializable
data class RoleRef(
    val id: Long,
    val name: String,
    val level: Int,
)

@Serializable
data class UserDto(
    val id: Long,
    val fio: String = "",
    val login: String? = null,
    val post: String? = null,
    val role: RoleRef? = null,
    @SerialName("company_id") val companyId: Long? = null,
    val phone: String? = null,
    val email: String? = null,
    @SerialName("avatar_path") val avatarPath: String? = null,
    @SerialName("last_seen_at") val lastSeenAt: String? = null,
)

// PATCH /api/users/me — частичное обновление профиля и/или смена пароля.
// explicitNulls=false: непереданные поля не сериализуются.
@Serializable
data class UpdateMeRequest(
    val fio: String? = null,
    val login: String? = null,
    val post: String? = null,
    val phone: String? = null,
    val email: String? = null,
    @SerialName("current_password") val currentPassword: String? = null,
    @SerialName("new_password") val newPassword: String? = null,
    @SerialName("confirm_password") val confirmPassword: String? = null,
)

// Короткая ссылка на пользователя в задачах/комментариях.
@Serializable
data class UserRef(
    val id: Long,
    val fio: String = "",
    @SerialName("avatar_path") val avatarPath: String? = null,
)
