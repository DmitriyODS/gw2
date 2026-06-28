package com.kodass.groovework.data.units

import com.kodass.groovework.data.dto.UnitDto
import com.kodass.groovework.data.repo.UnitsRepository
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.notifications.Notifier
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

// Единый источник правды об активном юните пользователя: держит состояние,
// постит/гасит ongoing-уведомление с отсчётом и кнопкой «Завершить», слушает
// сокет-события (старт/стоп с других устройств). У пользователя одновременно
// не более одного активного юнита — пока он есть, новые начинать нельзя.
class UnitManager(
    private val repo: UnitsRepository,
    private val session: SessionManager,
    gateway: GatewayClient,
    private val json: Json,
    private val notifier: Notifier,
    private val scope: CoroutineScope,
) {
    private val _activeUnit = MutableStateFlow<UnitDto?>(null)
    val activeUnit: StateFlow<UnitDto?> = _activeUnit

    // Запрос «открыть модалку юнита» (тап по уведомлению / по плашке) — UI снимает.
    val showSheet = MutableStateFlow(false)

    private val _errors = MutableSharedFlow<String>(extraBufferCapacity = 4)
    val errors: SharedFlow<String> = _errors

    private val myUserId: Long?
        get() = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId

    init {
        // Уведомление держим в синхроне с состоянием активного юнита.
        scope.launch {
            _activeUnit.collect { unit ->
                if (unit != null) notifier.showUnit(unit) else notifier.cancelUnit()
            }
        }
        scope.launch { gateway.events.collect { handleEvent(it.event, it.data) } }
        // Подтягиваем активный юнит при входе, сбрасываем при выходе.
        scope.launch {
            session.authState.collect { state ->
                if (state is AuthState.LoggedIn && !state.claims.forceChange) refresh()
                else clear()
            }
        }
    }

    fun refresh() {
        scope.launch { runCatching { _activeUnit.value = repo.activeUnit() } }
    }

    // Запуск юнита текущим пользователем. 409 → «уже есть активный юнит».
    suspend fun startUnit(taskId: Long, name: String, unitTypeId: Long): Result<UnitDto> =
        runCatching { repo.createUnit(taskId, name, unitTypeId) }
            .onSuccess { _activeUnit.value = it }

    // Создать новый юнит от существующего (на себя), сохранив название и тип.
    // Ограничение «1 активный» проверяет сервер (409); ошибки идут в errors.
    suspend fun cloneUnit(unit: UnitDto): Boolean {
        if (_activeUnit.value != null) {
            _errors.tryEmit("У вас уже есть активный юнит")
            return false
        }
        return startUnit(unit.taskId, unit.name, unit.unitType?.id ?: unit.unitTypeId)
            .onFailure { e ->
                val api = e as? com.kodass.groovework.data.network.ApiException
                _errors.tryEmit(
                    if (api?.status == 409) "У вас уже есть активный юнит"
                    else api?.message ?: "Не удалось запустить юнит"
                )
            }
            .isSuccess
    }

    // Завершение активного юнита (из плашки/модалки приложения).
    fun stopActiveUnit(onDone: (() -> Unit)? = null) {
        val id = _activeUnit.value?.id ?: return
        stopUnit(id, onDone)
    }

    // Завершение по id (в т.ч. из BroadcastReceiver уведомления, когда процесс
    // мог быть перезапущен и in-memory состояние пусто).
    fun stopUnit(unitId: Long, onDone: (() -> Unit)? = null) {
        scope.launch {
            if (stopUnitSuspend(unitId)) onDone?.invoke()
        }
    }

    // Suspend-вариант для вызова из BroadcastReceiver под goAsync(): процесс
    // держится живым до завершения сетевого запроса (иначе при медленной сети
    // система убивает процесс receiver'а и юнит не завершается).
    suspend fun stopUnitSuspend(unitId: Long): Boolean {
        return try {
            repo.stopUnit(unitId)
            if (_activeUnit.value?.id == unitId) _activeUnit.value = null
            notifier.cancelUnit()
            true
        } catch (e: Exception) {
            _errors.tryEmit((e as? com.kodass.groovework.data.network.ApiException)?.message
                ?: "Не удалось завершить юнит")
            false
        }
    }

    fun requestShowSheet() {
        if (_activeUnit.value == null) refresh()
        showSheet.value = true
    }

    fun consumeShowSheet() {
        showSheet.value = false
    }

    private fun clear() {
        _activeUnit.value = null
        showSheet.value = false
    }

    private fun handleEvent(event: String, data: kotlinx.serialization.json.JsonElement?) {
        when (event) {
            "unit:started" -> {
                val unit = data?.let { runCatching { json.decodeFromJsonElement<UnitDto>(it) }.getOrNull() } ?: return
                if (unit.userId == myUserId && unit.isActive) _activeUnit.value = unit
            }
            "unit:stopped", "unit:deleted" -> {
                val unitId = data.longField("unit_id") ?: return
                if (_activeUnit.value?.id == unitId) _activeUnit.value = null
            }
            "unit:updated" -> {
                val unitId = data.longField("unit_id") ?: return
                val current = _activeUnit.value ?: return
                if (current.id == unitId) {
                    // Юнит могли отредактировать (например, выставить datetime_end) —
                    // перепроверяем актуальность через refresh.
                    refresh()
                }
            }
        }
    }
}

// Момент старта в epoch-миллисекундах — для отсчёта в плашке и уведомлении.
fun unitStartMillis(unit: UnitDto): Long =
    com.kodass.groovework.ui.common.parseIso(unit.datetimeStart)?.toInstant()?.toEpochMilli()
        ?: System.currentTimeMillis()
