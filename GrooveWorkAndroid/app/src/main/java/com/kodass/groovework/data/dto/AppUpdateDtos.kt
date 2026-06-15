package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// Номер сборки приложения, лежащего на сервере (apps/version.json).
// Сравнивается с собственным versionCode при проверке обновлений.
@Serializable
data class AppBuildDto(
    @SerialName("current_build") val currentBuild: Long = 0,
)
