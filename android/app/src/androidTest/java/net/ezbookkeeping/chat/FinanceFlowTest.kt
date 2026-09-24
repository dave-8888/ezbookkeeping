package net.ezbookkeeping.chat

import android.app.Application
import androidx.lifecycle.ViewModelStore
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import java.util.UUID
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicLong

@RunWith(AndroidJUnit4::class)
class FinanceFlowTest {
    @Test fun sixRequestsPersistAndSwitchingConversationCannotReuseOrdinal() = runBlocking {
        val app = ApplicationProvider.getApplicationContext<Application>()
        val user = "finance-test-${UUID.randomUUID()}"
        val balance = AtomicLong(500000)
        val writes = AtomicInteger()
        val recognition = AtomicInteger()
        val server = MockWebServer()
        val responses = java.util.ArrayDeque(listOf(
            "今天暂无支出", "今天暂无收入", "资产汇总，不含隐藏账户", "资产汇总，含隐藏账户",
            "以下是你的所有资产列表", "已更新 测试账户c，余额6000元",
            "请先明确当前对话中的账户，我还没有修改余额。", "已记账：午饭30元"
        ))
        val conversationIds = mutableListOf<String>()
        fun account(id: String, amount: Long, hidden: Boolean = false) = JSONObject().put("id", id).put("name", "测试账户$id")
            .put("type", 1).put("category", 1).put("currency", "CNY").put("hidden", hidden).put("balance", amount)
        val userJson = JSONObject().put("username", user).put("defaultAccountId", "a")
        server.dispatcher = object: Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val path = request.path.orEmpty()
                val result: Any = when {
                    path.endsWith("/authorize.json") -> JSONObject().put("token", "mock-token").put("user", userJson)
                    path.contains("tokens/refresh") -> JSONObject().put("user", userJson)
                    path.contains("accounts/list") -> JSONArray().put(account("a", 10000)).put(account("b", 20000)).put(account("c", balance.get())).apply {
                        if(path.contains("visible_only=false")) put(account("d", 30000, true))
                    }
                    path.contains("categories/list") -> JSONObject("""{"2":[{"id":"p","name":"支出","subCategories":[{"id":"c","name":"餐饮","type":2}]}]}""")
                    path.contains("tags/list") -> JSONArray()
                    path.contains("conversations/respond") -> {
                        val body = JSONObject(request.body.readUtf8())
                        assertEquals(setOf("conversationId", "messageId", "text"), body.keys().asSequence().toSet())
                        conversationIds.add(body.getString("conversationId"))
                        val reply = responses.removeFirst()
                        if(reply.startsWith("已更新")) { balance.set(600000); writes.incrementAndGet() }
                        JSONObject().put("messageId", body.getString("messageId")).put("status", "completed")
                            .put("reply", reply).put("details", "1. 测试账户a\n2. 测试账户b\n3. 测试账户c\n4. 隐藏账户d")
                    }
                    path.contains("llm/get") -> JSONObject().put("enabled", true)
                    path.contains("generate_title") -> JSONObject().put("title", "资产查询")
                    path.contains("recognize_transaction_text") -> { recognition.incrementAndGet(); JSONObject("""{"type":3,"sourceAmount":3000,"categoryId":"c"}""") }
                    path.contains("transactions/add") -> JSONObject().put("id", "mock-transaction")
                    path.contains("transactions/get") -> JSONObject().put("id", "mock-transaction").put("type", 3).put("sourceAmount", 3000).put("sourceAccountId", "a")
                    path.contains("transactions/list") -> JSONObject().put("items", JSONArray())
                    path.contains("balance/modify") -> {
                        val body = JSONObject(request.body.readUtf8())
                        assertEquals("c", body.getString("id"))
                        assertEquals(balance.get(), body.getLong("expectedBalance"))
                        balance.set(body.getLong("balance")); writes.incrementAndGet()
                        account("c", balance.get())
                    }
                    else -> return MockResponse().setResponseCode(404)
                }
                return MockResponse().setBody(JSONObject().put("success", true).put("result", result).toString())
            }
        }
        server.start()
        val baseUrl = server.url("/").toString()
        val db = Room.databaseBuilder(app, ChatDatabase::class.java, "chat.db").addMigrations(ChatDatabase.MIGRATION_1_2, ChatDatabase.MIGRATION_2_3).build()
        var store = ViewModelStore()
        suspend fun model(): ChatViewModel = withContext(Dispatchers.Main) { ChatViewModel(app).also { store.put("finance-test", it) } }
        suspend fun ready(vm: ChatViewModel) { withTimeout(30000) { while(vm.busy.value) delay(30) } }
        var vm = model()
        suspend fun send(text: String): ChatMessage {
            withContext(Dispatchers.Main) { vm.input.value = text; vm.send() }
            ready(vm)
            return db.messages().conversationHistory(vm.activeOwner.value, vm.activeConversation.value).first()
        }
        try {
            ready(vm)
            withContext(Dispatchers.Main) { vm.logout(); vm.login(baseUrl, user, "test-only", "", false) }
            ready(vm)
            assertTrue(vm.loggedIn.value)
            assertTrue(send("今日支出").reply.contains("暂无支出"))
            assertTrue(send("今天的收入明细").reply.contains("暂无收入"))
            assertTrue(send("我现在的所有资产").reply.contains("不含隐藏"))
            assertTrue(send("我现在的所有资产包含隐藏的").reply.contains("含隐藏"))
            val list = send("列出我的所有资产列表")
            assertTrue(JSONObject(list.draft!!).getString("details").contains("4. 隐藏账户d"))
            // Recreate the ViewModel: only the stable conversation ID goes back to the server.
            withContext(Dispatchers.Main) { store.clear() }
            store = ViewModelStore(); vm = model(); ready(vm)
            assertTrue(send("第三条资产金额更新 6000元").reply.contains("已更新 测试账户c"))
            assertEquals(600000L, balance.get())
            assertEquals(1, writes.get())
            assertEquals(0, recognition.get())
            withContext(Dispatchers.Main) { vm.newConversation() }; ready(vm)
            assertTrue(send("第三条资产金额更新 7000元").reply.contains("还没有修改"))
            assertEquals(1, writes.get())
            assertTrue(send("午饭花了30元").reply.contains("已记账"))
            assertEquals(0, recognition.get())
            assertTrue(responses.isEmpty())
            assertEquals(1, conversationIds.take(6).toSet().size)
            assertNotEquals(conversationIds[5], conversationIds[6])
            assertEquals(conversationIds[6], conversationIds[7])
        } finally {
            withContext(Dispatchers.Main) {
                vm.logout()
                vm.loginProfiles.value.filter { it.username == user }.forEach { vm.deleteLoginProfile(it.id) }
                store.clear()
            }
            SessionStore(app).apply { base = ""; this.user = "" }
            db.close(); server.shutdown()
        }
    }

    @Test fun interruptedAssetUpdateIsUnknownAndHistoryIsScoped() = runBlocking {
        val db = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), ChatDatabase::class.java).build()
        try {
            val dao = db.messages()
            dao.put(ChatMessage("a", "server|a", text = "更新", state = "agent_writing", conversationId = "one"))
            dao.put(ChatMessage("b", "server|b", text = "查询", state = "answered", conversationId = "one"))
            dao.put(ChatMessage("c", "server|a", text = "查询", state = "answered", conversationId = "two"))
            dao.recoverAgentTurns()
            assertEquals("agent_unknown", dao.conversationHistory("server|a", "one").single().state)
            assertEquals("b", dao.conversationHistory("server|b", "one").single().id)
            assertEquals("c", dao.conversationHistory("server|a", "two").single().id)
        } finally { db.close() }
    }
}
