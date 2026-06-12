package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class CallParticipantDto(
    @SerialName("user_id") val userId: Long,
    val fio: String = "",
    @SerialName("avatar_path") val avatarPath: String? = null,
    val role: String = "invitee",
    @SerialName("joined_at") val joinedAt: String? = null,
    @SerialName("left_at") val leftAt: String? = null,
    val declined: Boolean = false,
)

@Serializable
data class CallDto(
    val id: Long,
    val kind: String = "p2p",
    val status: String = "ringing",
    val media: String = "video",
    @SerialName("started_at") val startedAt: String? = null,
    @SerialName("initiator_id") val initiatorId: Long = 0,
    @SerialName("initiator_fio") val initiatorFio: String? = null,
    @SerialName("conversation_id") val conversationId: Long? = null,
    @SerialName("share_code") val shareCode: String? = null,
    val participants: List<CallParticipantDto> = emptyList(),
)

@Serializable
data class LivekitInfoDto(
    val token: String,
    val url: String,
)
