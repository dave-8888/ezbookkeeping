package net.ezbookkeeping.chat

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class StorageTest {
    @Test fun recoveryDoesNotResubmitAndOwnersStaySeparate() = runBlocking {
        val db = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), ChatDatabase::class.java).build()
        try {
            val dao = db.messages()
            dao.put(ChatMessage("1", "server|a", text = "午饭", state = "saving", draft = Draft(amount = 100).json().toString()))
            dao.put(ChatMessage("2", "server|b", text = "工资", state = "saved", transactionId = "tx-2"))
            dao.put(ChatMessage("3", "server|a", text = "晚饭", state = "recognizing"))
            dao.recoverSaves(); dao.recoverRecognition()
            val a = dao.observe("server|a").first()
            assertEquals(listOf("unknown", "failed"), a.map { it.state })
            assertNotNull(a.first().draft)
            dao.clear("server|a")
            assertTrue(dao.observe("server|a").first().isEmpty())
            assertEquals("tx-2", dao.observe("server|b").first().single().transactionId)
        } finally { db.close() }
    }
}
