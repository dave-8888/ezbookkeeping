package net.ezbookkeeping.chat

import android.annotation.SuppressLint
import android.app.Activity
import android.content.Intent
import android.graphics.Bitmap
import android.net.Uri
import android.util.Base64
import android.view.View
import android.webkit.*
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.unit.dp
import androidx.webkit.JavaScriptReplyProxy
import androidx.webkit.WebMessageCompat
import androidx.webkit.WebViewCompat
import androidx.webkit.WebViewFeature
import kotlinx.coroutines.delay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.ByteArrayInputStream

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun LedgerWebView(vm: ChatViewModel, visible: Boolean, route: String, revision: Int) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val policy = remember(vm.address) { LedgerPolicy(vm.address) }
    var browser by remember { mutableStateOf<WebView?>(null) }
    var proxy by remember { mutableStateOf<JavaScriptReplyProxy?>(null) }
    var ready by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var problem by remember { mutableStateOf("") }
    var attempt by remember { mutableIntStateOf(0) }
    var fileCallback by remember { mutableStateOf<ValueCallback<Array<Uri>>?>(null) }
    val currentVisible by rememberUpdatedState(visible)
    val owner = remember { vm.activeOwner.value }
    val pickFile = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        fileCallback?.onReceiveValue(WebChromeClient.FileChooserParams.parseResult(result.resultCode, result.data))
        fileCallback = null
    }
    val saveFile = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        val bytes = vm.pendingExport
        vm.pendingExport = null
        val uri = result.data?.data
        if(result.resultCode == Activity.RESULT_OK && uri != null && bytes != null) {
            scope.launch {
                try {
                    withContext(Dispatchers.IO) {
                        context.contentResolver.openOutputStream(uri)?.use { it.write(bytes) }
                            ?: error("无法打开所选文件")
                    }
                    android.widget.Toast.makeText(context, "文件已保存", android.widget.Toast.LENGTH_SHORT).show()
                } catch(_: Exception) { vm.error.value = "文件保存失败，请重新导出" }
            }
        }
    }
    fun external(url: String) {
        val uri = Uri.parse(url)
        if(uri.scheme !in listOf("https", "http", "mailto", "tel")) return
        try { context.startActivity(Intent(Intent.ACTION_VIEW, uri)) }
        catch(_: Exception) { vm.error.value = "没有可打开此链接的应用" }
    }
    LaunchedEffect(attempt, ready) {
        if(!ready) {
            delay(15000)
            if(!ready && problem.isEmpty()) {
                loading = false
                problem = "账本尚未连接。请检查网络，并确认服务端已更新到支持 App 整合的版本。"
            }
        }
    }
    LaunchedEffect(visible, ready, revision, route) {
        if(visible && ready) {
            proxy?.postMessage(vm.webSession().put("type", "resume").put("revision", revision).put("route", route).toString())
        }
    }
    BackHandler(visible) {
        if(ready && proxy != null) proxy?.postMessage("{\"type\":\"back\"}")
        else (context as? Activity)?.moveTaskToBack(true)
    }
    DisposableEffect(Unit) {
        onDispose {
            fileCallback?.onReceiveValue(null)
            fileCallback = null
            browser?.apply {
                stopLoading()
                if(WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER))
                    WebViewCompat.removeWebMessageListener(this, "FinanciallyFreeNative")
                loadUrl("about:blank")
                clearHistory(); clearCache(true); removeAllViews(); destroy()
            }
            proxy = null
            if(!vm.loggedIn.value) WebStorage.getInstance().deleteAllData()
        }
    }
    Box(if(visible) Modifier.fillMaxSize() else Modifier.size(0.dp)) {
        AndroidView(factory = { ctx ->
            WebView(ctx).apply {
                browser = this
                setBackgroundColor(android.graphics.Color.TRANSPARENT)
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.allowFileAccess = false
                settings.allowContentAccess = true // Files explicitly selected by the system picker.
                settings.mixedContentMode = WebSettings.MIXED_CONTENT_NEVER_ALLOW
                settings.cacheMode = WebSettings.LOAD_NO_CACHE
                settings.setSupportMultipleWindows(false)
                CookieManager.getInstance().setAcceptThirdPartyCookies(this, false)
                webChromeClient = object: WebChromeClient() {
                    override fun onShowFileChooser(view: WebView?, callback: ValueCallback<Array<Uri>>, params: FileChooserParams): Boolean {
                        if(!policy.allowsDocument(url) || !currentVisible) return false
                        fileCallback?.onReceiveValue(null)
                        fileCallback = callback
                        try { pickFile.launch(params.createIntent()) }
                        catch(_: Exception) { callback.onReceiveValue(null); fileCallback = null; vm.error.value = "无法打开系统文件选择器" }
                        return true
                    }
                }
                webViewClient = object: WebViewClient() {
                    override fun onPageStarted(view: WebView, url: String, favicon: Bitmap?) {
                        ready = false; proxy = null; loading = true; problem = ""; attempt++
                    }
                    override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
                        if(!request.isForMainFrame) return false
                        if(policy.allowsDocument(request.url.toString())) return false
                        if(request.hasGesture()) external(request.url.toString())
                        return true
                    }
                    override fun shouldInterceptRequest(view: WebView, request: WebResourceRequest): WebResourceResponse? {
                        if(request.isForMainFrame && !policy.allowsDocument(request.url.toString()))
                            return WebResourceResponse("text/plain", "UTF-8", 403, "Forbidden", emptyMap(), ByteArrayInputStream(ByteArray(0)))
                        return null
                    }
                    override fun onReceivedError(view: WebView, request: WebResourceRequest, error: WebResourceError) {
                        if(request.isForMainFrame) { loading = false; problem = "无法连接账本，请检查服务地址和网络后重试" }
                    }
                    override fun onReceivedHttpError(view: WebView, request: WebResourceRequest, response: WebResourceResponse) {
                        if(request.isForMainFrame) { loading = false; problem = "账本页面暂不可用（${response.statusCode}），请检查服务端部署" }
                    }
                    // SSL errors retain WebView's default cancellation. Never bypass certificate checks.
                }
                setDownloadListener { url, _, _, _, _ ->
                    if(url.startsWith("blob:")) vm.error.value = "请使用页面中的下载按钮保存文件"
                    else external(url)
                }
                if(!WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER)) {
                    loading = false; problem = "请更新 Android System WebView 后使用账本"
                } else {
                    WebViewCompat.addWebMessageListener(this, "FinanciallyFreeNative", setOf(policy.origin)) { view, message: WebMessageCompat, _, mainFrame, reply ->
                        if(mainFrame && policy.allowsDocument(view.url) && vm.loggedIn.value && vm.activeOwner.value == owner) {
                            try {
                                val data = JSONObject(message.data ?: "{}")
                                when(data.optString("type")) {
                                    "session" -> {
                                        proxy = reply
                                        scope.launch {
                                            while(vm.busy.value && vm.webSession().getJSONObject("user").optString("username").isBlank()) delay(50)
                                            if(vm.loggedIn.value && vm.activeOwner.value == owner) {
                                                val session = vm.webSession()
                                                if(session.getJSONObject("user").optString("username").isNotBlank())
                                                    reply.postMessage(session.put("type", "session").toString())
                                                else { loading = false; problem = "尚未取得账号信息，请检查网络后重试" }
                                            }
                                        }
                                    }
                                    "ready" -> { ready = true; loading = false; problem = "" }
                                    "token" -> vm.receiveWebToken(data.getString("token"))
                                    "user" -> vm.receiveWebUser(data.getJSONObject("user"))
                                    "logout" -> scope.launch {
                                        while(vm.busy.value) delay(50)
                                        if(vm.activeOwner.value == owner) vm.logout()
                                    }
                                    "chat" -> if(currentVisible) vm.returnToChat()
                                    "exit" -> if(currentVisible) (context as? Activity)?.moveTaskToBack(true)
                                    "error" -> vm.error.value = data.optString("message", "操作失败")
                                    "download" -> if(currentVisible && vm.pendingExport == null) {
                                        val encoded = data.getString("data")
                                        require(encoded.length <= 28 * 1024 * 1024)
                                        vm.pendingExport = Base64.decode(encoded, Base64.DEFAULT)
                                        val name = data.optString("name", "export.csv").substringAfterLast('/').substringAfterLast('\\').take(160)
                                        try {
                                            saveFile.launch(Intent(Intent.ACTION_CREATE_DOCUMENT).apply {
                                                addCategory(Intent.CATEGORY_OPENABLE)
                                                type = data.optString("mime").ifBlank { "application/octet-stream" }
                                                putExtra(Intent.EXTRA_TITLE, name)
                                            })
                                        } catch(_: Exception) { vm.pendingExport = null; vm.error.value = "无法打开保存位置，请重试" }
                                    }
                                }
                            } catch(_: Exception) { vm.error.value = "账本通信失败，请返回后重试" }
                        }
                    }
                    loadUrl(policy.url(route))
                }
            }
        }, update = { it.visibility = if(visible) View.VISIBLE else View.INVISIBLE }, modifier = Modifier.fillMaxSize())
        if(visible && (loading || problem.isNotEmpty())) {
            Surface(Modifier.fillMaxSize()) {
                Column(Modifier.padding(28.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
                    if(problem.isEmpty()) { CircularProgressIndicator(); Spacer(Modifier.height(16.dp)); Text("正在打开账本…") }
                    else {
                        Text(problem)
                        Spacer(Modifier.height(16.dp))
                        Button(onClick = { vm.reconnect(); ready = false; problem = ""; loading = true; attempt++; browser?.loadUrl(policy.url(route)) }) { Text("重试") }
                        TextButton(onClick = vm::returnToChat) { Text("返回对话记账") }
                        TextButton(onClick = { external(policy.url(route)) }) { Text("在浏览器打开") }
                    }
                }
            }
        }
    }
}
