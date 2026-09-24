package net.ezbookkeeping.chat

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.time.LocalDate
import java.time.ZoneId

class ConversationTitlesTest {
    private val created = LocalDate.of(2026, 9, 13).atStartOfDay(ZoneId.systemDefault()).toInstant().toEpochMilli()

    @Test fun sendsDateAndContentToAuthenticatedTitleEndpoint() {
        MockWebServer().use { server ->
            server.enqueue(MockResponse().setBody("""{"success":true,"result":{"title":"9月13日午餐支出"}}"""))
            val title = ConversationTitles.generate(Api(server.url("/finance/").toString(), "test-token"), created, "午饭30.5元")
            assertEquals("9月13日午餐支出", title)
            val request = server.takeRequest()
            assertEquals("/finance/api/v1/llm/conversations/generate_title.json", request.path)
            assertEquals("Bearer test-token", request.getHeader("Authorization"))
            val body = JSONObject(request.body.readUtf8())
            assertEquals("2026-09-13", body.getString("date"))
            assertEquals("午饭30.5元", body.getString("text"))
        }
    }

    @Test fun lengthIncludesDateAndNeverSplitsSupplementaryCharacters() {
        val result = ConversationTitles.normalize("9月13日" + "🚗".repeat(20), created)
        assertEquals(15, result.codePointCount(0, result.length))
        assertEquals("9月13日" + "🚗".repeat(10), result)
        assertEquals("9月13日记账", ConversationTitles.fallback(created))
        assertEquals("9月13日午餐", ConversationTitles.normalize(" 9月13日 午餐\n", created))
    }

    @Test fun invalidOrWrongDateTitleIsRejected() {
        for(value in listOf("", "午餐", "9月13日", "9月14日午餐")) {
            try { ConversationTitles.normalize(value, created); fail(value) } catch(_: IllegalArgumentException) { }
        }
    }
}
