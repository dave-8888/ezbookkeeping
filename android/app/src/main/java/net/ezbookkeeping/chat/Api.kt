package net.ezbookkeeping.chat

import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.json.JSONObject
import org.json.JSONArray
import java.io.IOException
import java.time.ZonedDateTime
import java.util.concurrent.TimeUnit

class ApiError(val code: Int, message: String, val httpStatus: Int = 0): IOException(message)
class Api(val base: String, var token: String = "", private val client: OkHttpClient = OkHttpClient.Builder()
    .connectTimeout(10, TimeUnit.SECONDS).readTimeout(120, TimeUnit.SECONDS).callTimeout(130, TimeUnit.SECONDS)
    .retryOnConnectionFailure(false).followRedirects(false).followSslRedirects(false).build()) {
    companion object {
        fun normalize(raw: String): String {
            val u = raw.trim().toHttpUrl()
            require(u.username.isEmpty() && u.password.isEmpty() && u.query == null && u.fragment == null) { "请输入不含账号、参数的服务根地址" }
            return u.toString().trimEnd('/') + "/"
        }
    }
    fun call(path: String, data: JSONObject? = null, bearer: String = token): Any {
        val now = ZonedDateTime.now()
        val req = Request.Builder().url(base + "api/" + path).header("Accept-Language", "zh-Hans")
            .header("X-Timezone-Offset", (now.offset.totalSeconds / 60).toString()).header("X-Timezone-Name", now.zone.id)
            .header("User-Agent", "FinanciallyFree-Android/0.6")
        if(bearer.isNotEmpty()) req.header("Authorization", "Bearer $bearer")
        if(data != null) req.post(data.toString().toRequestBody("application/json".toMediaType()))
        client.newCall(req.build()).execute().use { response ->
            val body = response.body?.string().orEmpty()
            val json = try { JSONObject(body) } catch(_: Exception) { throw ApiError(response.code, "服务返回异常，请检查地址和部署子路径", response.code) }
            if(!response.isSuccessful || !json.optBoolean("success")) throw ApiError(json.optInt("errorCode", response.code), json.optString("errorMessage", "请求失败"), response.code)
            return json.opt("result") ?: JSONObject.NULL
        }
    }
    fun objectCall(path: String, data: JSONObject? = null, bearer: String = token) = call(path, data, bearer) as JSONObject
    fun conversationCall(data: JSONObject): JSONObject = Api(base, token, client.newBuilder()
        .readTimeout(9, TimeUnit.MINUTES).callTimeout(9, TimeUnit.MINUTES).build())
        .objectCall("v1/llm/conversations/respond.json", data)
    fun catalog(defaultAccount: String): Catalog {
        val accounts = mutableListOf<Choice>()
        val accountObjects = mutableMapOf<String, JSONObject>()
        fun account(a: JSONObject, parentHidden: Boolean = false) {
            accountObjects[a.getString("id")] = a
            if(parentHidden || a.optBoolean("hidden")) return
            val children = a.optJSONArray("subAccounts")
            if(children != null && children.length() > 0) for(i in 0 until children.length()) account(children.getJSONObject(i))
            else if(a.optInt("type") != 2) accounts.add(Choice(a.getString("id"), a.getString("name"), currency = a.optString("currency")))
        }
        val array = call("v1/accounts/list.json?visible_only=true") as JSONArray
        for(i in 0 until array.length()) account(array.getJSONObject(i))
        val categories = mutableListOf<Choice>()
        val groups = objectCall("v1/transaction/categories/list.json")
        for(key in groups.keys()) {
            val parents = groups.getJSONArray(key)
            for(i in 0 until parents.length()) {
                val p = parents.getJSONObject(i)
                if(p.optBoolean("hidden")) continue
                val children = p.optJSONArray("subCategories") ?: continue
                for(k in 0 until children.length()) {
                    val child = children.getJSONObject(k)
                    if(!child.optBoolean("hidden")) categories.add(Choice(child.getString("id"), p.getString("name") + " / " + child.getString("name"), child.optInt("type", key.toInt()), p.getString("id")))
                }
            }
        }
        val tagsArray = call("v1/transaction/tags/list.json") as JSONArray
        val tags = (0 until tagsArray.length()).map { tagsArray.getJSONObject(it) }.filter { !it.optBoolean("hidden") }.map { Choice(it.getString("id"), it.getString("name")) }
        val categoryOrder = listOf(1, 2, 8, 3, 4, 5, 6, 9, 7)
        val sortedAccounts = accounts.sortedWith { left, right ->
            val a = accountObjects.getValue(left.id)
            val b = accountObjects.getValue(right.id)
            val groupComparison = categoryOrder.indexOf(a.optInt("category")).compareTo(categoryOrder.indexOf(b.optInt("category")))
            if(groupComparison != 0) groupComparison else {
                val sameParent = a.optString("parentId", "0") == b.optString("parentId", "0")
                val orderA = if(sameParent) a.optInt("displayOrder") else (accountObjects[a.optString("parentId")] ?: a).optInt("displayOrder")
                val orderB = if(sameParent) b.optInt("displayOrder") else (accountObjects[b.optString("parentId")] ?: b).optInt("displayOrder")
                orderA.compareTo(orderB)
            }
        }
        return Catalog(sortedAccounts, categories, tags, defaultAccount)
    }
}
