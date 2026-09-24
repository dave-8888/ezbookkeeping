package net.ezbookkeeping.chat

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject
import java.util.UUID

data class LoginProfile(val id: String, val base: String, val username: String, val lastUsed: Long)

/** Only server addresses and login names; never passwords, tokens or account data. */
class LoginHistoryStore(context: Context) {
    private val prefs = context.getSharedPreferences("login_history", Context.MODE_PRIVATE)
    fun profiles(): List<LoginProfile> {
        val array = try { JSONArray(prefs.getString("profiles", "[]")) } catch(_: Exception) { JSONArray() }
        return (0 until array.length()).mapNotNull { index ->
            runCatching {
                val j = array.getJSONObject(index)
                LoginProfile(j.getString("id"), j.getString("base"), j.getString("username"), j.getLong("lastUsed"))
            }.getOrNull()
        }.sortedByDescending { it.lastUsed }
    }
    private fun write(profiles: List<LoginProfile>) {
        val array = JSONArray()
        profiles.forEach { array.put(JSONObject().put("id", it.id).put("base", it.base).put("username", it.username).put("lastUsed", it.lastUsed)) }
        prefs.edit().putString("profiles", array.toString()).apply()
    }
    fun migratePreviousLogin(base: String, username: String) {
        if(prefs.getBoolean("migrated", false)) return
        // A deleted record must not reappear from the active session on the next launch.
        if(base.isNotBlank() && username.isNotBlank()) runCatching { remember(base, username) }
        prefs.edit().putBoolean("migrated", true).apply()
    }
    fun remember(base: String, username: String): LoginProfile {
        val normalized = Api.normalize(base)
        val name = username.trim()
        require(name.isNotEmpty()) { "请填写用户名或邮箱" }
        val existing = profiles()
        val previous = existing.find { it.base == normalized && it.username == name }
        val profile = LoginProfile(previous?.id ?: UUID.randomUUID().toString(), normalized, name, System.currentTimeMillis())
        write(listOf(profile) + existing.filter { it.id != profile.id })
        return profile
    }
    fun edit(id: String, base: String, username: String): LoginProfile {
        val normalized = Api.normalize(base)
        val name = username.trim()
        require(name.isNotEmpty()) { "请填写用户名或邮箱" }
        val existing = profiles()
        val previous = existing.find { it.id == id } ?: throw IllegalArgumentException("这条登录记录已被删除")
        val duplicate = existing.find { it.id != id && it.base == normalized && it.username == name }
        val edited = previous.copy(base = normalized, username = name, lastUsed = maxOf(previous.lastUsed, duplicate?.lastUsed ?: 0))
        write(listOf(edited) + existing.filter { it.id != id && it.id != duplicate?.id })
        return edited
    }
    fun delete(id: String) { write(profiles().filter { it.id != id }) }
}
