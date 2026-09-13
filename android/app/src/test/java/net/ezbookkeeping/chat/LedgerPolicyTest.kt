package net.ezbookkeeping.chat

import org.junit.Assert.*
import org.junit.Test

class LedgerPolicyTest {
    private val policy = LedgerPolicy("https://finance.example/household/")
    @Test fun keepsSubpathAndAcceptsOnlyMobileDocument() {
        assertEquals("https://finance.example", policy.origin)
        assertEquals("https://finance.example/household/mobile#!/settings/ai", policy.url("/settings/ai"))
        assertTrue(policy.allowsDocument(policy.url("/transaction/list")))
        assertFalse(policy.allowsDocument("https://finance.example/mobile"))
        assertFalse(policy.allowsDocument("https://finance.example/other/mobile"))
        assertFalse(policy.allowsDocument("https://finance.example/household/mobile?redirect=evil"))
        assertFalse(policy.allowsDocument("https://finance.example/household/api/authorize.json"))
    }
    @Test fun rejectsDifferentOriginsCredentialsAndUnsafeSchemes() {
        listOf("http://finance.example/household/mobile", "https://finance.example:444/household/mobile",
            "https://finance.example.evil.test/household/mobile", "https://evil@finance.example/household/mobile",
            "javascript:alert(1)", "file:///household/mobile", "content://household/mobile", "about:blank")
            .forEach { assertFalse(it, policy.allowsDocument(it)) }
        assertEquals("http://localhost:18888", LedgerPolicy("http://localhost:18888/").origin)
    }
}
