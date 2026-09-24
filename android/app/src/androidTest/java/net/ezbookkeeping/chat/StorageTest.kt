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
    @Test fun lateTitlesCannotRestoreDeletedChatsOrReplaceSuccessfulTitles() = runBlocking {
        val db = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), ChatDatabase::class.java).build()
        try {
            val dao = db.messages()
            dao.addConversation(Conversation("chat", "owner"))
            dao.put(ChatMessage("first", "owner", text = "午餐", conversationId = "chat"))
            dao.saveGeneratedTitle("other", "chat", "first", "错误账号")
            assertFalse(dao.conversation("owner", "chat")!!.titleGenerated)
            dao.saveGeneratedTitle("owner", "chat", "first", "9月13日午餐")
            dao.setFallbackTitle("owner", "chat", "9月13日记账")
            dao.saveGeneratedTitle("owner", "chat", "first", "错误覆盖")
            assertEquals("9月13日午餐", dao.conversation("owner", "chat")!!.title)
            dao.deleteConversation("owner", "chat")
            dao.saveGeneratedTitle("owner", "chat", "first", "9月13日午餐")
            assertNull(dao.conversation("owner", "chat"))
            dao.addConversation(Conversation("empty", "owner"))
            dao.put(ChatMessage("cleared", "owner", text = "午餐", conversationId = "empty"))
            dao.clear("owner")
            dao.saveGeneratedTitle("owner", "empty", "cleared", "9月13日午餐")
            assertFalse(dao.conversation("owner", "empty")!!.titleGenerated)
        } finally { db.close() }
    }

    @Test fun versionTwoMigrationKeepsChatsAndLimitsLegacyTitles() = runBlocking {
        val context = ApplicationProvider.getApplicationContext<android.content.Context>()
        val name = "title-migration-${java.util.UUID.randomUUID()}.db"
        context.openOrCreateDatabase(name, 0, null).use { old ->
            old.execSQL("CREATE TABLE messages (id TEXT NOT NULL PRIMARY KEY, owner TEXT NOT NULL, created INTEGER NOT NULL, text TEXT NOT NULL, state TEXT NOT NULL, reply TEXT NOT NULL, draft TEXT, transactionId TEXT, conversationId TEXT NOT NULL DEFAULT '')")
            old.execSQL("CREATE INDEX index_messages_owner ON messages(owner)")
            old.execSQL("CREATE INDEX index_messages_owner_conversationId ON messages(owner, conversationId)")
            old.execSQL("CREATE TABLE conversations (id TEXT NOT NULL PRIMARY KEY, owner TEXT NOT NULL, title TEXT NOT NULL, created INTEGER NOT NULL)")
            old.execSQL("CREATE INDEX index_conversations_owner ON conversations(owner)")
            old.execSQL("INSERT INTO conversations VALUES ('chat','owner','这是一个超过十五个字的旧版对话标题需要截断',1)")
            old.execSQL("INSERT INTO messages VALUES ('1','owner',1,'午饭','saved','已记账',NULL,'tx1','chat')")
            old.version = 2
        }
        val db = Room.databaseBuilder(context, ChatDatabase::class.java, name).addMigrations(ChatDatabase.MIGRATION_2_3).build()
        try {
            val chat = db.messages().conversation("owner", "chat")!!
            assertEquals(15, chat.title.length)
            assertFalse(chat.titleGenerated)
            assertEquals("tx1", db.messages().firstMessage("owner", "chat")!!.transactionId)
        } finally { db.close(); context.deleteDatabase(name) }
    }

    @Test fun conversationsDeleteOnlyTheirOwnMessagesAndPersist() = runBlocking {
        val context = ApplicationProvider.getApplicationContext<android.content.Context>()
        val name = "conversations-${java.util.UUID.randomUUID()}.db"
        fun open() = Room.databaseBuilder(context, ChatDatabase::class.java, name).build()
        var db = open()
        try {
            db.messages().addConversation(Conversation("a1", "a"))
            db.messages().addConversation(Conversation("a2", "a"))
            db.messages().addConversation(Conversation("b1", "b"))
            db.messages().put(ChatMessage("1", "a", text = "午饭", conversationId = "a1", state = "saved", transactionId = "tx1"))
            db.messages().put(ChatMessage("2", "a", text = "晚饭", conversationId = "a2"))
            db.messages().put(ChatMessage("3", "b", text = "工资", conversationId = "b1"))
            db.close(); db = open()
            assertEquals(2, db.messages().conversations("a").size)
            assertEquals("tx1", db.messages().observeConversation("a", "a1").first().single().transactionId)
            db.messages().deleteConversation("b", "a1")
            assertEquals(1, db.messages().observeConversation("a", "a1").first().size)
            db.messages().deleteConversation("a", "a1")
            assertTrue(db.messages().observeConversation("a", "a1").first().isEmpty())
            assertEquals("a2", db.messages().conversations("a").single().id)
            assertEquals("2", db.messages().observe("a").first().single().id)
            assertEquals("3", db.messages().observe("b").first().single().id)
        } finally { db.close(); context.deleteDatabase(name) }
    }

    @Test fun versionOneMigrationPreservesHistoryAndPendingStates() = runBlocking {
        val context = ApplicationProvider.getApplicationContext<android.content.Context>()
        val name = "migration-${java.util.UUID.randomUUID()}.db"
        context.openOrCreateDatabase(name, 0, null).use { old ->
            old.execSQL("CREATE TABLE messages (id TEXT NOT NULL PRIMARY KEY, owner TEXT NOT NULL, created INTEGER NOT NULL, text TEXT NOT NULL, state TEXT NOT NULL, reply TEXT NOT NULL, draft TEXT, transactionId TEXT)")
            old.execSQL("CREATE INDEX index_messages_owner ON messages(owner)")
            old.execSQL("INSERT INTO messages VALUES ('1','a',1,'午饭','saved','已记账',NULL,'tx1'), ('2','b',2,'晚饭','saving','正在保存','{}',NULL)")
            old.version = 1
        }
        val db = Room.databaseBuilder(context, ChatDatabase::class.java, name).addMigrations(ChatDatabase.MIGRATION_1_2, ChatDatabase.MIGRATION_2_3).build()
        try {
            val a = db.messages().conversations("a").single()
            assertEquals("历史对话", a.title)
            assertEquals("tx1", db.messages().observeConversation("a", a.id).first().single().transactionId)
            val b = db.messages().conversations("b").single()
            assertNotEquals(a.id, b.id)
            db.messages().recoverSaves()
            val pending = db.messages().observeConversation("b", b.id).first().single()
            assertEquals("unknown", pending.state)
            assertEquals("{}", pending.draft)
        } finally { db.close(); context.deleteDatabase(name) }
    }

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
