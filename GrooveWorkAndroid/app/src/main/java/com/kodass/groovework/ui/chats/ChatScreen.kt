package com.kodass.groovework.ui.chats

import android.Manifest
import android.content.Intent
import android.provider.OpenableColumns
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Reply
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Done
import androidx.compose.material.icons.filled.DoneAll
import androidx.compose.material.icons.automirrored.filled.InsertDriveFile
import androidx.compose.material.icons.filled.TaskAlt
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.core.net.toUri
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import coil3.compose.AsyncImage
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.AttachmentDto
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.LocalServerUrl
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.formatDayHeader
import com.kodass.groovework.ui.common.formatFileSize
import com.kodass.groovework.ui.common.formatLastSeen
import com.kodass.groovework.ui.common.formatTime
import com.kodass.groovework.ui.common.parseIso
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch
import kotlin.math.roundToInt

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(container: AppContainer, conversationId: Long, onBack: () -> Unit) {
    val viewModel: ChatViewModel = viewModel(key = "chat-$conversationId") {
        ChatViewModel(container.messengerRepo, container.sessionManager, container.json, conversationId)
    }
    val conversations by container.messengerRepo.conversations.collectAsStateWithLifecycle()
    val online by container.messengerRepo.onlineUsers.collectAsStateWithLifecycle()
    val conversation = conversations.firstOrNull { it.id == conversationId }
    val peer = conversation?.otherUser

    // Открытый чат не шлёт уведомления и гасит существующее.
    DisposableEffect(conversationId) {
        container.notificationCenter.activeConversationId.value = conversationId
        container.notifier.cancelMessage(conversationId)
        onDispose {
            if (container.notificationCenter.activeConversationId.value == conversationId) {
                container.notificationCenter.activeConversationId.value = null
            }
        }
    }

    // Старт звонка после выдачи разрешений на микрофон/камеру.
    var pendingVideoCall by remember { mutableStateOf<Boolean?>(null) }
    val callPermissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        val video = pendingVideoCall ?: return@rememberLauncherForActivityResult
        pendingVideoCall = null
        val micOk = result[Manifest.permission.RECORD_AUDIO] == true
        val camOk = !video || result[Manifest.permission.CAMERA] == true
        if (micOk && camOk) {
            peer?.let { container.callManager.startCall(it.id, video) }
        }
    }
    val requestCall: (Boolean) -> Unit = { video ->
        pendingVideoCall = video
        callPermissionLauncher.launch(
            if (video) arrayOf(Manifest.permission.RECORD_AUDIO, Manifest.permission.CAMERA)
            else arrayOf(Manifest.permission.RECORD_AUDIO)
        )
    }
    val canCall = peer != null && conversation.isPetChat.not() && conversation.isDevChat.not()

    val listState = rememberLazyListState()

    // Подгрузка истории при прокрутке к старым сообщениям.
    LaunchedEffect(listState) {
        snapshotFlow {
            val last = listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0
            last >= listState.layoutInfo.totalItemsCount - 5
        }
            .distinctUntilChanged()
            .collect { nearEnd -> if (nearEnd) viewModel.loadMore() }
    }

    // Прилипание к низу при новых сообщениях, если пользователь уже внизу.
    val firstMessageId = viewModel.messages.firstOrNull()?.id
    LaunchedEffect(firstMessageId) {
        if (listState.firstVisibleItemIndex <= 1) {
            listState.animateScrollToItem(0)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        UserAvatar(
                            userId = peer?.id,
                            name = conversation?.let { conversationTitle(it) },
                            avatarPath = peer?.avatarPath,
                            size = 38.dp,
                        )
                        Column(modifier = Modifier.padding(start = 10.dp)) {
                            Text(
                                text = conversation?.let { conversationTitle(it) } ?: "Чат",
                                style = MaterialTheme.typography.titleMedium,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            if (peer != null) {
                                val subtitle = if (peer.id in online) "в сети" else formatLastSeen(peer.lastSeenAt)
                                Text(
                                    text = subtitle,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = if (peer.id in online) MaterialTheme.colorScheme.primary
                                    else MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                },
                actions = {
                    if (canCall) {
                        IconButton(onClick = { requestCall(false) }) {
                            Icon(Icons.Filled.Call, contentDescription = "Позвонить")
                        }
                        IconButton(onClick = { requestCall(true) }) {
                            Icon(Icons.Filled.Videocam, contentDescription = "Видеозвонок")
                        }
                    }
                },
            )
        },
        bottomBar = {
            MessageInputBar(viewModel)
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loading -> CenteredLoading()
                viewModel.error != null ->
                    ErrorState(viewModel.error ?: "", onRetry = { viewModel.loadInitial() })
                else -> LazyColumn(
                    state = listState,
                    reverseLayout = true,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 8.dp),
                ) {
                    itemsIndexed(viewModel.messages, key = { _, m -> m.id }) { index, message ->
                        val older = viewModel.messages.getOrNull(index + 1)
                        val messageDate = parseIso(message.createdAt)?.toLocalDate()
                        val olderDate = older?.let { parseIso(it.createdAt)?.toLocalDate() }
                        Column {
                            if (messageDate != null && messageDate != olderDate) {
                                DayHeader(formatDayHeader(messageDate))
                            }
                            SwipeToReply(onReply = { viewModel.replyTo = message }) {
                                MessageBubble(
                                    message = message,
                                    mine = message.senderId != null && message.senderId == viewModel.myUserId,
                                    onReply = { viewModel.replyTo = message },
                                    onForward = { viewModel.forwardTarget = message },
                                    onDelete = { forAll -> viewModel.deleteMessage(message, forAll) },
                                )
                            }
                        }
                    }
                    if (viewModel.loadingMore) {
                        item {
                            Box(
                                modifier = Modifier.fillMaxWidth().padding(12.dp),
                                contentAlignment = Alignment.Center,
                            ) {
                                CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
                            }
                        }
                    }
                }
            }
        }
    }

    viewModel.forwardTarget?.let { target ->
        ForwardSheet(
            conversations = conversations.filter { !it.isPetChat },
            online = online,
            onDismiss = { viewModel.forwardTarget = null },
            onSelect = { targetConversation ->
                viewModel.forward(target, targetConversation.id) {}
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ForwardSheet(
    conversations: List<com.kodass.groovework.data.dto.ConversationItemDto>,
    online: Set<Long>,
    onDismiss: () -> Unit,
    onSelect: (com.kodass.groovework.data.dto.ConversationItemDto) -> Unit,
) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            Text(
                text = "Переслать в чат",
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(bottom = 8.dp),
            )
            LazyColumn(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(max = 440.dp)
                    .navigationBarsPadding(),
            ) {
                itemsIndexed(conversations, key = { _, c -> c.id }) { _, conversation ->
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .fillMaxWidth()
                            .combinedClickable(onClick = { onSelect(conversation) })
                            .padding(vertical = 8.dp),
                    ) {
                        UserAvatar(
                            userId = conversation.otherUser?.id,
                            name = conversationTitle(conversation),
                            avatarPath = conversation.otherUser?.avatarPath,
                            size = 42.dp,
                        )
                        Column(modifier = Modifier.padding(start = 12.dp)) {
                            Text(conversationTitle(conversation), style = MaterialTheme.typography.bodyLarge)
                            if (conversation.otherUser?.id in online) {
                                Text(
                                    text = "в сети",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.primary,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

// Свайп вправо — ответ, как в каноничных мессенджерах: пузырь тянется за пальцем
// с сопротивлением, слева проявляется круглая иконка ответа, на пороге — хаптика,
// отпускание за порогом вызывает reply, элемент пружинит обратно.
@Composable
private fun SwipeToReply(onReply: () -> Unit, content: @Composable () -> Unit) {
    val offsetX = remember { Animatable(0f) }
    val scope = rememberCoroutineScope()
    val density = LocalDensity.current
    val triggerPx = with(density) { 56.dp.toPx() }
    val maxPx = with(density) { 80.dp.toPx() }
    val haptic = LocalHapticFeedback.current
    var pastTrigger by remember { mutableStateOf(false) }

    Box(modifier = Modifier.fillMaxWidth()) {
        val progress = (offsetX.value / triggerPx).coerceIn(0f, 1f)
        if (progress > 0.02f) {
            Box(
                contentAlignment = Alignment.Center,
                modifier = Modifier
                    .align(Alignment.CenterStart)
                    .padding(start = 16.dp)
                    .size(32.dp)
                    .graphicsLayer {
                        alpha = progress
                        scaleX = 0.5f + 0.5f * progress
                        scaleY = 0.5f + 0.5f * progress
                    }
                    .background(MaterialTheme.colorScheme.secondaryContainer, CircleShape),
            ) {
                Icon(
                    Icons.AutoMirrored.Filled.Reply,
                    contentDescription = "Ответить",
                    tint = MaterialTheme.colorScheme.onSecondaryContainer,
                    modifier = Modifier.size(18.dp),
                )
            }
        }
        Box(
            modifier = Modifier
                .offset { IntOffset(offsetX.value.roundToInt(), 0) }
                .pointerInput(Unit) {
                    detectHorizontalDragGestures(
                        onDragEnd = {
                            if (offsetX.value >= triggerPx) onReply()
                            pastTrigger = false
                            scope.launch {
                                offsetX.animateTo(0f, spring(stiffness = Spring.StiffnessMediumLow))
                            }
                        },
                        onDragCancel = {
                            pastTrigger = false
                            scope.launch { offsetX.animateTo(0f) }
                        },
                    ) { change, dragAmount ->
                        val target = (offsetX.value + dragAmount * 0.7f).coerceIn(0f, maxPx)
                        if (target > 0f) change.consume()
                        scope.launch { offsetX.snapTo(target) }
                        val past = target >= triggerPx
                        if (past && !pastTrigger) {
                            haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                        }
                        pastTrigger = past
                    }
                },
        ) {
            content()
        }
    }
}

@Composable
private fun DayHeader(text: String) {
    Box(modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp), contentAlignment = Alignment.Center) {
        Surface(
            shape = RoundedCornerShape(12.dp),
            color = MaterialTheme.colorScheme.surfaceContainerHigh,
        ) {
            Text(
                text = text,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
            )
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun MessageBubble(
    message: MessageDto,
    mine: Boolean,
    onReply: () -> Unit,
    onForward: () -> Unit,
    onDelete: (forAll: Boolean) -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    val bubbleColor = if (mine) MaterialTheme.colorScheme.primaryContainer
    else MaterialTheme.colorScheme.surfaceContainerHigh
    val shape = RoundedCornerShape(
        topStart = 18.dp,
        topEnd = 18.dp,
        bottomStart = if (mine) 18.dp else 6.dp,
        bottomEnd = if (mine) 6.dp else 18.dp,
    )

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 3.dp),
        horizontalArrangement = if (mine) Arrangement.End else Arrangement.Start,
    ) {
        Box {
            Surface(
                color = bubbleColor,
                shape = shape,
                modifier = Modifier
                    .widthIn(max = 300.dp)
                    .combinedClickable(
                        onClick = {},
                        onLongClick = { menuOpen = true },
                    ),
            ) {
                Column(modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp)) {
                    message.forwardedFrom?.fio?.let { fio ->
                        Text(
                            text = "Переслано от $fio",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                    message.replyTo?.let { reply ->
                        Surface(
                            color = MaterialTheme.colorScheme.surface.copy(alpha = 0.5f),
                            shape = RoundedCornerShape(8.dp),
                            modifier = Modifier.padding(bottom = 6.dp),
                        ) {
                            Column(modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)) {
                                Text(
                                    text = reply.senderFio ?: "Сообщение",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.primary,
                                )
                                Text(
                                    text = reply.text?.takeIf { it.isNotBlank() }
                                        ?: if (reply.hasAttachments) "Вложение" else "Сообщение",
                                    style = MaterialTheme.typography.bodySmall,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                        }
                    }
                    if (message.kind == "call") {
                        CallCard(message)
                    }
                    message.task?.let { TaskCard(name = it.name) }
                    message.attachments.forEach { attachment ->
                        AttachmentView(attachment)
                    }
                    message.text?.takeIf { it.isNotBlank() }?.let { text ->
                        Text(text = text, style = MaterialTheme.typography.bodyLarge)
                    }
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.align(Alignment.End).padding(top = 2.dp),
                    ) {
                        Text(
                            text = formatTime(message.createdAt),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        if (mine) {
                            Icon(
                                imageVector = if (message.readAt != null) Icons.Filled.DoneAll else Icons.Filled.Done,
                                contentDescription = if (message.readAt != null) "Прочитано" else "Отправлено",
                                tint = if (message.readAt != null) MaterialTheme.colorScheme.primary
                                else MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(start = 4.dp).size(14.dp),
                            )
                        }
                    }
                }
            }
            DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                DropdownMenuItem(
                    text = { Text("Ответить") },
                    onClick = {
                        menuOpen = false
                        onReply()
                    },
                )
                DropdownMenuItem(
                    text = { Text("Переслать") },
                    onClick = {
                        menuOpen = false
                        onForward()
                    },
                )
                DropdownMenuItem(
                    text = { Text("Удалить у себя") },
                    onClick = {
                        menuOpen = false
                        onDelete(false)
                    },
                )
                if (mine) {
                    DropdownMenuItem(
                        text = { Text("Удалить у всех") },
                        onClick = {
                            menuOpen = false
                            onDelete(true)
                        },
                    )
                }
            }
        }
    }
}

@Composable
private fun CallCard(message: MessageDto) {
    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 4.dp)) {
        Icon(Icons.Filled.Call, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
        Column(modifier = Modifier.padding(start = 8.dp)) {
            Text("Звонок", style = MaterialTheme.typography.bodyMedium)
            message.call?.durationSec?.let { seconds ->
                Text(
                    text = "${seconds / 60} мин ${seconds % 60} с",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun TaskCard(name: String) {
    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 4.dp)) {
        Icon(Icons.Filled.TaskAlt, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
        Text(
            text = name.ifBlank { "Задача" },
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.padding(start = 8.dp),
        )
    }
}

@Composable
private fun AttachmentView(attachment: AttachmentDto) {
    val serverUrl = LocalServerUrl.current
    val context = LocalContext.current
    val fullUrl = serverUrl.trimEnd('/') + "/" + attachment.url.trimStart('/')
    if (attachment.mimeType.startsWith("image/")) {
        AsyncImage(
            model = fullUrl,
            contentDescription = attachment.fileName,
            contentScale = ContentScale.Crop,
            modifier = Modifier
                .padding(vertical = 4.dp)
                .widthIn(max = 260.dp)
                .heightIn(max = 280.dp)
                .clip(RoundedCornerShape(12.dp))
                .combinedClickable(onClick = {
                    context.startActivity(Intent(Intent.ACTION_VIEW, fullUrl.toUri()))
                }),
        )
    } else {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .padding(vertical = 4.dp)
                .clip(RoundedCornerShape(10.dp))
                .background(MaterialTheme.colorScheme.surface.copy(alpha = 0.5f))
                .combinedClickable(onClick = {
                    context.startActivity(Intent(Intent.ACTION_VIEW, fullUrl.toUri()))
                })
                .padding(8.dp),
        ) {
            Icon(Icons.AutoMirrored.Filled.InsertDriveFile, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
            Column(modifier = Modifier.padding(start = 8.dp)) {
                Text(
                    text = attachment.fileName,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = formatFileSize(attachment.sizeBytes),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun MessageInputBar(viewModel: ChatViewModel) {
    val context = LocalContext.current
    val filePicker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        val resolver = context.contentResolver
        val name = resolver.query(uri, null, null, null, null)?.use { cursor ->
            val index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
            if (index >= 0 && cursor.moveToFirst()) cursor.getString(index) else null
        } ?: "file"
        val mime = resolver.getType(uri) ?: "application/octet-stream"
        val bytes = resolver.openInputStream(uri)?.use { it.readBytes() }
        if (bytes != null) viewModel.attachFile(name, mime, bytes)
    }

    Surface(color = MaterialTheme.colorScheme.surfaceContainer) {
        Column(modifier = Modifier.navigationBarsPadding().imePadding()) {
            viewModel.actionError?.let { error ->
                Text(
                    text = error,
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                )
            }
            viewModel.replyTo?.let { reply ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 4.dp, top = 4.dp),
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "Ответ: ${reply.text?.take(80) ?: "вложение"}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    IconButton(onClick = { viewModel.replyTo = null }) {
                        Icon(Icons.Filled.Close, contentDescription = "Отменить ответ")
                    }
                }
            }
            viewModel.pendingAttachment?.let { attachment ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 4.dp, top = 4.dp),
                ) {
                    Icon(
                        Icons.Filled.AttachFile,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(18.dp),
                    )
                    Text(
                        text = "${attachment.fileName} (${formatFileSize(attachment.sizeBytes)})",
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f).padding(start = 6.dp),
                    )
                    IconButton(onClick = { viewModel.clearAttachment() }) {
                        Icon(Icons.Filled.Close, contentDescription = "Убрать вложение")
                    }
                }
            }
            Row(
                verticalAlignment = Alignment.Bottom,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 6.dp),
            ) {
                IconButton(onClick = { filePicker.launch("*/*") }, enabled = !viewModel.uploading) {
                    if (viewModel.uploading) {
                        CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                    } else {
                        Icon(Icons.Filled.AttachFile, contentDescription = "Прикрепить файл")
                    }
                }
                TextField(
                    value = viewModel.input,
                    onValueChange = { viewModel.input = it },
                    placeholder = { Text("Сообщение…") },
                    maxLines = 5,
                    colors = TextFieldDefaults.colors(
                        focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        focusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                        unfocusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                    ),
                    shape = RoundedCornerShape(24.dp),
                    modifier = Modifier.weight(1f),
                )
                FilledIconButton(
                    onClick = { viewModel.send() },
                    enabled = viewModel.canSend,
                    modifier = Modifier.padding(start = 4.dp),
                ) {
                    Icon(Icons.AutoMirrored.Filled.Send, contentDescription = "Отправить")
                }
            }
        }
    }
}
