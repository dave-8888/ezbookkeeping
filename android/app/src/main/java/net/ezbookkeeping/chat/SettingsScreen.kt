package net.ezbookkeeping.chat

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp

@Composable fun SettingsScreen(vm: ChatViewModel, busy: Boolean, openAI: () -> Unit, close: () -> Unit) {
    var history by rememberSaveable { mutableStateOf(false) }
    var clear by rememberSaveable { mutableStateOf(false) }
    BackHandler(onBack = close)
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = close) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") }
            Text("设置", style = MaterialTheme.typography.headlineSmall)
        }
        SettingsGroup("账号与服务") {
            SettingsRow("管理登录记录", busy = busy) { history = true }
            SettingsRow("修改服务地址", "修改后需要重新登录", busy) { vm.logout(editService = true) }
            SettingsRow("重新连接", vm.address, busy) { vm.reconnect() }
            SettingsRow("退出登录", busy = busy) { vm.logout() }
        }
        SettingsGroup("记账设置") {
            SettingsRow("AI 设置", "配置用于识别账单的模型", busy) { openAI() }
        }
        SettingsGroup("本机数据") {
            SettingsRow("清空本机聊天记录", "已保存的账单不会删除", busy, destructive = true) { clear = true }
        }
    }
    if(history) LoginHistoryDialog(vm, onDismiss = { history = false })
    if(clear) AlertDialog(onDismissRequest = { clear = false }, title = { Text("清空聊天记录？") },
        text = { Text("只清空当前账号的本机聊天，已保存的账单不会删除。") },
        confirmButton = { TextButton(onClick = { vm.clearMessages(); clear = false }, enabled = !busy) { Text("清空") } },
        dismissButton = { TextButton(onClick = { clear = false }) { Text("取消") } })
}

@Composable private fun SettingsGroup(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(title, Modifier.padding(start = 12.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surface) { Column(content = content) }
    }
}

@Composable private fun SettingsRow(title: String, subtitle: String? = null, busy: Boolean, destructive: Boolean = false, action: () -> Unit) {
    Column(Modifier.fillMaxWidth().clickable(enabled = !busy, role = Role.Button, onClick = action).heightIn(min = 56.dp).padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(title, color = if(destructive) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurface)
        if(subtitle != null) Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
