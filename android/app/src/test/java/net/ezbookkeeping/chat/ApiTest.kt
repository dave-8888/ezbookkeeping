package net.ezbookkeeping.chat

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.util.concurrent.TimeUnit

class ApiTest {
    @Test fun subpathAuthAndHeaders() {
        MockWebServer().use { s ->
            s.enqueue(MockResponse().setBody("""{"success":true,"result":{"token":"test"}}"""))
            Api(s.url("/finance/").toString()).objectCall("authorize.json", JSONObject().put("loginName", "tester").put("password", "test-only"))
            val r = s.takeRequest()
            assertEquals("/finance/api/authorize.json", r.path)
            assertEquals("tester", JSONObject(r.body.readUtf8()).getString("loginName"))
            assertNotNull(r.getHeader("X-Timezone-Offset")); assertNotNull(r.getHeader("X-Timezone-Name"))
            assertNull(r.getHeader("Authorization"))
        }
    }
    @Test fun transactionWireFormatUsesBearerAndSession() {
        MockWebServer().use { s ->
            s.enqueue(MockResponse().setBody("""{"success":true,"result":{"id":"99"}}"""))
            val draft = Draft(amount = 3050, account = "a", category = "c")
            val result = Api(s.url("/").toString(), "sample-token").objectCall("v1/transactions/add.json", draft.json())
            assertEquals("99", result.getString("id"))
            val r = s.takeRequest(); val body = JSONObject(r.body.readUtf8())
            assertEquals("Bearer sample-token", r.getHeader("Authorization"))
            assertEquals(3050L, body.getLong("sourceAmount")); assertEquals(draft.session, body.getString("clientSessionId"))
        }
    }
    @Test fun applicationErrorPropagates() {
        MockWebServer().use { s ->
            s.enqueue(MockResponse().setResponseCode(401).setBody("""{"success":false,"errorCode":202003,"errorMessage":"expired"}"""))
            try { Api(s.url("/").toString()).call("anything"); fail() } catch(e: ApiError) { assertEquals(202003, e.code) }
        }
    }
    @Test fun redirectsAreNotFollowed() {
        MockWebServer().use { s ->
            s.enqueue(MockResponse().setResponseCode(302).setHeader("Location", s.url("/other")))
            try { Api(s.url("/").toString(), "token").call("anything"); fail() } catch(_: ApiError) { }
            assertEquals(1, s.requestCount)
        }
    }
    @Test fun serverFailureDoesNotRetry() {
        MockWebServer().use { s ->
            s.enqueue(MockResponse().setResponseCode(503).setBody("unavailable"))
            try { Api(s.url("/").toString()).objectCall("v1/transactions/add.json", Draft().json()); fail() } catch(_: ApiError) { }
            assertEquals(1, s.requestCount)
        }
    }
    @Test fun addressNormalizationRetainsSubpath() { assertEquals("http://192.168.1.2:8080/finance/", Api.normalize("http://192.168.1.2:8080/finance")) }
    @Test(expected = IllegalArgumentException::class) fun credentialsInUrlRejected() { Api.normalize("http://a:b@localhost/") }
}
