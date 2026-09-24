package net.ezbookkeeping.chat

import org.json.JSONObject
import java.time.Instant
import java.time.ZoneId

object ConversationTitles {
    private fun date(created: Long) = Instant.ofEpochMilli(created).atZone(ZoneId.systemDefault()).toLocalDate()
    fun prefix(created: Long): String = date(created).let { "${it.monthValue}月${it.dayOfMonth}日" }
    fun fallback(created: Long): String = prefix(created) + "记账"
    fun limit(text: String, max: Int): String = text.substring(0, text.offsetByCodePoints(0, minOf(max, text.codePointCount(0, text.length))))

    fun generate(api: Api, created: Long, text: String): String {
        val result = api.objectCall("v1/llm/conversations/generate_title.json", JSONObject()
            .put("date", date(created).toString()).put("text", limit(text, 2000)))
        return normalize(result.getString("title"), created)
    }

    fun normalize(raw: String, created: Long): String {
        val title = raw.filterNot { it.isWhitespace() }
        val prefix = prefix(created)
        require(title.startsWith(prefix) && title.length > prefix.length) { "标题格式无效" }
        return limit(title, 15)
    }
}
