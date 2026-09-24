package net.ezbookkeeping.chat

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class AISettingsTest {
    @Test fun updatingModelPreservesOtherProvidersAndConnectionOptionsWithoutInventingKeys() {
        val original = JSONObject("""{"enabled":true,"llmProvider":"openai_compatible","openAICompatibleBaseURL":"https://example.com/v1","openAICompatibleModelID":"old","openAICompatibleAPIKeySet":true,"openAIModelID":"other","largeLanguageModelAPIRequestTimeout":90,"largeLanguageModelAPIProxy":"system","largeLanguageModelAPISkipTLSVerify":false}""")
        val result = aiSettingsRequest(original, true, aiProviders[1], " new ", "https://example.com/v1", "")
        assertEquals("new", result.getString("openAICompatibleModelID"))
        assertEquals("other", result.getString("openAIModelID"))
        assertEquals(90, result.getInt("largeLanguageModelAPIRequestTimeout"))
        assertEquals("system", result.getString("largeLanguageModelAPIProxy"))
        assertFalse(result.getBoolean("largeLanguageModelAPISkipTLSVerify"))
        assertEquals("", result.getString("openAICompatibleAPIKey"))
        assertFalse(result.has("openAICompatibleAPIKeySet"))
        assertEquals("old", original.getString("openAICompatibleModelID"))
    }
    @Test fun providerSwitchWritesOnlySelectedKeyAndCanDisableRecognition() {
        val result = aiSettingsRequest(JSONObject(), false, aiProviders[2], "model", "", " new-secret ")
        assertEquals("openrouter", result.getString("llmProvider"))
        assertEquals("new-secret", result.getString("openRouterAPIKey"))
        assertEquals("", result.getString("openAIAPIKey"))
        assertFalse(result.getBoolean("enabled"))
    }
}
