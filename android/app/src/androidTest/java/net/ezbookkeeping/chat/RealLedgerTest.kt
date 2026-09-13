package net.ezbookkeeping.chat

import android.app.UiModeManager
import android.content.Context
import android.graphics.Bitmap
import android.view.View
import android.view.ViewGroup
import android.view.KeyEvent
import android.view.MotionEvent
import android.os.SystemClock
import org.json.JSONArray
import android.view.accessibility.AccessibilityNodeInfo
import android.webkit.WebView
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.*
import java.io.File
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/** Opt-in smoke against a disposable backend seeded with the Android test account. */
class RealLedgerTest {
    @get:Rule val compose = createAndroidComposeRule<MainActivity>()
    private fun enabled(text: String) {
        compose.waitUntil(20000) { compose.onAllNodes(hasText(text) and isEnabled()).fetchSemanticsNodes().isNotEmpty() }
    }
    private fun findWeb(view: View): WebView? {
        if(view is WebView) return view
        if(view is ViewGroup) for(i in 0 until view.childCount) findWeb(view.getChildAt(i))?.let { return it }
        return null
    }
    private fun js(script: String): String {
        val latch = CountDownLatch(1); var value = "null"
        compose.runOnIdle {
            val web = findWeb(compose.activity.window.decorView)
            if(web == null) latch.countDown() else web.evaluateJavascript(script) { value = it; latch.countDown() }
        }
        Assert.assertTrue(latch.await(5, TimeUnit.SECONDS)); return value
    }
    private fun webContains(text: String) {
        try { compose.waitUntil(20000) { js("document.body.innerText.includes('${text}')") == "true" } }
        catch(e: Exception) { throw AssertionError("Web page: " + js("document.body.innerText") + "\n" + compose.onRoot().printToString(), e) }
    }
    private fun screenshot(name: String) {
        val test = InstrumentationRegistry.getInstrumentation()
        File(test.targetContext.getExternalFilesDir(null), name).outputStream().use {
            test.uiAutomation.takeScreenshot().compress(Bitmap.CompressFormat.PNG, 100, it)
        }
    }
    private fun findNode(node: AccessibilityNodeInfo?, predicate: (AccessibilityNodeInfo) -> Boolean): AccessibilityNodeInfo? {
        if(node == null) return null
        if(predicate(node)) return node
        for(i in 0 until node.childCount) findNode(node.getChild(i), predicate)?.let { return it }
        return null
    }
    private fun fileRoundTrip() {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val automation = instrumentation.uiAutomation
        val name = "hybrid-export-${System.currentTimeMillis()}.txt"
        js("""var a=document.createElement('a');a.download='$name';a.href=URL.createObjectURL(new Blob(['hybrid file test'],{type:'text/plain'}));document.body.appendChild(a);a.click();a.remove();""")
        compose.waitUntil(10000) { automation.rootInActiveWindow?.packageName?.toString()?.endsWith("documentsui") == true }
        val save = findNode(automation.rootInActiveWindow) { it.viewIdResourceName?.endsWith("action_menu_select") == true || it.text?.toString() in listOf("SAVE", "保存") }
        Assert.assertNotNull("System save button", save)
        save!!.performAction(AccessibilityNodeInfo.ACTION_CLICK)
        compose.waitUntil(10000) { automation.rootInActiveWindow?.packageName?.toString() == "net.ezbookkeeping.chat" }
        compose.waitUntil(10000) {
            android.os.ParcelFileDescriptor.AutoCloseInputStream(automation.executeShellCommand("cat /sdcard/Download/$name")).bufferedReader().use { it.readText() } == "hybrid file test"
        }
        js("""var f=document.createElement('input');f.type='file';f.accept='text/plain';f.id='hybrid-file';f.style='position:fixed;top:100px;left:10px;width:250px;height:50px;z-index:99999;background:white';f.onchange=()=>{f.files[0]?.text().then(t=>window.hybridImported=t)};document.body.appendChild(f);""")
        fun tapInput() {
            val point = JSONArray(js("""(()=>{var r=document.getElementById('hybrid-file').getBoundingClientRect();return [(r.left+r.width/2)*devicePixelRatio,(r.top+r.height/2)*devicePixelRatio]})()"""))
            val origin = IntArray(2)
            compose.runOnIdle { findWeb(compose.activity.window.decorView)!!.getLocationOnScreen(origin) }
            val time = SystemClock.uptimeMillis()
            for(action in listOf(MotionEvent.ACTION_DOWN, MotionEvent.ACTION_UP)) {
                val event = MotionEvent.obtain(time, SystemClock.uptimeMillis(), action, origin[0] + point.getDouble(0).toFloat(), origin[1] + point.getDouble(1).toFloat(), 0)
                automation.injectInputEvent(event, true); event.recycle()
            }
        }
        tapInput()
        compose.waitUntil(10000) { automation.rootInActiveWindow?.packageName?.toString()?.endsWith("documentsui") == true }
        var file: AccessibilityNodeInfo? = null
        compose.waitUntil(10000) {
            file = findNode(automation.rootInActiveWindow) { it.text?.toString() == name }
            file != null
        }
        var clickable = file!!
        while(!clickable.isClickable && clickable.parent != null) clickable = clickable.parent
        clickable.performAction(AccessibilityNodeInfo.ACTION_CLICK)
        compose.waitUntil(10000) { automation.rootInActiveWindow?.packageName?.toString() == "net.ezbookkeeping.chat" }
        compose.waitUntil(10000) { js("window.hybridImported") == "\"hybrid file test\"" }
        automation.executeShellCommand("rm /sdcard/Download/$name").close()
        // Cancelling a second selection must leave the page usable.
        tapInput()
        compose.waitUntil(10000) { automation.rootInActiveWindow?.packageName?.toString()?.endsWith("documentsui") == true }
        instrumentation.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
        compose.waitUntil(10000) { automation.rootInActiveWindow?.packageName?.toString() == "net.ezbookkeeping.chat" }
    }
    @Test fun plusMenuOpensChatAndReturnsToLedger() {
        val base = InstrumentationRegistry.getArguments().getString("hybridTestBase")
        Assume.assumeTrue("Requires disposable backend", base != null && base.startsWith("http://10.0.2.2:"))
        if(compose.onAllNodesWithText("修改服务地址").fetchSemanticsNodes().isNotEmpty()) {
            compose.onNodeWithText("修改服务地址").performClick()
        }
        enabled("服务地址")
        compose.onNodeWithText("服务地址").performTextReplacement(base!!)
        compose.onNodeWithText("用户名或邮箱").performTextReplacement("android_mvp_test")
        compose.onNodeWithText("密码").performTextInput("android-test-only-93847")
        compose.onNodeWithText("登录").performScrollTo().performClick()
        webContains("本月")
        compose.onNodeWithText("对话记账").assertDoesNotExist()
        js("document.getElementById('homepage-add-button').click()")
        webContains("对话记账")
        screenshot("chat-plus-menu.png")
        js("Array.from(document.querySelectorAll('.popover a')).find(a => a.textContent.includes('对话记账')).click()")
        enabled("午饭 30 元")
        screenshot("chat-plus-page.png")
        compose.onNodeWithText("午饭 30 元").performTextInput("尚未发送的草稿")
        compose.onNodeWithContentDescription("返回账本").performClick()
        webContains("本月")
        js("document.getElementById('homepage-add-button').click()")
        webContains("对话记账")
        js("Array.from(document.querySelectorAll('.popover a')).find(a => a.textContent.includes('对话记账')).click()")
        enabled("尚未发送的草稿")
        compose.onNodeWithText("尚未发送的草稿").performTextClearance()
        compose.runOnIdle { compose.activity.onBackPressedDispatcher.onBackPressed() }
        webContains("本月")
        js("document.getElementById('homepage-add-button').click()")
        webContains("对话记账")
        js("document.querySelector('.popover a[href=\"/transaction/add\"]').click()")
        compose.waitUntil(10000) { js("!!document.querySelector('.page-current input')") == "true" }
        compose.runOnIdle { compose.activity.onBackPressedDispatcher.onBackPressed() }
        webContains("本月")
    }

    @Test fun realMobilePagesNativeSaveRefreshAndTheme() {
        val base = InstrumentationRegistry.getArguments().getString("hybridTestBase")
        Assume.assumeTrue("Requires disposable backend", base != null && base.startsWith("http://10.0.2.2:"))
        enabled("服务地址")
        compose.onNodeWithText("服务地址").performTextReplacement(base!!)
        compose.onNodeWithText("用户名或邮箱").performTextReplacement("android_mvp_test")
        compose.onNodeWithText("密码").performTextInput("android-test-only-93847")
        compose.onNodeWithText("登录").performScrollTo().performClick()
        compose.waitUntil(15000) { compose.onAllNodesWithText("登录").fetchSemanticsNodes().isEmpty() }
        webContains("本月")
        compose.waitUntil(10000) { js("/[0-9]/.test(document.querySelector('.month-expense')?.textContent || '')") == "true" }
        val beforeAmount = js("document.querySelector('.month-expense').textContent")
        Assert.assertEquals("null", js("localStorage.getItem('ebk_user_token')"))
        screenshot("hybrid-day.png")
        js("document.querySelector('a[href=\"/account/list\"]').click()")
        webContains("测试现金")
        compose.runOnIdle { compose.activity.onBackPressedDispatcher.onBackPressed() }
        webContains("本月")
        js("document.getElementById('homepage-add-button').click()")
        webContains("对话记账")
        js("Array.from(document.querySelectorAll('.popover a')).find(a => a.textContent.includes('对话记账')).click()")
        enabled("午饭 30 元")
        compose.onNodeWithText("午饭 30 元").performTextInput("午饭30.50元")
        compose.onNodeWithContentDescription("发送").performClick()
        compose.waitUntil(20000) { compose.onAllNodesWithText("已记账", substring = true).fetchSemanticsNodes().isNotEmpty() }
        enabled("午饭 30 元")
        screenshot("hybrid-chat.png")
        compose.onNodeWithContentDescription("返回账本").performClick()
        webContains("本月")
        compose.waitUntil(10000) { js("document.querySelector('.month-expense').textContent") != beforeAmount }
        js("document.querySelector('a[href=\"/transaction/list\"]').click()")
        webContains("午饭")
        screenshot("hybrid-transactions.png")
        compose.runOnIdle { compose.activity.onBackPressedDispatcher.onBackPressed() }
        webContains("本月")
        js("document.querySelector('a[href=\"/settings\"]').click()")
        webContains("AI")
        js("document.querySelector('a[href=\"/settings/ai\"]').click()")
        webContains("模型")
        screenshot("hybrid-ai.png")
        fileRoundTrip()
        if(android.os.Build.VERSION.SDK_INT >= 31) {
            (InstrumentationRegistry.getInstrumentation().targetContext.getSystemService(Context.UI_MODE_SERVICE) as UiModeManager).setApplicationNightMode(UiModeManager.MODE_NIGHT_YES)
            webContains("本月")
            compose.waitUntil(10000) { js("matchMedia('(prefers-color-scheme: dark)').matches") == "true" }
            screenshot("hybrid-night.png")
        }
    }
}
