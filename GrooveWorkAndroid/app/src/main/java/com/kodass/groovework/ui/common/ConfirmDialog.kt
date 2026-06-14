package com.kodass.groovework.ui.common

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable

// Отложенное действие для подтверждения: один ConfirmDialog обслуживает все
// пункты меню (удаление/открепление), не плодя по диалогу на пункт.
data class ConfirmSpec(
    val title: String,
    val text: String,
    val confirmLabel: String = "Удалить",
    val destructive: Boolean = true,
    val action: () -> Unit,
)

// Переиспользуемая модалка подтверждения деструктивных операций.
@Composable
fun ConfirmDialog(spec: ConfirmSpec, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(spec.title) },
        text = { Text(spec.text) },
        confirmButton = {
            TextButton(onClick = {
                spec.action()
                onDismiss()
            }) {
                Text(
                    spec.confirmLabel,
                    color = if (spec.destructive) MaterialTheme.colorScheme.error
                    else MaterialTheme.colorScheme.primary,
                )
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Отмена") }
        },
    )
}
