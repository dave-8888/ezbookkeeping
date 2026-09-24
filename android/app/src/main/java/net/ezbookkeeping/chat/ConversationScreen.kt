package net.ezbookkeeping.chat

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable fun ConversationScreen(vm: ChatViewModel, busy: Boolean, close: () -> Unit) {
    val stored by vm.conversations.collectAsStateWithLifecycle()
    val owner by vm.activeOwner.collectAsStateWithLifecycle()
    val selected by vm.activeConversation.collectAsStateWithLifecycle()
    val conversations = stored.filter { it.owner == owner }
    var deleting by rememberSaveable { mutableStateOf<String?>(null) }
    BackHandler(onBack = close)
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = close) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") }
            Text("对话列表", Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
            TextButton(onClick = { vm.newConversation(); close() }, enabled = !busy) { Text("新增对话") }
        }
        LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(conversations, key = { it.id }) { conversation ->
                Surface(shape = MaterialTheme.shapes.large,
                    color = if(conversation.id == selected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surface) {
                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f).clickable(enabled = !busy, role = Role.Button) {
                            vm.switchConversation(conversation.id); close()
                        }.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(conversation.title, maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.titleMedium)
                            val date = Instant.ofEpochMilli(conversation.created).atZone(ZoneId.systemDefault())
                                .format(DateTimeFormatter.ofPattern("M月d日 HH:mm"))
                            Text(if(conversation.id == selected) "当前对话 · $date" else date,
                                style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        IconButton(onClick = { deleting = conversation.id }, enabled = !busy) {
                            Icon(Icons.Default.Delete, "删除对话：${conversation.title}", tint = MaterialTheme.colorScheme.error)
                        }
                    }
                }
            }
        }
    }
    conversations.firstOrNull { it.id == deleting }?.let { conversation ->
        AlertDialog(onDismissRequest = { deleting = null }, title = { Text("删除整个对话？") },
            text = { Text("将删除“${conversation.title}”中的全部本机聊天记录，无法恢复。已保存的账目不会删除。") },
            confirmButton = {
                TextButton(onClick = { vm.deleteConversation(conversation.id); deleting = null }, enabled = !busy) {
                    Text("删除对话", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = { TextButton(onClick = { deleting = null }) { Text("取消") } })
    }
}
