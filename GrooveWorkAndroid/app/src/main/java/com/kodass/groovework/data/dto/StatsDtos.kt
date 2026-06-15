package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// GET /api/stats/profile — личная статистика пользователя за период.
@Serializable
data class ProfileStatsDto(
    @SerialName("total_hours") val totalHours: Double = 0.0,
    @SerialName("tasks_count") val tasksCount: Int = 0,
    @SerialName("by_unit_types") val byUnitTypes: List<UnitTypeStatDto> = emptyList(),
)

@Serializable
data class UnitTypeStatDto(
    @SerialName("type_id") val typeId: Long = 0,
    val name: String = "",
    val hours: Double = 0.0,
    @SerialName("tasks_count") val tasksCount: Int = 0,
)

// GET /api/stats/common
@Serializable
data class StatsPeriodDto(val from: String = "", val to: String = "")

@Serializable
data class TaskMetricsDto(
    val closed: Int = 0,
    val debt: Int = 0,
    val received: Int = 0,
    val remaining: Int = 0,
)

@Serializable
data class TaskByEmployeeDto(
    val fio: String = "",
    @SerialName("tasks_count") val tasksCount: Int = 0,
    @SerialName("total_hours") val totalHours: Double = 0.0,
    @SerialName("user_id") val userId: Long = 0,
)

@Serializable
data class TaskByHoursDto(
    val name: String = "",
    @SerialName("task_id") val taskId: Long = 0,
    @SerialName("total_hours") val totalHours: Double = 0.0,
)

@Serializable
data class StatsCommonDto(
    val period: StatsPeriodDto = StatsPeriodDto(),
    val tasks: TaskMetricsDto = TaskMetricsDto(),
    @SerialName("tasks_by_employees") val tasksByEmployees: List<TaskByEmployeeDto> = emptyList(),
    @SerialName("tasks_by_hours") val tasksByHours: List<TaskByHoursDto> = emptyList(),
)

// GET /api/stats/extended
@Serializable
data class DeptStatDto(
    @SerialName("dept_id") val deptId: Long = 0,
    val name: String = "",
    @SerialName("tasks_count") val tasksCount: Int = 0,
)

@Serializable
data class UnitTypeTotalDto(
    val name: String = "",
    @SerialName("tasks_count") val tasksCount: Int = 0,
    @SerialName("total_hours") val totalHours: Double = 0.0,
    @SerialName("type_id") val typeId: Long = 0,
)

@Serializable
data class CalendarDayDto(
    val closed: Int = 0,
    val date: String = "",
    val received: Int = 0,
    @SerialName("total_hours") val totalHours: Double = 0.0,
)

@Serializable
data class StatsExtendedDto(
    @SerialName("by_departments") val byDepartments: List<DeptStatDto> = emptyList(),
    @SerialName("by_unit_types") val byUnitTypes: List<UnitTypeTotalDto> = emptyList(),
    val calendar: List<CalendarDayDto> = emptyList(),
)
