package net.ezbookkeeping.chat

import android.widget.DatePicker
import android.widget.TimePicker
import androidx.compose.material3.Surface
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.test.espresso.Espresso.onView
import androidx.test.espresso.ViewAction
import androidx.test.espresso.UiController
import androidx.test.espresso.action.ViewActions.click
import androidx.test.espresso.matcher.ViewMatchers.*
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import java.io.File
import java.time.LocalDateTime
import java.time.ZoneId

class TransactionEditorUiTest {
    @get:Rule val compose = createComposeRule()
    @Test fun missingFieldsStayVisibleDetailsAndPickersPreserveDraft() {
        val initial = Draft(amount = null, account = "a", category = "c", comment = "午饭", tags = listOf("t"),
            time = LocalDateTime.of(2026, 9, 13, 12, 30).atZone(ZoneId.systemDefault()).toEpochSecond())
        val catalog = Catalog(accounts = listOf(Choice("a", "现金", currency = "CNY")),
            categories = listOf(Choice("c", "餐饮", type = 2)), tags = listOf(Choice("t", "工作日")))
        var saved: Draft? = null
        compose.setContent { BookkeepingTheme { Surface(color = androidx.compose.material3.MaterialTheme.colorScheme.background) { TransactionEditor(initial, catalog, false, {}, { saved = it }) } } }
        compose.onNodeWithText("金额 · CNY").assertIsDisplayed()
        compose.onNodeWithText("备注").assertDoesNotExist()
        compose.onNodeWithText("类型").assertDoesNotExist()
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        File(context.getExternalFilesDir(null), "simplified-editor.png").outputStream().use {
            compose.onRoot().captureToImage().asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it)
        }
        compose.onNodeWithText("日期 · 2026-09-13").performClick()
        onView(isAssignableFrom(DatePicker::class.java)).perform(object: ViewAction {
            override fun getConstraints() = isAssignableFrom(DatePicker::class.java)
            override fun getDescription() = "Choose transaction date"
            override fun perform(uiController: UiController, view: android.view.View) { (view as DatePicker).updateDate(2026, 8, 12) }
        })
        onView(withId(android.R.id.button1)).perform(click())
        compose.onNodeWithText("日期 · 2026-09-12").assertIsDisplayed()
        compose.onNodeWithText("时间 · 12:30").performClick()
        onView(isAssignableFrom(TimePicker::class.java)).perform(object: ViewAction {
            override fun getConstraints() = isAssignableFrom(TimePicker::class.java)
            override fun getDescription() = "Choose transaction time"
            override fun perform(uiController: UiController, view: android.view.View) { (view as TimePicker).apply { hour = 9; minute = 45 } }
        })
        onView(withId(android.R.id.button1)).perform(click())
        compose.onNodeWithText("时间 · 09:45").assertIsDisplayed()
        compose.onNodeWithText("更多信息").performScrollTo().performClick()
        compose.onNodeWithText("备注").performScrollTo().assert(hasText("午饭"))
        compose.onNodeWithText("工作日").performScrollTo().assertIsOn().performClick()
        compose.onNodeWithText("金额 · CNY").performScrollTo().performTextInput("18.25")
        compose.onNodeWithText("保存交易").performClick()
        compose.runOnIdle {
            assertNotNull(saved)
            assertEquals(1825L, saved!!.amount)
            assertEquals(initial.session, saved!!.session)
            assertEquals("c", saved!!.category)
            assertEquals("午饭", saved!!.comment)
            assertTrue(saved!!.tags.isEmpty())
            assertEquals(LocalDateTime.of(2026, 9, 12, 9, 45).atZone(ZoneId.systemDefault()).toEpochSecond(), saved!!.time)
        }
    }
    @Test fun inferredTypeStillRequiresExplicitReviewAndCanEditKnownFields() {
        val initial = Draft(type = 3, amount = 1200, account = "a", category = "c")
        val catalog = Catalog(accounts = listOf(Choice("a", "现金", currency = "CNY")), categories = listOf(Choice("c", "餐饮", type = 2)))
        compose.setContent { BookkeepingTheme { Surface(color = androidx.compose.material3.MaterialTheme.colorScheme.background) { TransactionEditor(initial, catalog, false, {}, {}, listOf("交易类型")) } } }
        compose.onNodeWithText("类型").assertIsDisplayed()
        compose.onNodeWithText("金额 · CNY").assertDoesNotExist()
        compose.onNodeWithText("修改已识别信息").performClick()
        compose.onNodeWithText("金额 · CNY").assertIsDisplayed()
        compose.onNodeWithText("金额 · CNY").performTextReplacement("1.234")
        compose.onNodeWithText("保存交易").performClick()
        compose.onNodeWithText("金额最多保留两位小数").assertIsDisplayed()
    }
}
