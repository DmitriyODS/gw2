package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class LoginRequest(
    val login: String,
    val password: String,
)

@Serializable
data class ChangeDefaultRequest(
    @SerialName("new_login") val newLogin: String,
    @SerialName("new_password") val newPassword: String,
    @SerialName("confirm_password") val confirmPassword: String,
)

// Тело login/refresh/change-default: токен + клеймы сессии (PASETO на клиенте не декодируется).
@Serializable
data class SessionResponse(
    @SerialName("access_token") val accessToken: String,
    @SerialName("user_id") val userId: Long,
    @SerialName("force_change") val forceChange: Boolean = false,
    @SerialName("company_id") val companyId: Long? = null,
    @SerialName("company_name") val companyName: String? = null,
    @SerialName("role_level") val roleLevel: Int? = null,
    @SerialName("is_root_admin") val isRootAdmin: Boolean = false,
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

// Короткая ссылка на пользователя в задачах/комментариях.
@Serializable
data class UserRef(
    val id: Long,
    val fio: String = "",
    @SerialName("avatar_path") val avatarPath: String? = null,
)
