package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class AttachmentDto(
    val id: Long,
    @SerialName("file_name") val fileName: String = "",
    @SerialName("mime_type") val mimeType: String = "",
    @SerialName("size_bytes") val sizeBytes: Long = 0,
    val url: String = "",
)

@Serializable
data class ReplyPreviewDto(
    val id: Long,
    @SerialName("sender_id") val senderId: Long? = null,
    @SerialName("sender_fio") val senderFio: String? = null,
    val text: String? = null,
    @SerialName("has_attachments") val hasAttachments: Boolean = false,
    val kind: String? = null,
)

@Serializable
data class ForwardedFromDto(
    val id: Long? = null,
    val fio: String? = null,
)

@Serializable
data class CallInfoDto(
    val id: Long,
    val status: String? = null,
    val media: String? = null,
    @SerialName("started_at") val startedAt: String? = null,
    @SerialName("ended_at") val endedAt: String? = null,
    @SerialName("duration_sec") val durationSec: Long? = null,
) {
    // Живой звонок, к которому можно присоединиться/вернуться.
    val isLive: Boolean get() = status == "ringing" || status == "active"
    val isVideo: Boolean get() = media == "video"
}

@Serializable
data class TaskCardDto(
    val id: Long,
    val name: String = "",
    @SerialName("is_archived") val isArchived: Boolean = false,
    val color: String? = null,
    @SerialName("responsible_fio") val responsibleFio: String? = null,
    val deadline: String? = null,
)

@Serializable
data class MessageDto(
    val id: Long,
    @SerialName("conversation_id") val conversationId: Long,
    @SerialName("sender_id") val senderId: Long? = null,
    @SerialName("is_bot") val isBot: Boolean = false,
    val text: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("read_at") val readAt: String? = null,
    val attachments: List<AttachmentDto> = emptyList(),
    @SerialName("reply_to") val replyTo: ReplyPreviewDto? = null,
    @SerialName("forwarded_from") val forwardedFrom: ForwardedFromDto? = null,
    val kind: String = "text",
    val call: CallInfoDto? = null,
    val task: TaskCardDto? = null,
    @SerialName("pinned_at") val pinnedAt: String? = null,
    @SerialName("is_from_support") val isFromSupport: Boolean = false,
)

@Serializable
data class ConversationItemDto(
    val id: Long,
    @SerialName("other_user") val otherUser: UserDto? = null,
    @SerialName("last_message") val lastMessage: MessageDto? = null,
    @SerialName("unread_count") val unreadCount: Int = 0,
    @SerialName("last_message_at") val lastMessageAt: String? = null,
    @SerialName("is_pinned") val isPinned: Boolean = false,
    @SerialName("pinned_at") val pinnedAt: String? = null,
    @SerialName("is_dev_chat") val isDevChat: Boolean = false,
    @SerialName("is_pet_chat") val isPetChat: Boolean = false,
    @SerialName("pet_name") val petName: String? = null,
    @SerialName("company_id") val companyId: Long? = null,
    @SerialName("company_name") val companyName: String? = null,
)

// Ответ POST /conversations, GET /dev-chat и /pet-chat.
@Serializable
data class OpenedConversationDto(
    val id: Long,
    @SerialName("other_user") val otherUser: UserDto? = null,
    @SerialName("is_dev_chat") val isDevChat: Boolean = false,
    @SerialName("is_pet_chat") val isPetChat: Boolean = false,
    @SerialName("pet_name") val petName: String? = null,
)

@Serializable
data class OpenConversationRequest(
    @SerialName("user_id") val userId: Long,
)

@Serializable
data class SendMessageRequest(
    val text: String? = null,
    @SerialName("attachment_ids") val attachmentIds: List<Long>? = null,
    @SerialName("reply_to_id") val replyToId: Long? = null,
    @SerialName("task_id") val taskId: Long? = null,
)

@Serializable
data class ForwardRequest(
    @SerialName("message_id") val messageId: Long,
    @SerialName("conversation_ids") val conversationIds: List<Long> = emptyList(),
    @SerialName("user_ids") val userIds: List<Long> = emptyList(),
)

// Ответ POST /messages/{id}/pin.
@Serializable
data class MessagePinDto(
    val pinned: Boolean = false,
    val message: MessageDto? = null,
)

// Ответ POST /conversations/{id}/pin.
@Serializable
data class ConversationPinDto(
    @SerialName("is_pinned") val isPinned: Boolean = false,
)

@Serializable
data class UnreadDto(val total: Int = 0)

@Serializable
data class PresenceDto(val online: List<Long> = emptyList())
