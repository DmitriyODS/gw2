package com.kodass.groovework.ui.chats

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.PushPin
import androidx.compose.material3.Badge
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.ConversationItemDto
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.formatChatStamp
import com.kodass.groovework.ui.common.rememberIsScrollingUp

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatsScreen(container: AppContainer, onOpenChat: (Long) -> Unit) {
    val viewModel: ChatsViewModel = viewModel {
        ChatsViewModel(container.messengerRepo, container.authApi, container.json)
    }
    val conversations by container.messengerRepo.conversations.collectAsStateWithLifecycle()
    val online by container.messengerRepo.onlineUsers.collectAsStateWithLifecycle()
    var showNewChat by remember { mutableStateOf(false) }

    val scrollBehavior = TopAppBarDefaults.pinnedScrollBehavior()
    val listState = rememberLazyListState()
    val fabVisible = listState.rememberIsScrollingUp()

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        topBar = {
            TopAppBar(
                title = { Text("Чаты") },
                scrollBehavior = scrollBehavior,
            )
        },
        floatingActionButton = {
            AnimatedVisibility(
                visible = fabVisible,
                enter = scaleIn() + fadeIn(),
                exit = scaleOut() + fadeOut(),
            ) {
                FloatingActionButton(onClick = {
                    showNewChat = true
                    viewModel.loadDirectory()
                }) {
                    Icon(Icons.Filled.Add, contentDescription = "Новый чат")
                }
            }
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = viewModel.refreshing,
            onRefresh = { viewModel.pullRefresh() },
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            when {
                viewModel.loading && conversations.isEmpty() -> CenteredLoading()
                viewModel.error != null && conversations.isEmpty() -> LazyColumn(modifier = Modifier.fillMaxSize()) {
                    item {
                        ErrorState(
                            viewModel.error ?: "",
                            onRetry = { viewModel.refresh() },
                            modifier = Modifier.fillParentMaxSize(),
                        )
                    }
                }
                conversations.isEmpty() -> LazyColumn(modifier = Modifier.fillMaxSize()) {
                    item {
                        EmptyState(
                            "Пока нет диалогов",
                            "Начните переписку с коллегой по кнопке «+»",
                            modifier = Modifier.fillParentMaxSize(),
                        )
                    }
                }
                else -> LazyColumn(state = listState, modifier = Modifier.fillMaxSize()) {
                    items(conversations, key = { it.id }) { conversation ->
                        ConversationRow(
                            conversation = conversation,
                            isOnline = conversation.otherUser?.id in online,
                            onClick = { onOpenChat(conversation.id) },
                        )
                    }
                }
            }
        }
    }

    if (showNewChat) {
        NewChatSheet(
            viewModel = viewModel,
            onDismiss = { showNewChat = false },
            onOpenChat = { id ->
                showNewChat = false
                onOpenChat(id)
            },
        )
    }
}

@Composable
private fun ConversationRow(
    conversation: ConversationItemDto,
    isOnline: Boolean,
    onClick: () -> Unit,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 10.dp),
    ) {
        Box {
            UserAvatar(
                userId = conversation.otherUser?.id,
                name = conversationTitle(conversation),
                avatarPath = conversation.otherUser?.avatarPath,
                size = 52.dp,
            )
            if (isOnline && conversation.otherUser != null) {
                Box(
                    modifier = Modifier
                        .size(14.dp)
                        .align(Alignment.BottomEnd)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.surface)
                        .padding(2.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primary),
                )
            }
        }
        Column(
            modifier = Modifier
                .weight(1f)
                .padding(start = 12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = conversationTitle(conversation),
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                if (conversation.isPinned) {
                    Icon(
                        Icons.Filled.PushPin,
                        contentDescription = "Закреплён",
                        tint = MaterialTheme.colorScheme.outline,
                        modifier = Modifier.padding(start = 4.dp).size(14.dp),
                    )
                }
            }
            Text(
                text = previewText(conversation.lastMessage),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
        Column(horizontalAlignment = Alignment.End, modifier = Modifier.padding(start = 8.dp)) {
            Text(
                text = formatChatStamp(conversation.lastMessageAt),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.outline,
            )
            if (conversation.unreadCount > 0) {
                Badge(modifier = Modifier.padding(top = 6.dp)) {
                    Text(conversation.unreadCount.toString())
                }
            }
        }
    }
}

internal fun conversationTitle(conversation: ConversationItemDto): String = when {
    conversation.isPetChat -> conversation.petName ?: "Питомец"
    conversation.isDevChat -> "Техподдержка"
    else -> conversation.otherUser?.fio?.takeIf { it.isNotBlank() } ?: "Диалог"
}

internal fun previewText(message: MessageDto?): String {
    if (message == null) return "Нет сообщений"
    message.text?.takeIf { it.isNotBlank() }?.let { return it }
    return when {
        message.kind == "call" -> "Звонок"
        message.kind == "task" || message.task != null -> "Задача: ${message.task?.name ?: ""}".trim()
        message.attachments.isNotEmpty() -> "Вложение: ${message.attachments.first().fileName}"
        else -> "Сообщение"
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun NewChatSheet(
    viewModel: ChatsViewModel,
    onDismiss: () -> Unit,
    onOpenChat: (Long) -> Unit,
) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            Text(
                text = "Новый чат",
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(bottom = 12.dp),
            )
            OutlinedTextField(
                value = viewModel.directoryQuery,
                onValueChange = { viewModel.updateDirectoryQuery(it) },
                label = { Text("Поиск коллеги") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            when {
                viewModel.directoryLoading && viewModel.directory.isEmpty() ->
                    Box(modifier = Modifier.fillMaxWidth().height(200.dp)) { CenteredLoading() }
                viewModel.directory.isEmpty() ->
                    Box(modifier = Modifier.fillMaxWidth().height(200.dp)) {
                        EmptyState("Никого не нашлось")
                    }
                else -> LazyColumn(
                    modifier = Modifier.fillMaxWidth().height(420.dp),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    items(viewModel.directory, key = { it.id }) { user ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable(enabled = !viewModel.openingChat) {
                                    viewModel.openChatWith(user.id, onOpenChat)
                                }
                                .padding(vertical = 8.dp),
                        ) {
                            UserAvatar(userId = user.id, name = user.fio, avatarPath = user.avatarPath, size = 44.dp)
                            Column(modifier = Modifier.padding(start = 12.dp)) {
                                Text(user.fio, style = MaterialTheme.typography.bodyLarge)
                                if (!user.post.isNullOrBlank()) {
                                    Text(
                                        user.post,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
