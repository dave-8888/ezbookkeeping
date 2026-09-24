package net.ezbookkeeping.chat

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class ServerConversationClientTest {
    private val message = ChatMessage("message-1", "server|tester", text = "输出我的所有资产", state = "server_pending", conversationId = "conversation-1")
    private fun response(status: String = "completed", id: String = message.id) = MockResponse().setBody(JSONObject()
        .put("success", true).put("result", JSONObject().put("messageId", id).put("status", status)
            .put("reply", "当前资产 37.42 CNY").put("details", "1. 钱包 37.42 CNY")
            .put("toolCalls", "must not persist").put("apiKey", "must not persist")).toString())

    @Test fun sendsOnlyPublicEnvelopeAndRendersVerifiedServiceResult() {
        MockWebServer().use { server ->
            server.enqueue(response())
            val result = ServerConversationClient(Api(server.url("/").toString(), "token")).send(message)
            val request = server.takeRequest()
            assertEquals("/api/v1/llm/conversations/respond.json", request.path)
            assertEquals("POST", request.method)
            val body = JSONObject(request.body.readUtf8())
            assertEquals(setOf("conversationId", "messageId", "text"), body.keys().asSequence().toSet())
            assertEquals(message.conversationId, body.getString("conversationId"))
            assertEquals(message.id, body.getString("messageId"))
            assertEquals("answered", result.state)
            assertEquals("当前资产 37.42 CNY", result.reply)
            assertFalse(result.draft!!.contains("must not persist"))
            assertEquals(1, server.requestCount)
        }
    }
    @Test fun lostResponseIsReadWithGetAndNeverResubmitsMutation() {
        MockWebServer().use { server ->
            server.enqueue(response("processing")); server.enqueue(response("unknown"))
            val client = ServerConversationClient(Api(server.url("/").toString(), "token"))
            assertEquals("server_pending", client.result(message).state)
            assertEquals("answered", client.result(message).state)
            repeat(2) {
                val request = server.takeRequest()
                assertEquals("GET", request.method)
                assertEquals("/api/v1/llm/conversations/result.json?conversation_id=conversation-1&message_id=message-1", request.path)
                assertEquals(0L, request.bodySize)
            }
        }
    }
    @Test fun mismatchedMessageCannotBePresentedAsThisResult() {
        MockWebServer().use { server ->
            server.enqueue(response(id = "another-message"))
            try { ServerConversationClient(Api(server.url("/").toString())).send(message); fail() }
            catch(_: IllegalArgumentException) { }
        }
    }
}
