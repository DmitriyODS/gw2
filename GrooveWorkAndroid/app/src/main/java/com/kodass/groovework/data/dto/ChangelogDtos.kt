package com.kodass.groovework.data.dto

import kotlinx.serialization.Serializable

@Serializable
data class ChangelogDto(val versions: List<ChangelogVersionDto> = emptyList())

@Serializable
data class ChangelogVersionDto(
    val version: String = "",
    val date: String? = null,
    val title: String? = null,
    val description: String? = null,
    val added: List<String> = emptyList(),
    val improved: List<String> = emptyList(),
    val fixed: List<String> = emptyList(),
)
