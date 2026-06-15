package com.kodass.groovework.ui.chats

import android.Manifest
import android.provider.OpenableColumns
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.filled.ArrowDownward
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Done
import androidx.compose.material.icons.filled.DoneAll
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.automirrored.filled.InsertDriveFile
import androidx.compose.material.icons.filled.PushPin
import androidx.compose.material.icons.filled.TaskAlt
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FloatingActionButton
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
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import coil3.compose.AsyncImage
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.AttachmentDto
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.data.files.DownloadState
import com.kodass.groovework.data.files.openDownloadedFile
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.ImageViewer
import com.kodass.groovework.ui.common.LocalServerUrl
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.UserInfoSheet
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
fun ChatScreen(
    container: AppContainer,
    conversationId: Long,
    onBack: () -> Unit,
    onOpenTask: (Long) -> Unit,
) {
    val viewModel: ChatViewModel = viewModel(key = "chat-$conversationId") {
        ChatViewModel(container.messengerRepo, container.sessionManager, container.json, conversationId)
    }
    val conversations by container.messengerRepo.conversations.collectAsStateWithLifecycle()
    val online by container.messengerRepo.onlineUsers.collectAsStateWithLifecycle()
    val typingConversations by container.messengerRepo.typingConversations.collectAsStateWithLifecycle()
    val conversation = conversations.firstOrNull { it.id == conversationId }
    val peer = conversation?.otherUser
    val isPeerTyping = conversationId in typingConversations

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

    // Возврат/вход в живой звонок по тапу на плашке в чате.
    val callPhase by container.callManager.phase.collectAsStateWithLifecycle()
    val currentCallId = when (val p = callPhase) {
        is com.kodass.groovework.data.calls.CallPhase.Active -> p.call.id
        is com.kodass.groovework.data.calls.CallPhase.Outgoing -> p.call.id
        is com.kodass.groovework.data.calls.CallPhase.Incoming -> p.call.id
        else -> null
    }
    var pendingJoinCall by remember { mutableStateOf<com.kodass.groovework.data.dto.CallInfoDto?>(null) }
    val joinCallPermLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        val call = pendingJoinCall ?: return@rememberLauncherForActivityResult
        pendingJoinCall = null
        val micOk = result[Manifest.permission.RECORD_AUDIO] == true
        val camOk = !call.isVideo || result[Manifest.permission.CAMERA] == true
        if (micOk && camOk) container.callManager.returnOrJoinCall(call.id, call.isVideo)
    }
    val returnToCall: (com.kodass.groovework.data.dto.CallInfoDto) -> Unit = { call ->
        if (container.callManager.currentCall?.id == call.id) {
            container.callManager.showCallUi()
        } else {
            pendingJoinCall = call
            joinCallPermLauncher.launch(
                if (call.isVideo) arrayOf(Manifest.permission.RECORD_AUDIO, Manifest.permission.CAMERA)
                else arrayOf(Manifest.permission.RECORD_AUDIO)
            )
        }
    }

    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    val inputFocus = remember { FocusRequester() }
    val keyboard = LocalSoftwareKeyboardController.current

    // Подтверждение деструктива (#5) и пикер задачи (#8) — на уровне экрана.
    var confirm by remember { mutableStateOf<ConfirmSpec?>(null) }
    var showTaskPicker by remember { mutableStateOf(false) }
    // Просмотр картинки из чата внутри приложения (зум + скачивание).
    var imageViewer by remember { mutableStateOf<AttachmentDto?>(null) }
    // Карточка собеседника (тап по шапке чата).
    var showPeerInfo by remember { mutableStateOf(false) }
    confirm?.let { spec -> ConfirmDialog(spec, onDismiss = { confirm = null }) }
    imageViewer?.let { attachment ->
        ImageViewer(container = container, attachment = attachment, onDismiss = { imageViewer = null })
    }
    if (showTaskPicker) {
        TaskPickerSheet(
            container = container,
            onDismiss = { showTaskPicker = false },
            onPick = { viewModel.attachedTask = it },
        )
    }

    // Подсветка сообщения, к которому перешли по ответу/закреплению.
    var highlightedId by remember { mutableStateOf<Long?>(null) }
    LaunchedEffect(highlightedId) {
        if (highlightedId != null) {
            kotlinx.coroutines.delay(1600)
            highlightedId = null
        }
    }

    // Переход к сообщению (тап по ответу или по баннеру закрепления): догружаем
    // историю, скроллим, подсвечиваем.
    val scrollToMessage: (Long) -> Unit = { messageId ->
        scope.launch {
            val index = viewModel.ensureLoaded(messageId)
            if (index != null) {
                listState.animateScrollToItem(index)
                highlightedId = messageId
            }
        }
    }

    // Фокус на поле ввода при выборе ответа (свайп/меню) — сразу можно печатать.
    LaunchedEffect(viewModel.replyTo?.id) {
        if (viewModel.replyTo != null) {
            inputFocus.requestFocus()
            keyboard?.show()
        }
    }

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
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.clickable(enabled = peer != null) { showPeerInfo = true },
                    ) {
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
                                val peerOnline = peer.id in online
                                Text(
                                    text = when {
                                        isPeerTyping -> "печатает…"
                                        peerOnline -> "в сети"
                                        else -> formatLastSeen(peer.lastSeenAt)
                                    },
                                    style = MaterialTheme.typography.bodySmall,
                                    color = if (isPeerTyping || peerOnline) MaterialTheme.colorScheme.primary
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
            MessageInputBar(viewModel, inputFocus, onAttachTask = { showTaskPicker = true })
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            if (viewModel.pinnedMessages.isNotEmpty()) {
                PinnedBanner(
                    pinned = viewModel.pinnedMessages,
                    onOpen = { scrollToMessage(it.id) },
                    onUnpin = { message ->
                        confirm = ConfirmSpec(
                            title = "Открепить сообщение",
                            text = "Открепить это сообщение для всех?",
                            confirmLabel = "Открепить",
                            destructive = false,
                            action = { viewModel.togglePin(message) },
                        )
                    },
                )
            }
            Box(modifier = Modifier.fillMaxSize().weight(1f)) {
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
                                        container = container,
                                        message = message,
                                        mine = message.senderId != null && message.senderId == viewModel.myUserId,
                                        highlighted = highlightedId == message.id,
                                        onReply = { viewModel.replyTo = message },
                                        onReplyClick = { replyId -> scrollToMessage(replyId) },
                                        onForward = { viewModel.forwardTarget = message },
                                        onTogglePin = { viewModel.togglePin(message) },
                                        onDelete = { forAll -> viewModel.deleteMessage(message, forAll) },
                                        onOpenTask = { message.task?.let { onOpenTask(it.id) } },
                                        onOpenImage = { imageViewer = it },
                                        onConfirm = { spec -> confirm = spec },
                                        currentCallId = currentCallId,
                                        onReturnToCall = returnToCall,
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

                // Кнопка «к новым сообщениям» — видна, когда пролистали вверх.
                val showScrollDown by remember {
                    derivedStateOf { listState.firstVisibleItemIndex > 1 }
                }
                val fabScale by animateFloatAsState(
                    targetValue = if (showScrollDown) 1f else 0f,
                    label = "scrollDownFab",
                )
                if (fabScale > 0.01f) {
                    FloatingActionButton(
                        onClick = { scope.launch { listState.animateScrollToItem(0) } },
                        containerColor = MaterialTheme.colorScheme.secondaryContainer,
                        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
                        modifier = Modifier
                            .align(Alignment.BottomEnd)
                            .padding(16.dp)
                            .size(44.dp)
                            .graphicsLayer {
                                scaleX = fabScale
                                scaleY = fabScale
                                alpha = fabScale
                            },
                    ) {
                        Icon(Icons.Filled.ArrowDownward, contentDescription = "К новым сообщениям")
                    }
                }
            }
        }
    }

    if (showPeerInfo && peer != null) {
        UserInfoSheet(
            container = container,
            userId = peer.id,
            fallback = peer,
            online = peer.id in online,
            canCall = canCall,
            onAudioCall = { showPeerInfo = false; requestCall(false) },
            onVideoCall = { showPeerInfo = false; requestCall(true) },
            onDismiss = { showPeerInfo = false },
        )
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

// Баннер закреплённых сообщений: тап открывает текущее и листает к следующему,
// крестик — открепляет.
@Composable
private fun PinnedBanner(
    pinned: List<MessageDto>,
    onOpen: (MessageDto) -> Unit,
    onUnpin: (MessageDto) -> Unit,
) {
    var index by remember(pinned.size) { mutableStateOf(0) }
    val current = pinned.getOrNull(index.coerceIn(0, pinned.size - 1)) ?: return
    Surface(color = MaterialTheme.colorScheme.surfaceContainer, tonalElevation = 2.dp) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .clickable {
                    onOpen(current)
                    if (pinned.size > 1) index = (index + 1) % pinned.size
                }
                .padding(start = 12.dp, end = 4.dp, top = 6.dp, bottom = 6.dp),
        ) {
            Icon(
                Icons.Filled.PushPin,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(18.dp),
            )
            Column(modifier = Modifier.weight(1f).padding(start = 10.dp)) {
                Text(
                    text = if (pinned.size > 1) "Закреплённые · ${index + 1}/${pinned.size}"
                    else "Закреплённое сообщение",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
                Text(
                    text = messageSummary(current),
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            IconButton(onClick = { onUnpin(current) }) {
                Icon(
                    Icons.Filled.Close,
                    contentDescription = "Открепить",
                    modifier = Modifier.size(18.dp),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

private fun messageSummary(message: MessageDto): String =
    message.text?.takeIf { it.isNotBlank() }
        ?: when {
            message.kind == "call" -> "Звонок"
            message.task != null -> "Задача: ${message.task.name}"
            message.attachments.isNotEmpty() -> message.attachments.first().fileName
            else -> "Сообщение"
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
    container: AppContainer,
    message: MessageDto,
    mine: Boolean,
    highlighted: Boolean,
    onReply: () -> Unit,
    onReplyClick: (Long) -> Unit,
    onForward: () -> Unit,
    onTogglePin: () -> Unit,
    onDelete: (forAll: Boolean) -> Unit,
    onOpenTask: () -> Unit,
    onOpenImage: (AttachmentDto) -> Unit,
    onConfirm: (ConfirmSpec) -> Unit,
    currentCallId: Long?,
    onReturnToCall: (com.kodass.groovework.data.dto.CallInfoDto) -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    val pinned = message.pinnedAt != null
    val bubbleColor = if (mine) MaterialTheme.colorScheme.primaryContainer
    else MaterialTheme.colorScheme.surfaceContainerHigh
    val shape = RoundedCornerShape(
        topStart = 18.dp,
        topEnd = 18.dp,
        bottomStart = if (mine) 18.dp else 6.dp,
        bottomEnd = if (mine) 6.dp else 18.dp,
    )
    val highlightColor by animateColorAsState(
        targetValue = if (highlighted) MaterialTheme.colorScheme.primary.copy(alpha = 0.12f)
        else androidx.compose.ui.graphics.Color.Transparent,
        label = "highlight",
    )

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(highlightColor)
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
                            modifier = Modifier
                                .padding(bottom = 6.dp)
                                .clickable { onReplyClick(reply.id) },
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
                        CallCard(
                            message = message,
                            isCurrent = currentCallId != null && message.call?.id == currentCallId,
                            onReturnToCall = onReturnToCall,
                        )
                    }
                    message.task?.let { TaskCard(task = it, onClick = onOpenTask) }
                    message.attachments.forEach { attachment ->
                        AttachmentView(container = container, attachment = attachment, onOpenImage = onOpenImage)
                    }
                    message.text?.takeIf { it.isNotBlank() }?.let { text ->
                        Text(text = text, style = MaterialTheme.typography.bodyLarge)
                    }
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.align(Alignment.End).padding(top = 2.dp),
                    ) {
                        if (pinned) {
                            Icon(
                                Icons.Filled.PushPin,
                                contentDescription = "Закреплено",
                                tint = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.padding(end = 4.dp).size(13.dp),
                            )
                        }
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
                    text = { Text(if (pinned) "Открепить" else "Закрепить") },
                    onClick = {
                        menuOpen = false
                        // Закрепление не деструктивно; открепление подтверждаем (#5).
                        if (pinned) {
                            onConfirm(
                                ConfirmSpec(
                                    title = "Открепить сообщение",
                                    text = "Открепить это сообщение для всех?",
                                    confirmLabel = "Открепить",
                                    destructive = false,
                                    action = onTogglePin,
                                )
                            )
                        } else {
                            onTogglePin()
                        }
                    },
                )
                DropdownMenuItem(
                    text = { Text("Удалить у себя") },
                    onClick = {
                        menuOpen = false
                        onConfirm(
                            ConfirmSpec(
                                title = "Удалить сообщение",
                                text = "Удалить это сообщение у себя?",
                                action = { onDelete(false) },
                            )
                        )
                    },
                )
                if (mine) {
                    DropdownMenuItem(
                        text = { Text("Удалить у всех") },
                        onClick = {
                            menuOpen = false
                            onConfirm(
                                ConfirmSpec(
                                    title = "Удалить у всех",
                                    text = "Удалить это сообщение у обоих собеседников?",
                                    action = { onDelete(true) },
                                )
                            )
                        },
                    )
                }
            }
        }
    }
}

@Composable
private fun CallCard(
    message: MessageDto,
    isCurrent: Boolean,
    onReturnToCall: (com.kodass.groovework.data.dto.CallInfoDto) -> Unit,
) {
    val call = message.call
    val live = call?.isLive == true
    val video = call?.isVideo == true
    val title = when {
        live -> if (video) "Видеозвонок · идёт сейчас" else "Звонок · идёт сейчас"
        else -> if (video) "Видеозвонок" else "Звонок"
    }
    val rowModifier = if (live && call != null) {
        Modifier
            .clip(RoundedCornerShape(12.dp))
            .clickable { onReturnToCall(call) }
            .background(MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f))
            .padding(horizontal = 10.dp, vertical = 8.dp)
    } else {
        Modifier.padding(vertical = 4.dp)
    }
    Row(verticalAlignment = Alignment.CenterVertically, modifier = rowModifier) {
        Icon(
            if (video) Icons.Filled.Videocam else Icons.Filled.Call,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
        )
        Column(modifier = Modifier.padding(start = 8.dp).weight(1f, fill = false)) {
            Text(title, style = MaterialTheme.typography.bodyMedium)
            when {
                live -> Text(
                    text = if (isCurrent) "Нажмите, чтобы вернуться" else "Нажмите, чтобы присоединиться",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.primary,
                )
                call?.durationSec != null -> Text(
                    text = "${call.durationSec / 60} мин ${call.durationSec % 60} с",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

// Карточка прикреплённой задачи в пузыре: тап открывает карточку задачи (#8).
@Composable
private fun TaskCard(task: com.kodass.groovework.data.dto.TaskCardDto, onClick: () -> Unit) {
    val accent = com.kodass.groovework.ui.tasks.taskAccentColor(task.color)
        ?: MaterialTheme.colorScheme.primary
    Surface(
        color = MaterialTheme.colorScheme.surface.copy(alpha = 0.5f),
        shape = RoundedCornerShape(10.dp),
        modifier = Modifier
            .padding(vertical = 4.dp)
            .clickable(onClick = onClick),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
        ) {
            Icon(Icons.Filled.TaskAlt, contentDescription = null, tint = accent)
            Column(modifier = Modifier.padding(start = 8.dp)) {
                Text(
                    text = task.name.ifBlank { "Задача" },
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = "Открыть задачу",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
        }
    }
}

@Composable
private fun AttachmentView(
    container: AppContainer,
    attachment: AttachmentDto,
    onOpenImage: (AttachmentDto) -> Unit,
) {
    val serverUrl = LocalServerUrl.current
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
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
                .combinedClickable(onClick = { onOpenImage(attachment) }),
        )
        return
    }

    // Файл: тап скачивает внутрь приложения (с прогрессом), повторный тап после
    // загрузки — открывает сохранённый файл.
    var dl by remember(attachment.id) { mutableStateOf<DownloadState>(DownloadState.Idle) }
    fun startDownload() {
        if (dl is DownloadState.Running) return
        dl = DownloadState.Running(-1f)
        scope.launch {
            try {
                val uri = container.downloader.download(
                    url = fullUrl,
                    fileName = attachment.fileName,
                    mime = attachment.mimeType,
                    toImages = false,
                ) { p -> dl = DownloadState.Running(p) }
                dl = DownloadState.Done(uri, attachment.mimeType)
                Toast.makeText(context, "Файл сохранён", Toast.LENGTH_SHORT).show()
            } catch (_: Exception) {
                dl = DownloadState.Failed("Ошибка загрузки")
                Toast.makeText(context, "Не удалось скачать", Toast.LENGTH_SHORT).show()
            }
        }
    }

    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .padding(vertical = 4.dp)
            .clip(RoundedCornerShape(10.dp))
            .background(MaterialTheme.colorScheme.surface.copy(alpha = 0.5f))
            .combinedClickable(onClick = {
                when (val s = dl) {
                    is DownloadState.Done -> openDownloadedFile(context, s.uri, s.mime)
                    is DownloadState.Running -> {}
                    else -> startDownload()
                }
            })
            .padding(8.dp),
    ) {
        val state = dl
        if (state is DownloadState.Running) {
            Box(modifier = Modifier.size(24.dp), contentAlignment = Alignment.Center) {
                if (state.progress >= 0f) {
                    CircularProgressIndicator(
                        progress = { state.progress },
                        modifier = Modifier.size(24.dp),
                        strokeWidth = 2.dp,
                    )
                } else {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
                }
            }
        } else {
            Icon(
                if (state is DownloadState.Done) Icons.Filled.Download
                else Icons.AutoMirrored.Filled.InsertDriveFile,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
            )
        }
        Column(modifier = Modifier.padding(start = 8.dp)) {
            Text(
                text = attachment.fileName,
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = when (val s = dl) {
                    is DownloadState.Running ->
                        if (s.progress >= 0f) "Загрузка… ${(s.progress * 100).toInt()}%" else "Загрузка…"
                    is DownloadState.Done -> "Открыть файл"
                    is DownloadState.Failed -> "Ошибка — нажмите ещё раз"
                    DownloadState.Idle -> formatFileSize(attachment.sizeBytes)
                },
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun MessageInputBar(
    viewModel: ChatViewModel,
    focusRequester: FocusRequester,
    onAttachTask: () -> Unit,
) {
    val context = LocalContext.current
    var attachMenuOpen by remember { mutableStateOf(false) }
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
            viewModel.attachedTask?.let { task ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 4.dp, top = 4.dp),
                ) {
                    Icon(
                        Icons.Filled.TaskAlt,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(18.dp),
                    )
                    Text(
                        text = "Задача: ${task.name.ifBlank { "#${task.id}" }}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.primary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f).padding(start = 6.dp),
                    )
                    IconButton(onClick = { viewModel.attachedTask = null }) {
                        Icon(Icons.Filled.Close, contentDescription = "Убрать задачу")
                    }
                }
            }
            Row(
                verticalAlignment = Alignment.Bottom,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 6.dp),
            ) {
                Box {
                    IconButton(onClick = { attachMenuOpen = true }, enabled = !viewModel.uploading) {
                        if (viewModel.uploading) {
                            CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Filled.AttachFile, contentDescription = "Прикрепить")
                        }
                    }
                    DropdownMenu(expanded = attachMenuOpen, onDismissRequest = { attachMenuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text("Файл") },
                            leadingIcon = { Icon(Icons.Filled.AttachFile, contentDescription = null) },
                            onClick = {
                                attachMenuOpen = false
                                filePicker.launch("*/*")
                            },
                        )
                        DropdownMenuItem(
                            text = { Text("Задачу") },
                            leadingIcon = { Icon(Icons.Filled.TaskAlt, contentDescription = null) },
                            onClick = {
                                attachMenuOpen = false
                                onAttachTask()
                            },
                        )
                    }
                }
                TextField(
                    value = viewModel.input,
                    onValueChange = { viewModel.onInputChange(it) },
                    placeholder = { Text("Сообщение…") },
                    maxLines = 5,
                    keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                    colors = TextFieldDefaults.colors(
                        focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        focusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                        unfocusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                    ),
                    shape = RoundedCornerShape(24.dp),
                    modifier = Modifier.weight(1f).focusRequester(focusRequester),
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
