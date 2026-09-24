package net.ezbookkeeping.chat

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable fun ChatNavigation(enabled: Boolean, settings: () -> Unit, conversations: () -> Unit = {}, newConversation: () -> Unit = {}) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        IconButton(onClick = conversations, enabled = enabled) {
            Icon(Icons.AutoMirrored.Filled.List, contentDescription = "对话列表")
        }
        Text("对话记账", modifier = Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
        IconButton(onClick = newConversation, enabled = enabled) {
            Icon(Icons.Default.Add, contentDescription = "新增对话")
        }
        IconButton(onClick = settings, enabled = enabled, modifier = Modifier.size(48.dp)) {
            Icon(Icons.Default.Settings, contentDescription = "设置")
        }
    }
}

@Composable fun ChatComposer(input: String, busy: Boolean, change: (String) -> Unit, send: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.Bottom) {
        TextField(input, change, placeholder = { Text("午饭 30 元") }, modifier = Modifier.weight(1f),
            maxLines = 4, enabled = !busy, shape = RoundedCornerShape(24.dp),
            colors = TextFieldDefaults.colors(
                focusedContainerColor = MaterialTheme.colorScheme.surface,
                unfocusedContainerColor = MaterialTheme.colorScheme.surface,
                disabledContainerColor = MaterialTheme.colorScheme.surface,
                focusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                unfocusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                disabledIndicatorColor = androidx.compose.ui.graphics.Color.Transparent))
        FilledIconButton(onClick = send, enabled = !busy && input.isNotBlank(), modifier = Modifier.size(56.dp)) {
            Icon(Icons.AutoMirrored.Filled.Send, contentDescription = "发送")
        }
    }
}

fun messageDay(message: ChatMessage): LocalDate = Instant.ofEpochMilli(message.created).atZone(ZoneId.systemDefault()).toLocalDate()

@Composable fun MessageDate(message: ChatMessage) {
    val date = messageDay(message)
    Text(if(date == LocalDate.now()) "今天" else date.format(DateTimeFormatter.ofPattern("yyyy年M月d日")),
        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
}

@Composable fun MessageItem(message: ChatMessage, busy: Boolean, edit: () -> Unit, verify: () -> Unit) {
    val finance = remember(message.draft) { ServerConversationClient.presentation(message.draft) }
    var expanded by rememberSaveable(message.id, finance?.optBoolean("expanded")) { mutableStateOf(finance?.optBoolean("expanded") ?: false) }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Surface(Modifier.align(Alignment.End).widthIn(max = 300.dp), color = MaterialTheme.colorScheme.primaryContainer,
            contentColor = MaterialTheme.colorScheme.onPrimaryContainer, shape = RoundedCornerShape(20.dp)) {
            Text(message.text, Modifier.padding(horizontal = 16.dp, vertical = 12.dp))
        }
        // Keep the immutable saved summary; names in the catalog may have changed.
        val lines = message.reply.lines()
        val amountLine = lines.getOrNull(1).orEmpty().split(' ')
        val structured = message.state == "saved" && lines.firstOrNull() == "已记账" && lines.size >= 4 && amountLine.size >= 2
        if(structured) {
            val currency = amountLine.getOrNull(2).orEmpty()
            val value = amountLine[1].toBigDecimalOrNull()
            val cashFlow = if(amountLine[0] == "支出") value?.negate() else value
            val sign = when(cashFlow?.signum()) { 1 -> "+"; -1 -> "−"; else -> "" }
            val magnitude = value?.abs()?.toPlainString() ?: amountLine[1]
            val amount = sign + when(currency) { "CNY" -> "¥$magnitude"; else -> "$magnitude $currency" }.trim()
            Surface(onClick = { expanded = !expanded }, modifier = Modifier.fillMaxWidth().semantics {
                stateDescription = if(expanded) "交易详情已展开" else "交易详情已收起"
            }, shape = RoundedCornerShape(18.dp), color = MaterialTheme.colorScheme.surface) {
                Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text(amount, Modifier.weight(1f), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                        Icon(Icons.Default.CheckCircle, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(16.dp))
                        Text("已记账", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Text(lines[2], style = MaterialTheme.typography.bodyMedium)
                    Text(if(expanded) "收起详情" else "查看详情", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                    if(expanded) {
                        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                        Text("${amountLine[0]} · ${lines[3]}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        if(lines.size > 4) Text(lines.drop(4).joinToString("\n"), style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }
        } else Surface(Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp), color = MaterialTheme.colorScheme.surface) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(message.reply)
                val verified = finance?.optString("verified").orEmpty()
                if(verified.isNotBlank()) {
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    Text(verified, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                val details = finance?.optString("details").orEmpty()
                if(details.isNotBlank()) {
                    TextButton(onClick = { expanded = !expanded }) { Text(if(expanded) "收起详情" else "查看明细") }
                    if(expanded) {
                        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                        Text(details, style = MaterialTheme.typography.bodyMedium)
                    }
                }
                if(message.state in listOf("recognizing", "saving", "updating", "agent_running", "agent_writing")) LinearProgressIndicator(Modifier.fillMaxWidth())
                if(message.state == "server_pending") TextButton(onClick = edit, enabled = !busy) { Text("查看处理结果") }
                if(message.state == "draft") TextButton(onClick = edit, enabled = !busy) { Text("补全交易") }
                if(message.state == "unknown") TextButton(onClick = verify, enabled = !busy) { Text("如何核实") }
            }
        }
    }
}

@Composable fun ChatMessageList(messages: List<ChatMessage>, busy: Boolean, edit: (ChatMessage) -> Unit, verify: () -> Unit, modifier: Modifier = Modifier) {
    val listState = rememberLazyListState()
    LaunchedEffect(messages.size, messages.lastOrNull()?.state) {
        if(messages.isNotEmpty()) listState.animateScrollToItem(messages.lastIndex)
    }
    LazyColumn(modifier.fillMaxWidth(), state = listState, contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(20.dp)) {
        if(messages.isEmpty()) item {
            Column(Modifier.fillMaxWidth().padding(vertical = 48.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                Text("今天记点什么？", style = MaterialTheme.typography.headlineSmall)
                Spacer(Modifier.height(12.dp))
                Text("说一句，信息完整即自动记账", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        itemsIndexed(messages, key = { _, message -> message.id }) { index, message ->
            if(index == 0 || messageDay(messages[index - 1]) != messageDay(message)) MessageDate(message)
            MessageItem(message, busy, { edit(message) }, verify)
        }
    }
}
