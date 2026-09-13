package net.ezbookkeeping.chat

import android.os.Bundle
import androidx.activity.compose.BackHandler
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import org.json.JSONObject

class MainActivity: ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            BookkeepingTheme {
                Surface(Modifier.fillMaxSize()) { App() }
            }
        }
    }
}

@Composable fun App(vm: ChatViewModel = viewModel()) {
    val logged by vm.loggedIn.collectAsStateWithLifecycle()
    val busy by vm.busy.collectAsStateWithLifecycle()
    val error by vm.error.collectAsStateWithLifecycle()
    val connectionStatus by vm.status.collectAsStateWithLifecycle()
    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).safeDrawingPadding().imePadding()) {
        if(error.isNotEmpty()) {
            Surface(color = MaterialTheme.colorScheme.errorContainer) {
                Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text(error, Modifier.weight(1f), style = MaterialTheme.typography.bodySmall)
                    if(logged && connectionStatus == "连接异常") TextButton(onClick = vm::reconnect, enabled = !busy) { Text("重试") }
                    TextButton(onClick = { vm.error.value = "" }) { Text("关闭") }
                }
            }
        }
        if(!logged) Login(vm, busy) else key(vm.activeOwner.collectAsStateWithLifecycle().value) { HybridHome(vm, busy) }
    }
}

@Composable private fun HybridHome(vm: ChatViewModel, busy: Boolean) {
    val ledger by vm.ledgerVisible.collectAsStateWithLifecycle()
    val route by vm.ledgerRoute.collectAsStateWithLifecycle()
    val revision by vm.ledgerRevision.collectAsStateWithLifecycle()
    val editing by vm.editor.collectAsStateWithLifecycle()
    val catalog by vm.catalog.collectAsStateWithLifecycle()
    var opened by rememberSaveable { mutableStateOf(ledger) }
    if(ledger) opened = true
    var settings by rememberSaveable { mutableStateOf(false) }
    val focus = LocalFocusManager.current
    BackHandler(!ledger && !settings) {
        if(!busy) {
            if(editing != null) vm.editor.value = null else vm.openLedger()
        }
    }
    Box(Modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize().then(if(settings) Modifier.clearAndSetSemantics {} else Modifier)) {
            if(!ledger) {
                ChatNavigation(ledger, !busy && editing == null, { selected ->
                    if(selected != ledger) { if(selected) vm.openLedger() else vm.returnToChat() }
                }, { focus.clearFocus(); settings = true })
            }
            Box(Modifier.weight(1f)) {
                if(!ledger) {
                    if(editing != null) key(editing!!.id) {
                        TransactionEditor(Draft.from(JSONObject(editing!!.draft!!)), catalog, busy, { vm.editor.value = null }, vm::saveEditor, listOf("交易类型", "金额", "账户", "分类").filter { editing!!.reply.contains(it) })
                    } else Chat(vm, busy)
                }
                if(opened) LedgerWebView(vm, ledger && !settings, route, revision)
            }
        }
        if(settings) Surface(Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
            SettingsScreen(vm, busy) { settings = false }
        }
    }
}

@Composable private fun Login(vm: ChatViewModel, busy: Boolean) {
    val profiles by vm.loginProfiles.collectAsStateWithLifecycle()
    var address by rememberSaveable { mutableStateOf(profiles.firstOrNull()?.base.orEmpty()) }
    var user by rememberSaveable { mutableStateOf(profiles.firstOrNull()?.username.orEmpty()) }
    var showHistory by rememberSaveable { mutableStateOf(false) }
    var editConnection by rememberSaveable { mutableStateOf(profiles.isEmpty() || vm.editServiceOnLogin.value) }
    // Passwords and verification codes never enter SavedState or persistent storage.
    var password by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var recovery by rememberSaveable { mutableStateOf(false) }
    val twoFA by vm.need2FA.collectAsStateWithLifecycle()
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Spacer(Modifier.height(40.dp))
        Text("财神", style = MaterialTheme.typography.headlineMedium)
        Text("说一句，记一笔", style = MaterialTheme.typography.titleMedium)
        if(!twoFA) {
            if(editConnection || profiles.isEmpty()) {
                AppField(address, { address = it; password = "" }, label = { Text("服务地址") }, placeholder = { Text("http://192.168.1.10:8080/") },
                    supportingText = { Text("填写手机可访问的服务地址") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri))
                AppField(user, { user = it; password = "" }, label = { Text("用户名或邮箱") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy)
            } else {
                Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surface) {
                    Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(user, style = MaterialTheme.typography.titleLarge)
                        Text(address, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            if(profiles.isNotEmpty()) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    TextButton(onClick = { showHistory = true }, enabled = !busy) { Text("切换账号") }
                    if(!editConnection) TextButton(onClick = { editConnection = true; password = "" }, enabled = !busy) { Text("修改服务地址") }
                }
            }
            AppField(password, { password = it }, label = { Text("密码") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy, visualTransformation = PasswordVisualTransformation())
        } else {
            LaunchedEffect(Unit) { password = "" }
            Text("账号已开启双重验证")
            AppField(code, { code = it }, label = { Text(if(recovery) "恢复码" else "验证码") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy)
            Row(verticalAlignment = Alignment.CenterVertically) { Checkbox(recovery, { recovery = it }, enabled = !busy); Text("使用恢复码") }
            TextButton(onClick = vm::cancel2FA, enabled = !busy) { Text("返回登录") }
        }
        Button(onClick = { vm.login(address, user, password, code, recovery) }, modifier = Modifier.fillMaxWidth(), enabled = !busy && if(twoFA) code.isNotBlank() else address.isNotBlank() && user.isNotBlank() && password.isNotBlank()) {
            Text(if(busy) "正在连接…" else if(twoFA) "验证并登录" else "登录")
        }
    }
    if(showHistory) LoginHistoryDialog(vm, onDismiss = { showHistory = false }, onNewAccount = {
        showHistory = false; editConnection = true; user = ""; password = ""; code = ""
    }, onSelect = {
        address = it.base; user = it.username; password = ""; code = ""; editConnection = false
    }, onChanged = { before, after ->
        if(address.trim().trimEnd('/') == before.base.trimEnd('/') && user.trim() == before.username) {
            address = after?.base.orEmpty(); user = after?.username.orEmpty(); password = ""; code = ""; editConnection = after == null
        }
    })
}

@Composable private fun Chat(vm: ChatViewModel, busy: Boolean) {
    val storedMessages by vm.messages.collectAsStateWithLifecycle()
    val activeOwner by vm.activeOwner.collectAsStateWithLifecycle()
    val messages = storedMessages.filter { it.owner == activeOwner }
    val status by vm.status.collectAsStateWithLifecycle()
    val error by vm.error.collectAsStateWithLifecycle()
    val input by vm.input.collectAsStateWithLifecycle()
    val aiNeeded by vm.aiSettingsNeeded.collectAsStateWithLifecycle()
    Column(Modifier.fillMaxSize()) {
        if(status == "连接异常" && error.isBlank()) {
            Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("连接中断", Modifier.weight(1f), color = MaterialTheme.colorScheme.error)
                TextButton(onClick = vm::reconnect, enabled = !busy) { Text("重试") }
            }
        }
        if(aiNeeded) TextButton(onClick = { vm.openLedger("/settings/ai") }, enabled = !busy) { Text("设置 AI 记账") }
        ChatMessageList(messages, busy, { vm.openEditor(it) }, { vm.openLedger("/transaction/list") }, Modifier.weight(1f))
        ChatComposer(input, busy, { vm.input.value = it }, vm::send)
    }

}
