package com.kodass.groovework.ui.registries

import android.Manifest
import android.content.pm.PackageManager
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.InsertDriveFile
import androidx.compose.material.icons.filled.PhotoCamera
import androidx.compose.material.icons.filled.PhotoLibrary
import androidx.compose.material3.AssistChip
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.window.Dialog
import androidx.core.content.ContextCompat
import coil3.compose.AsyncImage
import com.kodass.groovework.data.dto.RegistryFieldDto
import com.kodass.groovework.data.dto.RegistryFieldType
import com.kodass.groovework.data.files.readPickedFile
import com.kodass.groovework.ui.common.LocalServerUrl
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import java.time.Instant
import java.time.LocalDate
import java.time.LocalTime
import java.time.ZoneId
import java.time.ZoneOffset

// Один редактор значения на каждый тип поля. value/onChange работают с JsonElement
// (как хранится в Record.data). Для image/file загрузка идёт через onUpload, а
// onChange(null) очищает значение.
@Composable
fun RegistryFieldInput(
    field: RegistryFieldDto,
    value: JsonElement?,
    uploading: Boolean,
    onChange: (JsonElement?) -> Unit,
    onUpload: (fileName: String, mime: String, bytes: ByteArray) -> Unit,
) {
    when (field.type) {
        RegistryFieldType.TEXT -> {
            val text = value.asStringOrNull() ?: ""
            OutlinedTextField(
                value = text,
                onValueChange = { onChange(if (it.isEmpty()) null else JsonPrimitive(it)) },
                singleLine = !field.config.multiline,
                minLines = if (field.config.multiline) 3 else 1,
                modifier = Modifier.fillMaxWidth(),
            )
        }
        RegistryFieldType.NUMBER -> {
            val text = value.asStringOrNull() ?: ""
            OutlinedTextField(
                value = text,
                onValueChange = { onChange(if (it.isBlank()) null else JsonPrimitive(it)) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                placeholder = field.config.pattern?.takeIf { it.isNotBlank() }?.let { { Text("Шаблон: $it") } },
                modifier = Modifier.fillMaxWidth(),
            )
        }
        RegistryFieldType.LINK -> {
            val text = value.asStringOrNull() ?: ""
            OutlinedTextField(
                value = text,
                onValueChange = { onChange(if (it.isBlank()) null else JsonPrimitive(it)) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                placeholder = { Text("https://…") },
                modifier = Modifier.fillMaxWidth(),
            )
        }
        RegistryFieldType.CHECKBOX -> {
            val on = value.asBool()
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Switch(checked = on, onCheckedChange = { onChange(JsonPrimitive(it)) })
                Text(if (on) "Да" else "Нет", style = MaterialTheme.typography.bodyLarge)
            }
        }
        RegistryFieldType.SELECT -> {
            if (field.config.multiple) MultiSelectInput(field, value, onChange)
            else SingleSelectInput(field, value, onChange)
        }
        RegistryFieldType.DATETIME -> DateTimeInput(field, value, onChange)
        RegistryFieldType.IMAGE -> ImageInput(field, value, uploading, onChange, onUpload)
        RegistryFieldType.FILE -> FileInput(value, uploading, onChange, onUpload)
        else -> {
            val text = value.asStringOrNull() ?: ""
            OutlinedTextField(
                value = text,
                onValueChange = { onChange(if (it.isEmpty()) null else JsonPrimitive(it)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}

@Composable
private fun SingleSelectInput(
    field: RegistryFieldDto,
    value: JsonElement?,
    onChange: (JsonElement?) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    val current = value.asStringOrNull()
    Box(modifier = Modifier.fillMaxWidth()) {
        OutlinedButton(onClick = { expanded = true }, modifier = Modifier.fillMaxWidth()) {
            Text(
                current ?: "Выберите",
                modifier = Modifier.weight(1f),
                color = if (current == null) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface,
            )
            Icon(Icons.Filled.ArrowDropDown, contentDescription = null)
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            DropdownMenuItem(text = { Text("— не выбрано —") }, onClick = { onChange(null); expanded = false })
            field.config.options.forEach { opt ->
                DropdownMenuItem(text = { Text(opt) }, onClick = { onChange(JsonPrimitive(opt)); expanded = false })
            }
        }
    }
}

@Composable
private fun MultiSelectInput(
    field: RegistryFieldDto,
    value: JsonElement?,
    onChange: (JsonElement?) -> Unit,
) {
    val selected = value.asSelectValues().toSet()
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
        field.config.options.forEach { opt ->
            val isOn = opt in selected
            FilterChip(
                selected = isOn,
                onClick = {
                    val next = if (isOn) selected - opt else selected + opt
                    onChange(if (next.isEmpty()) null else JsonArray(next.map { JsonPrimitive(it) }))
                },
                label = { Text(opt) },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DateTimeInput(
    field: RegistryFieldDto,
    value: JsonElement?,
    onChange: (JsonElement?) -> Unit,
) {
    val cfg = field.config
    val needsDate = cfg.monthDay || cfg.year
    val current = value.asStringOrNull()
    val display = formatDateTime(current, cfg).ifBlank { "Не выбрано" }

    var showDate by remember { mutableStateOf(false) }
    var showTime by remember { mutableStateOf(false) }
    var pickedDate by remember { mutableStateOf<LocalDate?>(null) }

    val existing = remember(current) { current?.let { runCatching { Instant.parse(it) }.getOrNull() }?.atZone(ZoneId.systemDefault()) }

    fun commit(date: LocalDate?, time: LocalTime) {
        val d = date ?: LocalDate.now()
        val instant = d.atTime(time).atZone(ZoneId.systemDefault()).toInstant()
        onChange(JsonPrimitive(instant.toString()))
    }

    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlinedButton(
            onClick = { if (needsDate) showDate = true else showTime = true },
            modifier = Modifier.weight(1f),
        ) {
            Icon(Icons.Filled.CalendarMonth, contentDescription = null, modifier = Modifier.size(18.dp))
            Text(display, modifier = Modifier.padding(start = 8.dp))
        }
        if (current != null) {
            IconButton(onClick = { onChange(null) }) {
                Icon(Icons.Filled.Close, contentDescription = "Очистить")
            }
        }
    }

    if (showDate) {
        val dateState = rememberDatePickerState(
            initialSelectedDateMillis = existing?.toInstant()?.toEpochMilli(),
        )
        DatePickerDialog(
            onDismissRequest = { showDate = false },
            confirmButton = {
                TextButton(onClick = {
                    val millis = dateState.selectedDateMillis
                    val date = millis?.let { Instant.ofEpochMilli(it).atZone(ZoneOffset.UTC).toLocalDate() }
                    pickedDate = date
                    showDate = false
                    if (cfg.time) showTime = true
                    else commit(date, LocalTime.MIDNIGHT)
                }) { Text("Готово") }
            },
            dismissButton = { TextButton(onClick = { showDate = false }) { Text("Отмена") } },
        ) {
            DatePicker(state = dateState)
        }
    }

    if (showTime) {
        val timeState = rememberTimePickerState(
            initialHour = existing?.hour ?: 12,
            initialMinute = existing?.minute ?: 0,
            is24Hour = true,
        )
        Dialog(onDismissRequest = { showTime = false }) {
            Box(
                modifier = Modifier
                    .clip(RoundedCornerShape(24.dp))
                    .background(MaterialTheme.colorScheme.surfaceContainerHigh)
                    .padding(20.dp),
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("Время", style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(bottom = 12.dp))
                    TimePicker(state = timeState)
                    Row(modifier = Modifier.fillMaxWidth().padding(top = 8.dp), horizontalArrangement = Arrangement.End) {
                        TextButton(onClick = { showTime = false }) { Text("Отмена") }
                        TextButton(onClick = {
                            showTime = false
                            commit(pickedDate ?: existing?.toLocalDate(), LocalTime.of(timeState.hour, timeState.minute))
                        }) { Text("Готово") }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ImageInput(
    field: RegistryFieldDto,
    value: JsonElement?,
    uploading: Boolean,
    onChange: (JsonElement?) -> Unit,
    onUpload: (String, String, ByteArray) -> Unit,
) {
    val context = LocalContext.current
    val serverUrl = LocalServerUrl.current
    val file = value.asUploadedFile()

    var showSource by remember { mutableStateOf(false) }
    var cropUri by remember { mutableStateOf<Uri?>(null) }
    var pendingCameraUri by remember { mutableStateOf<Uri?>(null) }

    val cameraLauncher = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { ok ->
        if (ok) pendingCameraUri?.let { cropUri = it }
    }
    val galleryLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) cropUri = uri
    }

    fun launchCamera() {
        val uri = createCameraImageUri(context)
        pendingCameraUri = uri
        cameraLauncher.launch(uri)
    }
    // CAMERA объявлена в манифесте (для звонков) → ACTION_IMAGE_CAPTURE требует
    // выданного разрешения, иначе SecurityException.
    val cameraPermLauncher = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) launchCamera()
    }
    fun startCamera() {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED) {
            launchCamera()
        } else {
            cameraPermLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        file?.let { f ->
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                AsyncImage(
                    model = serverUrl.trimEnd('/') + "/uploads/" + f.path.trimStart('/'),
                    contentDescription = f.name,
                    modifier = Modifier
                        .size(96.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.surfaceContainerHighest),
                )
                IconButton(onClick = { onChange(null) }) {
                    Icon(Icons.Filled.Close, contentDescription = "Убрать", tint = MaterialTheme.colorScheme.error)
                }
            }
        }
        OutlinedButton(onClick = { showSource = true }, enabled = !uploading) {
            if (uploading) {
                CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Icon(Icons.Filled.PhotoCamera, contentDescription = null, modifier = Modifier.size(18.dp))
            }
            Text(
                if (uploading) "Загрузка…" else if (file != null) "Заменить фото" else "Добавить фото",
                modifier = Modifier.padding(start = 8.dp),
            )
        }
    }

    if (showSource) {
        ModalBottomSheet(onDismissRequest = { showSource = false }) {
            Column {
                ListItem(
                    headlineContent = { Text("Сделать фото") },
                    leadingContent = { Icon(Icons.Filled.PhotoCamera, contentDescription = null) },
                    modifier = Modifier.fillMaxWidth().clickable {
                        showSource = false
                        startCamera()
                    },
                )
                ListItem(
                    headlineContent = { Text("Выбрать из галереи") },
                    leadingContent = { Icon(Icons.Filled.PhotoLibrary, contentDescription = null) },
                    modifier = Modifier.fillMaxWidth().clickable {
                        showSource = false
                        galleryLauncher.launch("image/*")
                    },
                )
            }
        }
    }

    cropUri?.let { uri ->
        ImageCropDialog(
            uri = uri,
            onCancel = { cropUri = null },
            onCropped = { bytes ->
                cropUri = null
                onUpload("photo.jpg", "image/jpeg", bytes)
            },
        )
    }
}

@Composable
private fun FileInput(
    value: JsonElement?,
    uploading: Boolean,
    onChange: (JsonElement?) -> Unit,
    onUpload: (String, String, ByteArray) -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val file = value.asUploadedFile()

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri != null) {
            scope.launch {
                val picked = readPickedFile(context, uri) ?: return@launch
                if (picked.bytes.size > MAX_FILE_BYTES) return@launch
                onUpload(picked.name, picked.mime, picked.bytes)
            }
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        file?.let { f ->
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icon(Icons.Filled.InsertDriveFile, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                Text(f.name.ifBlank { "Файл" }, modifier = Modifier.weight(1f, fill = false))
                IconButton(onClick = { onChange(null) }) {
                    Icon(Icons.Filled.Close, contentDescription = "Убрать", tint = MaterialTheme.colorScheme.error)
                }
            }
        }
        OutlinedButton(onClick = { picker.launch(arrayOf("*/*")) }, enabled = !uploading) {
            if (uploading) {
                CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Icon(Icons.Filled.AttachFile, contentDescription = null, modifier = Modifier.size(18.dp))
            }
            Text(
                if (uploading) "Загрузка…" else if (file != null) "Заменить файл" else "Выбрать файл",
                modifier = Modifier.padding(start = 8.dp),
            )
        }
    }
}

private const val MAX_FILE_BYTES = 25 * 1024 * 1024
