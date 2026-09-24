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
    private val db = Room.databaseBuilder(app, ChatDatabase::class.java, "chat.db").addMigrations(ChatDatabase.MIGRATION_1_2, ChatDatabase.MIGRATION_2_3).build()
    private val dao = db.messages()
    private val owner = MutableStateFlow("")
    val activeOwner: StateFlow<String> = owner.asStateFlow()
    private val selectedConversation = MutableStateFlow("")
    val activeConversation: StateFlow<String> = selectedConversation.asStateFlow()
    @OptIn(ExperimentalCoroutinesApi::class)
    val conversations = owner.flatMapLatest { dao.observeConversations(it) }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())
    @OptIn(ExperimentalCoroutinesApi::class)
    val messages = combine(owner, selectedConversation) { account, id -> account to id }
        .flatMapLatest { (account, id) -> dao.observeConversation(account, id) }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())
    val loggedIn = MutableStateFlow(false)
    val busy = MutableStateFlow(false)
    val error = MutableStateFlow("")
    val status = MutableStateFlow("未连接")
    val catalog = MutableStateFlow(Catalog())
    val editor = MutableStateFlow<ChatMessage?>(null)
    val input = MutableStateFlow("")
    val need2FA = MutableStateFlow(false)
    val aiSettingsNeeded = MutableStateFlow(false)
    val aiSettings = MutableStateFlow<JSONObject?>(null)
    fun loadAISettings() = task {
        refresh()
        aiSettings.value = io { api!!.objectCall("v1/users/settings/llm/get.json") }
    }
    fun saveAISettings(settings: JSONObject, saved: () -> Unit) = task {
        val result = io { api!!.objectCall("v1/users/settings/llm/update.json", settings) }
        aiSettings.value = result
        aiSettingsNeeded.value = !result.optBoolean("enabled")
        saved()
    }
    var address = session.base
    private var api: Api? = null
    private var temporaryToken = ""
    private var defaultAccount = ""
    private val titleJobs = mutableMapOf<String, Job>()
    init {
        busy.value = true
        viewModelScope.launch {
            dao.recoverSaves(); dao.recoverRecognition(); dao.recoverAssetUpdates(); dao.recoverAgentTurns()
            if(session.token.isNotBlank() && session.base.isNotBlank()) {
                api = Api(session.base, session.token)
                loggedIn.value = true
                owner.value = session.base + "|" + session.user
                restoreConversation()
                try { refresh(); generateTitle(selectedConversation.value) } catch(e: Exception) { report(e) }
            }
            busy.value = false
        }
    }
    private suspend fun <T> io(block: () -> T): T = withContext(Dispatchers.IO) { block() }
    private fun report(e: Exception) {
        error.value = e.message ?: "操作失败，请检查网络连接"
        if(e is IOException || (e is ApiError && e.httpStatus >= 500)) status.value = "连接异常"
        if(e is ApiError && e.code in listOf(401, 202001, 202002, 202003, 202004, 202005, 202006, 202012)) {
            cancelTitleJobs()
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
        val connection = api!!
        connection.token = auth.getString("token")
        session.base = connection.base; address = connection.base; session.user = user.getString("username"); session.token = connection.token
        history.remember(connection.base, pendingLoginName.ifBlank { session.user })
        loginProfiles.value = history.profiles()
        pendingLoginName = ""
        defaultAccount = user.optString("defaultAccountId")
        owner.value = session.base + "|" + session.user
        restoreConversation()
        temporaryToken = ""; need2FA.value = false; loggedIn.value = true
        loadCatalog()
        generateTitle(selectedConversation.value)
    }
    private suspend fun refresh(loadDirectories: Boolean = true) {
        val connection = api ?: kotlin.error("请先登录")
        val result = io { connection.objectCall("v1/tokens/refresh.json", JSONObject()) }
        result.optString("newToken").takeIf { it.isNotBlank() }?.let { connection.token = it; session.token = it }
        defaultAccount = result.getJSONObject("user").optString("defaultAccountId")
        if(loadDirectories) loadCatalog()
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
    fun reconnect() = task { refresh(); generateTitle(selectedConversation.value) }
    fun cancel2FA() { if(!busy.value) { need2FA.value = false; temporaryToken = ""; api = null } }
    val editServiceOnLogin = MutableStateFlow(false)
    fun logout(editService: Boolean = false) {
        if(busy.value) return
        cancelTitleJobs()
        editServiceOnLogin.value = editService
        session.token = ""; api = null; loggedIn.value = false; owner.value = ""; editor.value = null
        input.value = ""; error.value = ""; status.value = "未连接"; catalog.value = Catalog(); aiSettingsNeeded.value = false
        aiSettings.value = null
    }
    private fun selectConversation(id: String) {
        selectedConversation.value = id
        session.selectConversation(owner.value, id)
        editor.value = null
        input.value = ""
    }
    private suspend fun restoreConversation() {
        val existing = dao.conversations(owner.value)
        val selected = existing.firstOrNull { it.id == session.selectedConversation(owner.value) } ?: existing.firstOrNull()
        if(selected != null) selectConversation(selected.id) else createConversation()
    }
    private suspend fun createConversation() {
        val conversation = Conversation(UUID.randomUUID().toString(), owner.value)
        dao.addConversation(conversation)
        selectConversation(conversation.id)
    }
    fun newConversation() = task { if(loggedIn.value) createConversation() }
    fun switchConversation(id: String) = task {
        if(loggedIn.value && dao.conversations(owner.value).any { it.id == id }) {
            selectConversation(id)
            generateTitle(id)
        }
    }
    fun deleteConversation(id: String) = task {
        if(!loggedIn.value) return@task
        titleJobs.remove(id)?.cancel()
        dao.deleteConversation(owner.value, id)
        if(selectedConversation.value == id) restoreConversation()
    }
    fun clearMessages() = task { cancelTitleJobs(); dao.clear(owner.value) }
    private fun cancelTitleJobs() {
        titleJobs.values.toList().forEach { it.cancel() }
        titleJobs.clear()
    }
    private fun generateTitle(id: String) {
        if(!loggedIn.value || id.isEmpty() || titleJobs[id]?.isActive == true) return
        val account = owner.value
        val connection = api?.let { Api(it.base, it.token) } ?: return
        val job = viewModelScope.launch(start = CoroutineStart.LAZY) {
            try {
                val conversation = dao.conversation(account, id) ?: return@launch
                if(conversation.titleGenerated) return@launch
                val first = dao.firstMessage(account, id) ?: return@launch
                dao.setFallbackTitle(account, id, ConversationTitles.fallback(conversation.created))
                val title = io { ConversationTitles.generate(connection, conversation.created, first.text) }
                ensureActive()
                dao.saveGeneratedTitle(account, id, first.id, title)
            } catch(e: CancellationException) {
                throw e
            } catch(_: Exception) {
                // Title generation is optional. Keep the date title and retry when this chat is reopened.
            }
        }
        titleJobs[id] = job
        job.invokeOnCompletion { if(titleJobs[id] === job) titleJobs.remove(id) }
        job.start()
    }
    fun openEditor(message: ChatMessage) {
        if(!busy.value && message.state == "server_pending" && message.owner == owner.value && message.conversationId == selectedConversation.value) {
            task { refresh(false); dao.put(io { ServerConversationClient(api!!).result(message) }) }
            return
        }
        if(!busy.value && message.state == "draft" && message.owner == owner.value && message.conversationId == selectedConversation.value) editor.value = message }
    fun send() {
        val text = input.value.trim()
        if(text.isEmpty() || !loggedIn.value) return
        task {
            val message = ChatMessage(UUID.randomUUID().toString(), owner.value, text = text, conversationId = selectedConversation.value,
                state = "server_pending", reply = "正在处理…")
            dao.conversation(message.owner, message.conversationId)?.let {
                dao.setFallbackTitle(it.owner, it.id, ConversationTitles.fallback(it.created))
            }
            // Persist the request identity before sending. Checking a lost reply never resends a write.
            dao.put(message)
            var submitted = false
            try {
                refresh(false)
                submitted = true
                val result = io { ServerConversationClient(api!!).send(message) }
                dao.put(result)
                if(input.value.trim() == text) input.value = ""
            } catch(e: Exception) {
                if(e is CancellationException) throw e
                val reply = if(submitted) "连接暂时中断，消息可能仍在处理。请查看处理结果，避免重复提交。" else "暂时无法连接服务，请检查连接后重试。"
                dao.put(message.copy(state = if(submitted) "server_pending" else "failed", reply = reply))
                report(e)
                if(loggedIn.value) error.value = reply
            }
            generateTitle(message.conversationId)
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
            status.value = "已连接内网服务"
        } catch(e: Exception) {
            val definiteRejection = e is ApiError && e.code >= 200000 && e.httpStatus in 400..499 && e.code !in listOf(202001,202002,202003,202004,202005,202006,202012)
            dao.put(pending.copy(state = if(definiteRejection) "draft" else "unknown",
                reply = if(definiteRejection) "保存失败：${e.message}，请检查并补全交易" else "保存结果待确认，请在网页版核实，避免重复记账。"))
            report(e)
        }
    }
    override fun onCleared() { super.onCleared(); db.close() }
}
