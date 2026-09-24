package net.ezbookkeeping.chat

import org.json.JSONObject

/** Mobile only transports user messages and renders the service's public result. */
class ServerConversationClient(private val api: Api) {
    companion object {
        fun presentation(raw: String?): JSONObject? = try {
            raw?.let { JSONObject(it) }?.takeIf { it.optString("kind") in listOf("finance", "service_reply") }
        } catch(_: Exception) { null }
    }
    fun send(message: ChatMessage): ChatMessage = read(message, api.conversationCall(JSONObject()
        .put("conversationId", message.conversationId).put("messageId", message.id).put("text", message.text)))
    fun result(message: ChatMessage): ChatMessage = read(message, api.objectCall("v1/llm/conversations/result.json?conversation_id=${java.net.URLEncoder.encode(message.conversationId, "UTF-8")}&message_id=${java.net.URLEncoder.encode(message.id, "UTF-8")}"))
    private fun read(message: ChatMessage, response: JSONObject): ChatMessage {
        require(response.getString("messageId") == message.id) { "返回的消息不匹配，请重新查看结果。" }
        val status = response.getString("status")
        require(status in listOf("completed", "interrupted", "unknown", "processing", "busy", "not_found")) { "返回状态不正确，请重新查看结果。" }
        // Never persist or interpret model/tool protocol even if extra fields appear in a response.
        val display = JSONObject().put("kind", "service_reply").put("details", response.optString("details"))
            .put("verified", response.optString("verified")).put("expanded", response.optString("details").isNotBlank())
        return message.copy(state = if(status == "processing") "server_pending" else "answered", reply = response.getString("reply"), draft = display.toString())
    }
}
