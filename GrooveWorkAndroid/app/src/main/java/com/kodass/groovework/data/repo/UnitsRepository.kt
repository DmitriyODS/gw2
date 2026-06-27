package com.kodass.groovework.data.repo

import com.kodass.groovework.data.api.UnitsApi
import com.kodass.groovework.data.dto.CreateUnitRequest
import com.kodass.groovework.data.dto.UnitDto
import com.kodass.groovework.data.dto.UnitTypeDto
import com.kodass.groovework.data.dto.UpdateUnitRequest
import com.kodass.groovework.data.network.apiCall
import kotlinx.serialization.json.Json

class UnitsRepository(
    private val api: UnitsApi,
    private val json: Json,
) {
    suspend fun taskUnits(taskId: Long): List<UnitDto> = apiCall(json) { api.taskUnits(taskId) }

    suspend fun createUnit(taskId: Long, name: String, unitTypeId: Long): UnitDto =
        apiCall(json) { api.createUnit(taskId, CreateUnitRequest(name = name, unitTypeId = unitTypeId)) }

    suspend fun updateUnit(unitId: Long, body: UpdateUnitRequest): UnitDto =
        apiCall(json) { api.updateUnit(unitId, body) }

    suspend fun activeUnit(): UnitDto? = apiCall(json) { api.activeUnit() }

    suspend fun stopUnit(unitId: Long): UnitDto = apiCall(json) { api.stopUnit(unitId) }

    suspend fun deleteUnit(unitId: Long) = apiCall(json) { api.deleteUnit(unitId) }

    suspend fun unitTypes(): List<UnitTypeDto> = apiCall(json) { api.unitTypes() }
}
