package net.ezbookkeeping.chat

import android.content.res.Configuration
import androidx.compose.foundation.layout.*
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Density
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import java.io.File
import java.time.LocalDate
import java.time.ZoneId

class AppearanceTest {
    @get:Rule val compose = createComposeRule()
    private fun capture(name: String) {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        File(context.getExternalFilesDir(null), name).outputStream().use {
            compose.onRoot().captureToImage().asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it)
        }
    }
    @Test fun themeChangesKeepLayoutInputAndExpandedReceipt() {
        val mode = mutableIntStateOf(Configuration.UI_MODE_NIGHT_NO)
        val input = mutableStateOf("晚餐")
        val now = System.currentTimeMillis()
        val messages = listOf(
            ChatMessage("one", "test", text = "午饭花了30.50元", state = "saved", created = now,
                reply = "已记账\n支出 30.50 CNY\n餐饮 · 现金\n2026-09-13 11:46\n工作日午餐"),
            ChatMessage("two", "test", text = "地铁3元", state = "saved", created = now,
                reply = "已记账\n支出 3.00 CNY\n交通 · 现金\n2026-09-13 12:00")
        )
        compose.setContent {
            val config = Configuration(LocalConfiguration.current).apply {
                uiMode = (uiMode and Configuration.UI_MODE_NIGHT_MASK.inv()) or mode.intValue
            }
            CompositionLocalProvider(LocalConfiguration provides config) {
                BookkeepingTheme {
                    Surface(color = androidx.compose.material3.MaterialTheme.colorScheme.background) {
                        Column(Modifier.fillMaxSize()) {
                            ChatNavigation(false, true, {}, {})
                            ChatMessageList(messages, false, {}, {}, Modifier.weight(1f))
                            ChatComposer(input.value, false, { input.value = it }, {})
                        }
                    }
                }
            }
        }
        compose.waitForIdle()
        compose.onAllNodesWithText("对话记账").assertCountEquals(1)
        compose.onAllNodesWithText("今天").assertCountEquals(1)
        compose.onNodeWithText("工作日午餐").assertDoesNotExist()
        val dayBounds = compose.onNodeWithContentDescription("发送").fetchSemanticsNode().boundsInRoot
        capture("simplified-chat-day.png")
        compose.onAllNodesWithText("查看详情")[0].performClick()
        compose.onNodeWithText("工作日午餐").assertIsDisplayed()
        compose.runOnIdle { mode.intValue = Configuration.UI_MODE_NIGHT_YES }
        compose.waitForIdle()
        compose.onNodeWithText("工作日午餐").assertIsDisplayed()
        compose.onNodeWithText("晚餐").assertIsDisplayed()
        assertEquals(dayBounds, compose.onNodeWithContentDescription("发送").fetchSemanticsNode().boundsInRoot)
        capture("simplified-chat-expanded-night.png")
        compose.onNodeWithText("收起详情").performClick()
        compose.onNodeWithText("工作日午餐").assertDoesNotExist()
        capture("simplified-chat-night.png")
        compose.onNodeWithText("晚餐").performTextClearance()
        compose.onNodeWithContentDescription("发送").assertIsNotEnabled()
    }
    @Test fun largerTextKeepsPrimaryActionsAndReceiptReadable() {
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(LocalDensity.current.density, 1.5f)) {
                BookkeepingTheme {
                    Surface(color = androidx.compose.material3.MaterialTheme.colorScheme.background) {
                        Column(Modifier.fillMaxSize()) {
                            ChatNavigation(false, true, {}, {})
                            ChatMessageList(listOf(ChatMessage("large", "test", text = "今天工资到账8000元", state = "saved",
                                reply = "已记账\n收入 8000.00 CNY\n工资 · 银行卡\n2026-09-13 12:00")), false, {}, {}, Modifier.weight(1f))
                            ChatComposer("晚餐", false, {}, {})
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("对话记账").assertIsDisplayed()
        compose.onNodeWithText("+¥8000.00").assertIsDisplayed()
        compose.onNodeWithContentDescription("设置").assertIsDisplayed()
        compose.onNodeWithContentDescription("发送").assertIsDisplayed().assertIsEnabled()
        capture("simplified-large-text.png")
    }
    @Test fun expenseRefundShowsMoneyReturningWithoutChangingItsType() {
        compose.setContent {
            BookkeepingTheme {
                MessageItem(ChatMessage("refund", "test", text = "午饭退款30元", state = "saved",
                    reply = "已记账\n支出 -30.00 CNY\n餐饮 · 现金\n2026-09-13 12:00"), false, {}, {})
            }
        }
        compose.onNodeWithText("+¥30.00").assertIsDisplayed()
        compose.onNodeWithText("查看详情").performClick()
        compose.onNodeWithText("支出 · 2026-09-13 12:00").assertIsDisplayed()
    }
    @Test fun separateDaysAndUnknownSaveOfferReadOnlyVerification() {
        val yesterday = LocalDate.now().minusDays(1).atStartOfDay(ZoneId.systemDefault()).toInstant().toEpochMilli()
        var verified = false
        compose.setContent {
            BookkeepingTheme {
                ChatMessageList(listOf(
                    ChatMessage("one", "test", text = "午饭", state = "failed", created = yesterday, reply = "识别失败"),
                    ChatMessage("two", "test", text = "晚饭", state = "unknown", reply = "保存结果待确认，请查看账单，避免重复记账。")
                ), false, {}, { verified = true })
            }
        }
        compose.onAllNodesWithText("今天").assertCountEquals(1)
        compose.onNodeWithText(LocalDate.now().minusDays(1).format(java.time.format.DateTimeFormatter.ofPattern("yyyy年M月d日"))).assertIsDisplayed()
        compose.onNodeWithText("查看账单").performClick()
        assertTrue(verified)
        compose.onNodeWithContentDescription("发送").assertDoesNotExist()
    }
}
