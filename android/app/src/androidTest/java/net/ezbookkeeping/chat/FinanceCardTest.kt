package net.ezbookkeeping.chat

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.test.platform.app.InstrumentationRegistry
import org.json.JSONObject
import org.junit.Rule
import org.junit.Test
import java.io.File

class FinanceCardTest {
    @get:Rule val compose = createComposeRule()
    @Test fun queryDetailsCanExpandAndCollapse() {
        val data = JSONObject().put("kind", "finance").put("details", "12:30 · 30.00 CNY\n餐饮 · 测试现金\n午餐\n\n18:20 · 20.00 CNY\n餐饮 · 测试现金\n晚餐").put("expanded", false)
        compose.setContent {
            BookkeepingTheme {
                Surface(Modifier.fillMaxSize()) {
                    Column(Modifier.padding(16.dp)) {
                        MessageItem(ChatMessage("query", "test", text = "今日支出", state = "answered", reply = "今日支出\n合计 50.00 CNY\n共 2 笔\n时区：Asia/Shanghai · 全部账户（含隐藏）", draft = data.toString()), false, {}, {})
                    }
                }
            }
        }
        compose.onNodeWithText("午餐", substring = true).assertDoesNotExist()
        compose.onNodeWithText("查看明细").performClick()
        compose.onNodeWithText("午餐", substring = true).assertIsDisplayed()
        val app = InstrumentationRegistry.getInstrumentation().targetContext
        File(app.getExternalFilesDir(null), "finance-query.png").outputStream().use {
            compose.onRoot().captureToImage().asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it)
        }
        compose.onNodeWithText("收起详情").performClick()
        compose.onNodeWithText("午餐", substring = true).assertDoesNotExist()
    }
}
