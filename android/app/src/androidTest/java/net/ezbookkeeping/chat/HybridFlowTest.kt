package net.ezbookkeeping.chat

import android.webkit.WebView
import android.view.View
import android.view.ViewGroup
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import okhttp3.mockwebserver.*
import org.junit.*
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicReference

class HybridFlowTest {
    @get:Rule val compose = createAndroidComposeRule<MainActivity>()
    private lateinit var server: MockWebServer
    private val refreshAuth = AtomicReference("")
    @Before fun setup() {
        server = MockWebServer()
        server.dispatcher = object: Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                if(request.path == "/test/mobile") return MockResponse().setHeader("Content-Type", "text/html").setBody("""
                    <meta name="viewport" content="width=device-width,initial-scale=1"><body><h1 id="name"></h1><input id="draft"><button id="chat" onclick="FinanciallyFreeNative.postMessage(JSON.stringify({type:'chat'}))">对话记账</button><iframe src="frame.html"></iframe>
                    <script>
                    window.currentToken = '';
                    FinanciallyFreeNative.onmessage = function(e) {
                        var m = JSON.parse(e.data);
                        if(m.type === 'session') {
                            currentToken = m.token;
                            document.getElementById('name').textContent = m.user.username;
                            FinanciallyFreeNative.postMessage(JSON.stringify({type:'token', token:'web-token'}));
                            FinanciallyFreeNative.postMessage(JSON.stringify({type:'ready'}));
                        }
                        if(m.type === 'resume') currentToken = m.token;
                        if(m.type === 'back') FinanciallyFreeNative.postMessage(JSON.stringify({type:'exit'}));
                    };
                    FinanciallyFreeNative.postMessage(JSON.stringify({type:'session'}));
                    </script></body>
                """.trimIndent())
                if(request.path == "/test/frame.html") return MockResponse().setHeader("Content-Type", "text/html").setBody("""
                    <script>FinanciallyFreeNative.postMessage('{"type":"logout"}');FinanciallyFreeNative.postMessage('{"type":"token","token":"evil"}');</script>
                """)
                val result = when {
                    request.path!!.endsWith("authorize.json") -> """{"token":"initial","user":{"username":"hybrid-test","defaultAccountId":"a"}}"""
                    request.path!!.contains("tokens/refresh") -> {
                        refreshAuth.set(request.getHeader("Authorization"))
                        """{"newToken":"native-refreshed","user":{"username":"hybrid-test","defaultAccountId":"a"}}"""
                    }
                    request.path!!.contains("accounts/list") -> """[{"id":"a","name":"现金","type":1,"currency":"CNY"}]"""
                    request.path!!.contains("categories/list") -> "{}"
                    request.path!!.contains("tags/list") -> "[]"
                    else -> return MockResponse().setResponseCode(404)
                }
                return MockResponse().setHeader("Content-Type", "application/json").setBody("""{"success":true,"result":$result}""")
            }
        }
        server.start()
    }
    @After fun close() { server.shutdown() }
    private fun enabled(text: String) {
        compose.waitUntil(15000) { compose.onAllNodes(hasText(text) and isEnabled()).fetchSemanticsNodes().isNotEmpty() }
    }
    private fun findWeb(view: View): WebView? {
        if(view is WebView) return view
        if(view is ViewGroup) for(i in 0 until view.childCount) findWeb(view.getChildAt(i))?.let { return it }
        return null
    }
    private fun js(script: String): String {
        val latch = CountDownLatch(1)
        var value = ""
        compose.runOnIdle {
            findWeb(compose.activity.window.decorView)!!.evaluateJavascript(script) { value = it; latch.countDown() }
        }
        Assert.assertTrue(latch.await(5, TimeUnit.SECONDS))
        return value
    }
    @Test fun sessionSwitchingPreservesDraftRejectsFrameAndLogsOut() {
        enabled("服务地址")
        compose.onNodeWithText("服务地址").performTextReplacement(server.url("/test/").toString())
        compose.onNodeWithText("用户名或邮箱").performTextReplacement("hybrid-test")
        compose.onNodeWithText("密码").performTextInput("password")
        compose.onNodeWithText("登录").performScrollTo().performClick()
        compose.waitUntil(15000) { compose.onAllNodesWithText("登录").fetchSemanticsNodes().isEmpty() }
        compose.waitUntil(15000) { compose.onAllNodesWithText("正在打开账本…").fetchSemanticsNodes().isEmpty() }
        compose.onNodeWithText("财神").assertDoesNotExist()
        compose.onNodeWithContentDescription("设置").assertDoesNotExist()
        Assert.assertEquals("\"hybrid-test\"", js("document.getElementById('name').textContent"))
        js("document.getElementById('draft').value = '未完成的表单'")
        compose.onNodeWithText("对话记账").assertDoesNotExist()
        js("document.getElementById('chat').click()")
        enabled("午饭 30 元")
        compose.onNodeWithContentDescription("设置").performClick()
        compose.onNodeWithText("账号与服务").assertIsDisplayed()
        compose.onNodeWithContentDescription("返回").performClick()
        enabled("午饭 30 元")
        Assert.assertEquals("Bearer web-token", refreshAuth.get())
        compose.onNodeWithContentDescription("返回账本").performClick()
        compose.waitForIdle()
        Assert.assertEquals("\"未完成的表单\"", js("document.getElementById('draft').value"))
        Assert.assertEquals("\"native-refreshed\"", js("currentToken"))
        js("FinanciallyFreeNative.postMessage('{\"type\":\"logout\"}')")
        enabled("切换账号")
        compose.onNodeWithText("服务地址").assertDoesNotExist()
        compose.onNodeWithText("账本").assertDoesNotExist()
    }
}
