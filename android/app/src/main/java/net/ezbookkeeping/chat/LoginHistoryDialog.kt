package net.ezbookkeeping.chat

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle

@Composable fun LoginHistoryDialog(
    vm: ChatViewModel,
    onDismiss: () -> Unit,
    onSelect: ((LoginProfile) -> Unit)? = null,
    onNewAccount: (() -> Unit)? = null,
    onChanged: (LoginProfile, LoginProfile?) -> Unit = { _, _ -> }
) {
    val profiles by vm.loginProfiles.collectAsStateWithLifecycle()
    val busy by vm.busy.collectAsStateWithLifecycle()
    var editingId by rememberSaveable { mutableStateOf<String?>(null) }
    var deletingId by rememberSaveable { mutableStateOf<String?>(null) }
    val editing = profiles.find { it.id == editingId }
    val deleting = profiles.find { it.id == deletingId }
    when {
        editing != null -> key(editing.id) {
            var base by rememberSaveable { mutableStateOf(editing.base) }
            var username by rememberSaveable { mutableStateOf(editing.username) }
            var problem by rememberSaveable { mutableStateOf("") }
            AlertDialog(onDismissRequest = { editingId = null }, title = { Text("修改登录记录") }, text = {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    AppField(base, { base = it }, label = { Text("服务地址") }, singleLine = true, enabled = !busy, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri))
                    AppField(username, { username = it }, label = { Text("用户名或邮箱") }, singleLine = true, enabled = !busy)
                    Text("仅修改本机记录，不会更改当前登录或服务器账号。", style = MaterialTheme.typography.bodySmall)
                    if(problem.isNotEmpty()) Text(problem, color = MaterialTheme.colorScheme.error)
                }
            }, confirmButton = { TextButton(enabled = !busy, onClick = {
                try {
                    val updated = vm.editLoginProfile(editing.id, base, username)
                    onChanged(editing, updated)
                    editingId = null
                } catch(_: IllegalArgumentException) { problem = "请填写有效的 http/https 服务地址和用户名" }
            }) { Text("保存修改") } }, dismissButton = { TextButton(onClick = { editingId = null }) { Text("取消") } })
        }
        deleting != null -> AlertDialog(onDismissRequest = { deletingId = null }, title = { Text("删除登录记录？") }, text = {
            Text("${deleting.username}\n${deleting.base}\n\n仅移除本机登录记录，不删除账号或聊天记录。")
        }, confirmButton = { TextButton(enabled = !busy, onClick = {
            vm.deleteLoginProfile(deleting.id)
            onChanged(deleting, null)
            deletingId = null
        }) { Text("删除记录") } }, dismissButton = { TextButton(onClick = { deletingId = null }) { Text("取消") } })
        else -> AlertDialog(onDismissRequest = onDismiss, title = { Text("登录记录") }, text = {
            if(profiles.isEmpty()) Text("暂无登录记录，成功登录后会自动保存地址和用户名。")
            else LazyColumn(Modifier.heightIn(max = 400.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(profiles, key = { it.id }) { profile ->
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(profile.username, style = MaterialTheme.typography.titleSmall)
                        Text(profile.base, style = MaterialTheme.typography.bodySmall)
                        Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                            if(onSelect != null) TextButton(onClick = { onSelect(profile); onDismiss() }, enabled = !busy) { Text("使用") }
                            TextButton(onClick = { editingId = profile.id }, enabled = !busy) { Text("修改") }
                            TextButton(onClick = { deletingId = profile.id }, enabled = !busy) { Text("删除") }
                        }
                        HorizontalDivider()
                    }
                }
            }
        }, confirmButton = { TextButton(onClick = onDismiss) { Text("完成") } },
            dismissButton = { if(onNewAccount != null) TextButton(onClick = onNewAccount, enabled = !busy) { Text("使用其他账号") } })
    }
}
