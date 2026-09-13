package net.ezbookkeeping.chat

import org.json.JSONArray
import org.json.JSONObject
import java.math.BigDecimal
import java.time.Instant
import java.time.ZoneId
import java.util.UUID

data class Choice(val id: String, val name: String, val type: Int = 0, val parent: String = "0", val currency: String = "")
data class Catalog(val accounts: List<Choice> = emptyList(), val categories: List<Choice> = emptyList(), val tags: List<Choice> = emptyList(), val defaultAccount: String = "")
data class Draft(val type: Int = 3, val amount: Long? = null, val account: String = "", val category: String = "", val time: Long = Instant.now().epochSecond, val tags: List<String> = emptyList(), val comment: String = "", val session: String = UUID.randomUUID().toString()) {
    fun json(): JSONObject = JSONObject().put("type", type).put("sourceAmount", amount ?: JSONObject.NULL)
        .put("sourceAccountId", account).put("categoryId", category).put("time", time)
        .put("utcOffset", ZoneId.systemDefault().rules.getOffset(Instant.ofEpochSecond(time)).totalSeconds / 60)
        .put("destinationAccountId", "0").put("destinationAmount", 0).put("hideAmount", false)
        .put("tagIds", JSONArray(tags)).put("pictureIds", JSONArray()).put("comment", comment).put("clientSessionId", session)
    companion object {
        fun from(j: JSONObject) = Draft(j.optInt("type", 3), if(j.has("sourceAmount") && !j.isNull("sourceAmount")) j.getLong("sourceAmount") else null,
            j.optString("sourceAccountId"), j.optString("categoryId"), j.optLong("time", Instant.now().epochSecond),
            j.optJSONArray("tagIds").strings(), j.optString("comment"), j.optString("clientSessionId").ifBlank { UUID.randomUUID().toString() })
    }
}
fun JSONArray?.strings(): List<String> = if(this == null) emptyList() else (0 until length()).map { getString(it) }
object TransactionRules {
    fun recognize(r: JSONObject, text: String, c: Catalog): Pair<Draft, List<String>> {
        require(listOf("type", "time", "categoryId", "sourceAccountId", "destinationAccountId", "sourceAmount", "destinationAmount").any { r.has(it) && !r.isNull(it) } || (r.optJSONArray("tagIds")?.length() ?: 0) > 0) { "没有识别到交易信息，请描述一笔支出或收入" }
        val explicit = r.optInt("type", 0)
        require(explicit in listOf(0, 2, 3)) { "首版仅支持支出和收入，请每次输入一笔交易" }
        val recognizedCategory = c.categories.find { it.id == r.optString("categoryId") }
            ?: c.categories.find { it.parent == r.optString("categoryId") }
        val inferred = recognizedCategory?.type?.plus(1) ?: 0
        require(inferred in listOf(0, 2, 3)) { "首版暂不支持转账" }
        val type = if(explicit != 0) explicit else if(inferred != 0) inferred else 3
        val categories = c.categories.filter { it.type == type - 1 }
        val category = categories.find { it.id == r.optString("categoryId") }
            ?: categories.find { it.parent == r.optString("categoryId") }
            ?: categories.firstOrNull()
        val account = c.accounts.find { it.id == r.optString("sourceAccountId") }
            ?: c.accounts.find { it.id == c.defaultAccount } ?: c.accounts.firstOrNull()
        val d = Draft.from(r).copy(type = type, category = category?.id ?: "", account = account?.id ?: "",
            tags = r.optJSONArray("tagIds").strings().filter { id -> c.tags.any { it.id == id } }, comment = r.optString("comment").ifBlank { text })
        val missing = validate(d, c).toMutableList()
        if(explicit == 0 && inferred == 0) missing.add("交易类型")
        if(r.optString("categoryId").isNotBlank() && recognizedCategory?.type != type - 1 && categories.none { it.parent == r.optString("categoryId") }) missing.add("分类")
        return d to missing.distinct()
    }
    fun validate(d: Draft, c: Catalog): List<String> = buildList {
        if(d.type !in listOf(2,3)) add("交易类型")
        if(d.amount == null || d.amount == 0L) add("金额")
        if(c.accounts.none { it.id == d.account }) add("账户")
        if(c.categories.none { it.id == d.category && it.type == d.type - 1 }) add("分类")
    }
    fun parseAmount(text: String): Long = BigDecimal(text.trim()).movePointRight(2).longValueExact()
    fun amount(value: Long?): String = value?.let { BigDecimal.valueOf(it, 2).toPlainString() } ?: ""
    fun summary(d: Draft, c: Catalog): String = "${if(d.type == 2) "收入" else "支出"} ${amount(d.amount)} ${c.accounts.find { it.id == d.account }?.currency.orEmpty()}\n" +
        "${c.categories.find { it.id == d.category }?.name.orEmpty()} · ${c.accounts.find { it.id == d.account }?.name.orEmpty()}\n" +
        java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm").withZone(ZoneId.systemDefault()).format(Instant.ofEpochSecond(d.time)) +
        if(d.comment.isBlank()) "" else "\n${d.comment}"
}
