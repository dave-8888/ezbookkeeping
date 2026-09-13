package net.ezbookkeeping.chat

import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

/** The bridge belongs only to the configured mobile entry, including its subpath. */
class LedgerPolicy(base: String) {
    val page = (Api.normalize(base) + "mobile").toHttpUrl()
    val origin = page.newBuilder().encodedPath("/").build().toString().trimEnd('/')
    fun allowsDocument(url: String?): Boolean {
        val candidate = url?.toHttpUrlOrNull() ?: return false
        return candidate.scheme == page.scheme && candidate.host == page.host && candidate.port == page.port &&
            candidate.username.isEmpty() && candidate.password.isEmpty() && candidate.encodedPath == page.encodedPath && candidate.query == null
    }
    fun url(route: String) = page.toString() + "#!" + (route.takeIf { it.startsWith('/') } ?: "/")
}
