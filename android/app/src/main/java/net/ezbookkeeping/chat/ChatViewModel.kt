package net.ezbookkeeping.chat

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.room.Room
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import org.json.JSONObject
import java.util.UUID
import java.io.IOException

class ChatViewModel(app: Application): AndroidViewModel(app) {
    private val session = SessionStore(app)
    private val history = LoginHistoryStore(app).apply { migratePreviousLogin(session.base, session.user) }
    val loginProfiles = MutableStateFlow(history.profiles())
    private var pendingLoginName = ""
    private val db = Room.databaseBuilder(app, ChatDatabase::class.java, "chat.db").build()
    private val dao = db.messages()
    private val owner = MutableStateFlow("")
    val activeOwner: StateFlow<String> = owner.asStateFlow()
    @OptIn(ExperimentalCoroutinesApi::class)
    val messages = owner.flatMapLatest { dao.observe(it) }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())
    val loggedIn = MutableStateFlow(false)
    val busy = MutableStateFlow(false)
    val error = MutableStateFlow("")
    val status = MutableStateFlow("未连接")
    val catalog = MutableStateFlow(Catalog())
    val editor = MutableStateFlow<ChatMessage?>(null)
    val input = MutableStateFlow("")
    val need2FA = MutableStateFlow(false)
    val aiSettingsNeeded = MutableStateFlow(false)
    val ledgerVisible = MutableStateFlow(true)
    val ledgerRoute = MutableStateFlow("")
    val ledgerRevision = MutableStateFlow(0)
    private var userInfo = JSONObject()
    var pendingExport: ByteArray? = null
    fun webSession(): JSONObject = JSONObject().put("token", session.token).put("user", userInfo)
    fun receiveWebToken(token: String) {
        if(loggedIn.value && token.isNotBlank()) { session.token = token; api?.token = token }
    }
    fun receiveWebUser(user: JSONObject) {
        if(user.optString("username") == session.user) {
            userInfo = user; defaultAccount = user.optString("defaultAccountId")
        }
    }
    fun openLedger(route: String = "") {
        if(busy.value) return
        ledgerRoute.value = route; ledgerVisible.value = true
    }
    fun returnToChat() {
        if(busy.value) return
        ledgerVisible.value = false
        reconnect()
    }
    var address = session.base
    private var api: Api? = null
    private var temporaryToken = ""
    private var defaultAccount = ""
    init {
        busy.value = true
        viewModelScope.launch {
            dao.recoverSaves(); dao.recoverRecognition()
            if(session.token.isNotBlank() && session.base.isNotBlank()) {
                api = Api(session.base, session.token)
                loggedIn.value = true
                owner.value = session.base + "|" + session.user
                try { refresh() } catch(e: Exception) { report(e) }
            }
            busy.value = false
        }
    }
    private suspend fun <T> io(block: () -> T): T = withContext(Dispatchers.IO) { block() }
    private fun report(e: Exception) {
        error.value = e.message ?: "操作失败，请检查网络连接"
        if(e is IOException || (e is ApiError && e.httpStatus >= 500)) status.value = "连接异常"
        if(e is ApiError && e.code in listOf(401, 202001, 202002, 202003, 202004, 202005, 202006, 202012)) {
            session.token = ""; loggedIn.value = false; owner.value = ""; api?.token = ""
            error.value = "登录已失效，请重新登录"
        }
    }
    private fun task(block: suspend () -> Unit) {
        if(busy.value) return
        busy.value = true; error.value = ""
        viewModelScope.launch {
            try { block() } catch(e: Exception) { report(e) } finally { busy.value = false }
        }
    }
    fun login(raw: String, user: String, password: String, code: String, recovery: Boolean) = task {
        if(!need2FA.value) {
            val base = Api.normalize(raw)
            val connection = Api(base)
            api = connection
            pendingLoginName = user.trim()
            val auth = io { connection.objectCall("authorize.json", JSONObject().put("loginName", user.trim()).put("password", password)) }
            if(auth.optBoolean("need2FA")) {
                temporaryToken = auth.getString("token"); need2FA.value = true; status.value = "请输入双重验证码"
            } else finishLogin(auth)
        } else {
            val auth = io { api!!.objectCall(if(recovery) "2fa/recovery.json" else "2fa/authorize.json",
                JSONObject().put(if(recovery) "recoveryCode" else "passcode", code.trim()), temporaryToken) }
            finishLogin(auth)
        }
    }
    private suspend fun finishLogin(auth: JSONObject) {
        val user = auth.getJSONObject("user")
        userInfo = user
        val connection = api!!
        connection.token = auth.getString("token")
        session.base = connection.base; address = connection.base; session.user = user.getString("username"); session.token = connection.token
        history.remember(connection.base, pendingLoginName.ifBlank { session.user })
        loginProfiles.value = history.profiles()
        pendingLoginName = ""
        defaultAccount = user.optString("defaultAccountId")
        owner.value = session.base + "|" + session.user
        temporaryToken = ""; need2FA.value = false; loggedIn.value = true
        loadCatalog()
    }
    private suspend fun refresh() {
        val connection = api ?: kotlin.error("请先登录")
        val result = io { connection.objectCall("v1/tokens/refresh.json", JSONObject()) }
        result.optString("newToken").takeIf { it.isNotBlank() }?.let { connection.token = it; session.token = it }
        defaultAccount = result.getJSONObject("user").optString("defaultAccountId")
        userInfo = result.getJSONObject("user")
        loadCatalog()
    }
    private suspend fun loadCatalog() {
        catalog.value = io { api!!.catalog(defaultAccount) }
        status.value = "已连接内网服务"
    }
    fun editLoginProfile(id: String, base: String, username: String): LoginProfile {
        check(!busy.value) { "请等待当前操作完成" }
        val profile = history.edit(id, base, username)
        loginProfiles.value = history.profiles()
        return profile
    }
    fun deleteLoginProfile(id: String) {
        if(busy.value) return
        history.delete(id)
        loginProfiles.value = history.profiles()
    }
    fun reconnect() = task { refresh() }
    fun cancel2FA() { if(!busy.value) { need2FA.value = false; temporaryToken = ""; api = null } }
    val editServiceOnLogin = MutableStateFlow(false)
    fun logout(editService: Boolean = false) {
        if(busy.value) return
        editServiceOnLogin.value = editService
        session.token = ""; api = null; loggedIn.value = false; owner.value = ""; editor.value = null
        input.value = ""; error.value = ""; status.value = "未连接"; catalog.value = Catalog(); aiSettingsNeeded.value = false
        ledgerVisible.value = true; ledgerRoute.value = "/"; userInfo = JSONObject(); pendingExport = null
    }
    fun clearMessages() = task { dao.clear(owner.value) }
    fun openEditor(message: ChatMessage) { if(!busy.value && message.state == "draft" && message.owner == owner.value) editor.value = message }
    fun send() {
        val text = input.value.trim()
        if(text.isEmpty() || !loggedIn.value) return
        task {
            var message = ChatMessage(UUID.randomUUID().toString(), owner.value, text = text)
            dao.put(message)
            try {
                refresh()
                val ai = io { api!!.objectCall("v1/users/settings/llm/get.json") }
                if(!ai.optBoolean("enabled")) {
                    aiSettingsNeeded.value = true
                    throw IllegalStateException("请先在 AI 设置中启用记账模型")
                }
                aiSettingsNeeded.value = false
                val result = io { api!!.objectCall("v1/llm/transactions/recognize_transaction_text.json", JSONObject().put("text", text)) }
                val (draft, missing) = TransactionRules.recognize(result, text, catalog.value)
                message = message.copy(draft = draft.json().toString())
                if(missing.isNotEmpty()) {
                    dao.put(message.copy(state = "draft", reply = "请补全：${missing.joinToString("、")}"))
                } else save(message, draft)
                if(input.value.trim() == text) input.value = ""
            } catch(e: Exception) {
                dao.put(message.copy(state = "failed", reply = e.message ?: "识别失败，请重新输入"))
                report(e)
            }
        }
    }
    fun saveEditor(draft: Draft) {
        val message = editor.value ?: return
        task {
            refresh()
            val missing = TransactionRules.validate(draft, catalog.value)
            require(missing.isEmpty()) { "请检查：${missing.joinToString("、")}" }
            editor.value = null
            save(message, draft)
        }
    }
    private suspend fun save(message: ChatMessage, draft: Draft) {
        val pending = message.copy(state = "saving", reply = "正在保存…", draft = draft.json().toString())
        // Persist before any request so process death cannot trigger an automatic duplicate submission.
        dao.put(pending)
        try {
            val result = io { api!!.objectCall("v1/transactions/add.json", draft.json()) }
            val id = result.getString("id")
            dao.put(pending.copy(state = "saved", reply = "已记账\n" + TransactionRules.summary(draft, catalog.value), transactionId = id))
            ledgerRevision.value += 1
            status.value = "已连接内网服务"
        } catch(e: Exception) {
            val definiteRejection = e is ApiError && e.code >= 200000 && e.httpStatus in 400..499 && e.code !in listOf(202001,202002,202003,202004,202005,202006,202012)
            dao.put(pending.copy(state = if(definiteRejection) "draft" else "unknown",
                reply = if(definiteRejection) "保存失败：${e.message}，请检查并补全交易" else "保存结果待确认，请查看账单，避免重复记账。"))
            report(e)
        }
    }
    override fun onCleared() { super.onCleared(); db.close() }
}
