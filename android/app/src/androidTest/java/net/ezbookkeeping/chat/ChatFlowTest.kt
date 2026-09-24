package net.ezbookkeeping.chat

import androidx.compose.ui.test.*
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.test.platform.app.InstrumentationRegistry
import java.io.File
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.test.ext.junit.runners.AndroidJUnit4
import okhttp3.mockwebserver.*
import org.junit.*
import org.junit.runner.RunWith
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicBoolean

@RunWith(AndroidJUnit4::class)
class ChatFlowTest {
    @get:Rule val compose = createAndroidComposeRule<MainActivity>()
    private val saves = AtomicInteger(0)
    private val lastAmount = java.util.concurrent.atomic.AtomicLong(3050)
    private val titleRequests = AtomicInteger(0)
    private val failTitle = AtomicBoolean(false)
    private val title = ConversationTitles.prefix(System.currentTimeMillis()) + "午餐支出"
    private val webRequests = AtomicInteger(0)
    private val settingsSaves = AtomicInteger(0)
    private val failRefresh = AtomicBoolean(false)
    private lateinit var server: MockWebServer
    @Before fun setup() {
        server = MockWebServer()
        server.dispatcher = object: Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                if(request.path!!.contains("tokens/refresh") && failRefresh.get()) return MockResponse().setResponseCode(503)
                if(!request.path!!.contains("/api/")) webRequests.incrementAndGet()
                val result = when {
                    request.path!!.endsWith("/api/authorize.json") -> """{"token":"temporary","need2FA":true}"""
                    request.path!!.endsWith("2fa/authorize.json") -> """{"token":"test-token","need2FA":false,"user":{"username":"android-test","defaultAccountId":"a"}}"""
                    request.path!!.contains("tokens/refresh") -> """{"user":{"username":"android-test","defaultAccountId":"a"}}"""
                    request.path!!.contains("accounts/list") -> """[{"id":"a","name":"测试现金","type":1,"currency":"CNY","hidden":false}]"""
                    request.path!!.contains("categories/list") -> """{"2":[{"id":"p","name":"支出","subCategories":[{"id":"c","name":"餐饮","type":2}]}]}"""
                    request.path!!.contains("tags/list") -> "[]"
                    request.path!!.contains("llm/get") -> """{"enabled":true,"llmProvider":"openai","openAIModelID":"test-model","openAIAPIKeySet":true}"""
                    request.path!!.contains("llm/update") -> {
                        val data = org.json.JSONObject(request.body.readUtf8())
                        Assert.assertEquals("test-model-updated", data.getString("openAIModelID"))
                        Assert.assertEquals("", data.getString("openAIAPIKey"))
                        settingsSaves.incrementAndGet()
                        data.toString()
                    }
                    request.path!!.contains("conversations/respond") -> {
                        val body = org.json.JSONObject(request.body.readUtf8())
                        Assert.assertEquals(setOf("conversationId", "messageId", "text"), body.keys().asSequence().toSet())
                        val text = body.getString("text")
                        if(!text.contains("不完整")) {
                            saves.incrementAndGet()
                            lastAmount.set(if(text.contains("18.25")) 1825 else 3050)
                        }
                        org.json.JSONObject().put("messageId", body.getString("messageId")).put("status", "completed")
                            .put("reply", if(text.contains("不完整")) "这笔午饭花了多少钱？" else "已记账").toString()
                    }
                    request.path!!.contains("transactions/get") -> org.json.JSONObject().put("id", "test-1").put("type", 3).put("sourceAmount", lastAmount.get()).put("sourceAccountId", "a").toString()
                    request.path!!.contains("recognize_transaction_text") -> if(request.body.readUtf8().contains("不完整")) """{"type":3,"categoryId":"c"}""" else """{"type":3,"sourceAmount":3050,"categoryId":"c"}"""
                    request.path!!.contains("generate_title") -> {
                        val data = org.json.JSONObject(request.body.readUtf8())
                        Assert.assertEquals("午饭30.5元", data.getString("text"))
                        Assert.assertEquals(java.time.LocalDate.now().toString(), data.getString("date"))
                        titleRequests.incrementAndGet()
                        if(failTitle.get()) return MockResponse().setResponseCode(503)
                        org.json.JSONObject().put("title", title).toString()
                    }
                    request.path!!.contains("transactions/add") -> { lastAmount.set(org.json.JSONObject(request.body.readUtf8()).getLong("sourceAmount")); saves.incrementAndGet(); """{"id":"test-1"}""" }
                    else -> return MockResponse().setResponseCode(404)
                }
                return MockResponse().setBody("""{"success":true,"result":$result}""")
            }
        }
        server.start()
    }
    @After fun close() { server.shutdown() }
    private fun awaitText(text: String) {
        compose.waitUntil(15000) { compose.onAllNodesWithText(text, substring = true).fetchSemanticsNodes().isNotEmpty() }
    }
    private fun awaitEnabled(text: String) {
        compose.waitUntil(15000) { compose.onAllNodes(hasText(text) and isEnabled()).fetchSemanticsNodes().isNotEmpty() }
    }
    private fun capture(name: String, dialog: Boolean = false) {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        File(context.getExternalFilesDir(null), name).outputStream().use {
            (if(dialog) compose.onNode(isDialog()) else compose.onRoot()).captureToImage().asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it)
        }
    }
    @Test fun loginAutoSaveConversationalClarificationAndRecreation() {
        awaitEnabled("服务地址")
        compose.onNodeWithText("服务地址").performTextReplacement(server.url("/test/").toString())
        compose.onNodeWithText("用户名或邮箱").performTextInput("android-test")
        compose.onNodeWithText("密码").performTextInput("test-password")
        compose.onNodeWithText("登录").performScrollTo().performClick()
        awaitText("账号已开启双重验证")
        awaitEnabled("验证码")
        compose.onNodeWithText("验证码").performTextInput("123456")
        compose.onNodeWithText("验证并登录").performScrollTo().performClick()
        compose.onNodeWithContentDescription("返回账本").assertDoesNotExist()
        awaitText("今天记点什么")
        awaitEnabled("午饭 30 元")
        compose.onNodeWithText("午饭 30 元").performTextInput("午饭30.5元")
        compose.onNodeWithContentDescription("发送").performClick()
        awaitText("已记账")
        awaitEnabled("午饭 30 元")
        Assert.assertEquals(1, saves.get())
        compose.onNodeWithContentDescription("对话列表").performClick()
        awaitText(title)
        compose.onNodeWithContentDescription("返回").performClick()
        compose.activityRule.scenario.recreate()
        awaitText("已记账")
        awaitEnabled("午饭 30 元")
        Assert.assertEquals(1, saves.get())
        awaitEnabled("午饭 30 元")
        compose.onNodeWithText("午饭 30 元").performTextInput("不完整，午饭")
        compose.onNodeWithContentDescription("发送").performClick()
        awaitText("这笔午饭花了多少钱")
        Assert.assertEquals(1, saves.get())
        awaitEnabled("午饭 30 元")
        compose.onNodeWithText("午饭 30 元").performTextInput("18.25元")
        compose.onNodeWithContentDescription("发送").performClick()
        try { compose.waitUntil(15000) { saves.get() == 2 } } catch(e: Exception) { throw AssertionError(compose.onRoot().printToString(), e) }
        awaitText("18.25")
        awaitEnabled("午饭 30 元")
        compose.onNodeWithText("已连接内网服务").assertDoesNotExist()
        capture("simplified-app-chat.png")
        compose.onNodeWithContentDescription("新增对话").performClick()
        awaitText("今天记点什么")
        compose.onNodeWithText("午饭30.5元").assertDoesNotExist()
        compose.activityRule.scenario.recreate()
        awaitText("今天记点什么")
        awaitEnabled("午饭 30 元")
        compose.onNodeWithContentDescription("对话列表").performClick()
        awaitText(title)
        Assert.assertEquals(1, titleRequests.get())
        capture("conversation-list.png")
        compose.onNodeWithText(title).performClick()
        awaitText("已记账")
        awaitEnabled("午饭 30 元")
        awaitEnabled("午饭 30 元")
        compose.onNodeWithContentDescription("对话列表").performClick()
        compose.onNodeWithContentDescription("删除对话：$title").performClick()
        compose.onNodeWithText("已保存的账目不会删除", substring = true).assertExists()
        capture("conversation-delete.png", dialog = true)
        compose.onNodeWithText("取消").performClick()
        compose.onNodeWithText(title).assertExists()
        compose.onNodeWithContentDescription("删除对话：$title").performClick()
        compose.onNodeWithText("删除对话", substring = false).performClick()
        compose.waitUntil(15000) { compose.onAllNodesWithText(title).fetchSemanticsNodes().isEmpty() }
        compose.onNodeWithContentDescription("删除对话：新对话").performClick()
        compose.onNodeWithText("删除对话", substring = false).performClick()
        awaitEnabled("新增对话")
        compose.onNodeWithContentDescription("返回").performClick()
        awaitText("今天记点什么")
        compose.activityRule.scenario.recreate()
        awaitText("今天记点什么")
        awaitEnabled("午饭 30 元")
        compose.onNodeWithContentDescription("对话列表").performClick()
        compose.onNodeWithText(title).assertDoesNotExist()
        compose.onNodeWithText("新对话", substring = false).assertExists()
        compose.onNodeWithContentDescription("返回").performClick()
        Assert.assertEquals(2, saves.get())
        failTitle.set(true)
        compose.onNodeWithText("午饭 30 元").performTextInput("午饭30.5元")
        compose.onNodeWithContentDescription("发送").performClick()
        awaitText("已记账")
        awaitEnabled("午饭 30 元")
        awaitEnabled("午饭 30 元")
        compose.waitUntil(15000) { titleRequests.get() == 2 }
        compose.onNodeWithContentDescription("对话列表").performClick()
        awaitText(ConversationTitles.fallback(System.currentTimeMillis()))
        compose.onNodeWithContentDescription("返回").performClick()
        compose.onNodeWithText("已记账").assertExists()
        compose.onNodeWithText("服务返回异常", substring = true).assertDoesNotExist()
        Assert.assertEquals(3, saves.get())
        compose.onNodeWithContentDescription("设置").performClick()
        compose.onNodeWithText("账号与服务").assertIsDisplayed()
        compose.onNodeWithText("记账设置").assertIsDisplayed()
        compose.onNodeWithText("本机数据").assertIsDisplayed()
        capture("simplified-settings.png")
        compose.onNodeWithText("AI 设置").performClick()
        awaitEnabled("模型 ID")
        compose.onNodeWithText("模型 ID").performTextReplacement("test-model-updated")
        compose.onNodeWithText("保存").performScrollTo().performClick()
        awaitEnabled("退出登录")
        Assert.assertEquals(1, settingsSaves.get())
        Assert.assertEquals(0, webRequests.get())
        failRefresh.set(true)
        compose.onNodeWithText("重新连接").performClick()
        awaitEnabled("重试")
        compose.onAllNodesWithText("重试").assertCountEquals(1)
        failRefresh.set(false)
        compose.onNodeWithText("重试").performClick()
        awaitEnabled("退出登录")
        compose.onNodeWithText("重试").assertDoesNotExist()
        compose.onNodeWithText("退出登录").performClick()
        awaitEnabled("切换账号")
        compose.onNodeWithText("服务地址").assertDoesNotExist()
        compose.onNodeWithText("android-test").assertIsDisplayed()
        capture("simplified-login.png")
        compose.onNodeWithText("切换账号").performClick()
        compose.onNodeWithText("修改").performClick()
        compose.onNode(hasText("用户名或邮箱") and hasSetTextAction() and hasAnyAncestor(isDialog())).performTextReplacement("edited-user")
        compose.onNodeWithText("保存修改").performClick()
        compose.onNode(hasText("edited-user") and hasAnyAncestor(isDialog())).assertExists()
        compose.onNodeWithText("使用").performClick()
        compose.onNodeWithText("edited-user").assertIsDisplayed()
        compose.activityRule.scenario.recreate()
        awaitEnabled("切换账号")
        compose.onNodeWithText("edited-user").assertIsDisplayed()
        compose.onNodeWithText("密码").performTextInput("must-clear")
        compose.onNodeWithText("修改服务地址").performClick()
        compose.onNodeWithText("服务地址").assertIsDisplayed()
        compose.onNodeWithText("密码").assert(hasText(""))
        compose.onNodeWithText("切换账号").performClick()
        compose.onNodeWithText("删除").performClick()
        compose.onNodeWithText("删除记录").performClick()
        compose.onNodeWithText("完成").performClick()
        compose.onNodeWithText("用户名或邮箱").assert(hasText(""))
        compose.activityRule.scenario.recreate()
        awaitEnabled("服务地址")
        compose.onNodeWithText("切换账号").assertDoesNotExist()
    }
}
