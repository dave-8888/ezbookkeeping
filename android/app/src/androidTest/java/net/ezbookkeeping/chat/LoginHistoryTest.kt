package net.ezbookkeeping.chat

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.After
import org.junit.Before
import org.junit.Test
import org.junit.Assert.*
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class LoginHistoryTest {
    private val context: Context = ApplicationProvider.getApplicationContext()
    @Before fun reset() { context.getSharedPreferences("login_history", Context.MODE_PRIVATE).edit().clear().commit() }
    @After fun clear() = reset()
    @Test fun migrationSurvivesReloadAndDeletionDoesNotResurrect() {
        val store = LoginHistoryStore(context)
        store.migratePreviousLogin("http://192.168.1.3:18888", "alice")
        val saved = store.profiles().single()
        assertEquals("http://192.168.1.3:18888/", saved.base)
        assertEquals(saved, LoginHistoryStore(context).profiles().single())
        store.delete(saved.id)
        LoginHistoryStore(context).migratePreviousLogin(saved.base, saved.username)
        assertTrue(store.profiles().isEmpty())
    }
    @Test fun deduplicatesSuccessfulLoginAndMergesEditedDuplicates() {
        val store = LoginHistoryStore(context)
        val alice = store.remember("http://192.168.1.3:18888", " alice ")
        val same = store.remember("http://192.168.1.3:18888/", "alice")
        assertEquals(alice.id, same.id)
        assertEquals(1, store.profiles().size)
        val bob = store.remember("http://192.168.1.5:8080/", "bob")
        assertEquals(2, store.profiles().size)
        val edited = store.edit(bob.id, alice.base, "alice")
        assertEquals(listOf(edited), LoginHistoryStore(context).profiles())
    }
    @Test fun invalidEditPreservesRecordAndNoCredentialsAreSerialized() {
        val store = LoginHistoryStore(context)
        val saved = store.remember("https://example.test/finance/", "alice@example.test")
        try { store.edit(saved.id, "not-a-url", "alice"); fail("Invalid URL accepted") } catch(_: IllegalArgumentException) { }
        try { store.edit(saved.id, saved.base, " "); fail("Blank username accepted") } catch(_: IllegalArgumentException) { }
        assertEquals(saved, store.profiles().single())
        val json = context.getSharedPreferences("login_history", Context.MODE_PRIVATE).getString("profiles", "")!!
        assertFalse(json.contains("password")); assertFalse(json.contains("token"))
    }
}
