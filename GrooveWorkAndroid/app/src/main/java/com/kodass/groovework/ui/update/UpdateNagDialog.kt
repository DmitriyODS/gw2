package com.kodass.groovework.ui.update

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.data.update.AppUpdater
import com.kodass.groovework.data.update.UpdateState
import kotlinx.coroutines.delay

private const val SNOOZE_MS = 10 * 60 * 1000L
private const val RECHECK_MS = 30 * 60 * 1000L

// Обязательная всплывашка обновления: это приложение замещается новой версией
// (веб-обёртка), и старые сборки должны уйти. Пока на сервере лежит сборка
// новее установленной — модалка поверх всего приложения; «Позже» прячет её
// лишь на 10 минут. Кнопка ведёт в штатный AppUpdater (скачивание + установка
// поверх, подпись и applicationId те же).
@Composable
fun UpdateNagDialog(updater: AppUpdater) {
    val state by updater.state.collectAsStateWithLifecycle()

    // Периодический пере-чек: приложение может жить в памяти днями, а нового
    // релиза ещё не было на момент старта. Сетевой запрос троттлит сам updater.
    LaunchedEffect(Unit) {
        while (true) {
            updater.autoCheck()
            delay(RECHECK_MS)
        }
    }

    // Сборка на сервере, о которой мы узнали: нагоняем и после Failed
    // (упавшего скачивания), пока обновление реально не встало.
    var knownBuild by rememberSaveable { mutableLongStateOf(0L) }
    when (val s = state) {
        is UpdateState.Available -> knownBuild = s.build
        is UpdateState.ReadyToInstall -> knownBuild = s.build
        else -> Unit
    }

    var snoozed by rememberSaveable { mutableStateOf(false) }
    LaunchedEffect(snoozed) {
        if (snoozed) {
            delay(SNOOZE_MS)
            snoozed = false
        }
    }

    if (knownBuild == 0L || snoozed) return

    val downloading = state as? UpdateState.Downloading

    AlertDialog(
        onDismissRequest = { /* только кнопками */ },
        properties = DialogProperties(dismissOnBackPress = false, dismissOnClickOutside = false),
        title = { Text("Приложение устарело") },
        text = {
            Column {
                Text(
                    "Вышла новая версия Groove Work. Это приложение больше не будет " +
                        "обновляться — установите новое, чтобы уведомления, чаты и звонки " +
                        "продолжили работать.",
                )
                if (downloading != null) {
                    if (downloading.progress >= 0f) {
                        LinearProgressIndicator(
                            progress = { downloading.progress },
                            modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
                        )
                    } else {
                        LinearProgressIndicator(modifier = Modifier.fillMaxWidth().padding(top = 16.dp))
                    }
                }
                if (state is UpdateState.Failed) {
                    Text(
                        (state as UpdateState.Failed).message,
                        color = MaterialTheme.colorScheme.error,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.padding(top = 12.dp),
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = downloading == null,
                onClick = {
                    when (state) {
                        is UpdateState.ReadyToInstall -> updater.install()
                        else -> updater.downloadBuild(knownBuild)
                    }
                },
            ) {
                Text(if (state is UpdateState.ReadyToInstall) "Установить" else "Скачать и установить")
            }
        },
        dismissButton = {
            TextButton(enabled = downloading == null, onClick = { snoozed = true }) {
                Text("Позже")
            }
        },
    )
}
