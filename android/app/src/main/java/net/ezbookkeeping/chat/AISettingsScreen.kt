package net.ezbookkeeping.chat

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import org.json.JSONObject

data class AIProvider(val id: String, val title: String, val model: String, val key: String? = null, val url: String? = null)
val aiProviders = listOf(
    AIProvider("openai", "OpenAI", "openAIModelID", "openAIAPIKey"),
    AIProvider("openai_compatible", "OpenAI 兼容服务", "openAICompatibleModelID", "openAICompatibleAPIKey", "openAICompatibleBaseURL"),
    AIProvider("openrouter", "OpenRouter", "openRouterModelID", "openRouterAPIKey"),
    AIProvider("ollama", "Ollama", "ollamaModelID", url = "ollamaServerURL"),
    AIProvider("google_ai", "Google AI", "googleAIModelID", "googleAIAPIKey")
)

// Preserve settings for other providers and advanced connection options. Blank keys
// mean "keep the stored key" in the API; never copy secret-presence flags as keys.
fun aiSettingsRequest(original: JSONObject, enabled: Boolean, provider: AIProvider, model: String, url: String, apiKey: String): JSONObject =
    JSONObject(original.toString()).apply {
        remove("updatedUnixTime")
        aiProviders.mapNotNull { it.key }.forEach { remove(it + "Set"); put(it, "") }
        put("enabled", enabled)
        put("llmProvider", provider.id)
        put(provider.model, model.trim())
        provider.url?.let { put(it, url.trim()) }
        provider.key?.let { put(it, apiKey.trim()) }
    }

@Composable fun AISettingsScreen(vm: ChatViewModel, busy: Boolean, close: () -> Unit) {
    val loaded by vm.aiSettings.collectAsStateWithLifecycle()
    var original by remember { mutableStateOf<JSONObject?>(null) }
    var enabled by remember { mutableStateOf(false) }
    var provider by remember { mutableStateOf(aiProviders[0]) }
    var model by remember { mutableStateOf("") }
    var url by remember { mutableStateOf("") }
    // API keys are held only in composition memory, never in saved instance state.
    var apiKey by remember { mutableStateOf("") }
    var choosing by remember { mutableStateOf(false) }
    var validation by remember { mutableStateOf("") }
    LaunchedEffect(Unit) { vm.aiSettings.value = null; vm.loadAISettings() }
    LaunchedEffect(loaded) {
        loaded?.let {
            original = it
            enabled = it.optBoolean("enabled")
            provider = aiProviders.find { p -> p.id == it.optString("llmProvider") } ?: aiProviders[0]
            model = it.optString(provider.model)
            url = provider.url?.let { field -> it.optString(field) }.orEmpty()
            apiKey = ""
        }
    }
    BackHandler { if(!busy) close() }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = close, enabled = !busy) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") }
            Text("AI 设置", style = MaterialTheme.typography.headlineSmall)
        }
        if(original == null) {
            if(busy) CircularProgressIndicator()
            else Button(onClick = vm::loadAISettings) { Text("重新加载") }
        } else {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text("启用 AI 记账", Modifier.weight(1f))
                Switch(enabled, { enabled = it }, enabled = !busy)
            }
            Box {
                OutlinedButton(onClick = { choosing = true }, enabled = !busy, modifier = Modifier.fillMaxWidth()) { Text(provider.title) }
                DropdownMenu(expanded = choosing, onDismissRequest = { choosing = false }) {
                    aiProviders.forEach { option ->
                        DropdownMenuItem(text = { Text(option.title) }, onClick = {
                            // Keep unsaved non-secret edits when moving between providers.
                            original = aiSettingsRequest(original!!, enabled, provider, model, url, "")
                            provider = option
                            model = original!!.optString(option.model)
                            url = option.url?.let { original!!.optString(it) }.orEmpty()
                            apiKey = ""; validation = ""; choosing = false
                        })
                    }
                }
            }
            provider.url?.let {
                AppField(url, { url = it }, label = { Text("模型服务地址") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy)
            }
            AppField(model, { model = it }, label = { Text("模型 ID") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy)
            provider.key?.let { field ->
                AppField(apiKey, { apiKey = it }, label = { Text("API Key") }, modifier = Modifier.fillMaxWidth(), singleLine = true, enabled = !busy,
                    visualTransformation = PasswordVisualTransformation(),
                    supportingText = { Text(if(provider.url != null && url.trim() != loaded?.optString(provider.url)) "更换服务地址时，请填写该服务的密钥" else if(loaded?.optBoolean(field + "Set") == true) "已配置；留空保留现有密钥" else "填写服务商提供的密钥") })
            }
            if(validation.isNotEmpty()) Text(validation, color = MaterialTheme.colorScheme.error)
            Button(onClick = {
                validation = when {
                    enabled && model.isBlank() -> "请填写模型 ID"
                    enabled && provider.url != null && runCatching { Api.normalize(url) }.isFailure -> "请填写有效的模型服务地址"
                    else -> ""
                }
                if(validation.isEmpty()) vm.saveAISettings(aiSettingsRequest(original!!, enabled, provider, model, url, apiKey)) { apiKey = ""; close() }
            }, enabled = !busy, modifier = Modifier.fillMaxWidth()) { Text(if(busy) "正在保存…" else "保存") }
        }
    }
}
