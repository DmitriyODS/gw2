package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class UnitTypeDto(
    val id: Long,
    val name: String = "",
)

@Serializable
data class UnitDto(
    val id: Long,
    val name: String = "",
    @SerialName("task_id") val taskId: Long,
    @SerialName("user_id") val userId: Long = 0,
    val user: UserRef? = null,
    @SerialName("unit_type_id") val unitTypeId: Long = 0,
    @SerialName("unit_type") val unitType: UnitTypeDto? = null,
    @SerialName("datetime_start") val datetimeStart: String? = null,
    @SerialName("datetime_end") val datetimeEnd: String? = null,
    @SerialName("is_edited") val isEdited: Boolean = false,
    @SerialName("created_at") val createdAt: String? = null,
) {
    val isActive: Boolean get() = datetimeEnd == null
}

@Serializable
data class CreateUnitRequest(
    val name: String,
    @SerialName("unit_type_id") val unitTypeId: Long,
)

// datetimeEnd шлём только для завершённого юнита; для активного он null и
// при explicitNulls=false поле опускается — бэкенд не трогает время окончания.
@Serializable
data class UpdateUnitRequest(
    val name: String,
    @SerialName("unit_type_id") val unitTypeId: Long,
    @SerialName("datetime_start") val datetimeStart: String,
    @SerialName("datetime_end") val datetimeEnd: String? = null,
)
