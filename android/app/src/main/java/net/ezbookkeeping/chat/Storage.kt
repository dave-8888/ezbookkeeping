package net.ezbookkeeping.chat

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import androidx.room.*
import androidx.room.migration.Migration
import androidx.sqlite.db.SupportSQLiteDatabase
import kotlinx.coroutines.flow.Flow
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

@Entity(tableName = "messages", indices = [Index("owner"), Index(value = ["owner", "conversationId"])])
data class ChatMessage(@PrimaryKey val id: String, val owner: String, val created: Long = System.currentTimeMillis(), val text: String,
    val state: String = "recognizing", val reply: String = "正在识别…", val draft: String? = null, val transactionId: String? = null,
    @ColumnInfo(defaultValue = "''") val conversationId: String = "")
@Entity(tableName = "conversations", indices = [Index("owner")])
data class Conversation(@PrimaryKey val id: String, val owner: String,
    val title: String = "新对话", val created: Long = System.currentTimeMillis(),
    @ColumnInfo(defaultValue = "0") val titleGenerated: Boolean = false)

@Dao interface MessageDao {
    @Query("SELECT * FROM conversations WHERE owner = :owner ORDER BY created DESC, id DESC")
    fun observeConversations(owner: String): Flow<List<Conversation>>
    @Query("SELECT * FROM conversations WHERE owner = :owner ORDER BY created DESC, id DESC")
    suspend fun conversations(owner: String): List<Conversation>
    @Insert suspend fun addConversation(conversation: Conversation)
    @Query("SELECT * FROM conversations WHERE owner = :owner AND id = :id")
    suspend fun conversation(owner: String, id: String): Conversation?
    @Query("SELECT * FROM messages WHERE owner = :owner AND conversationId = :id ORDER BY created, id LIMIT 1")
    suspend fun firstMessage(owner: String, id: String): ChatMessage?
    @Query("UPDATE conversations SET title = :title WHERE owner = :owner AND id = :id AND titleGenerated = 0")
    suspend fun setFallbackTitle(owner: String, id: String, title: String)
    @Query("UPDATE conversations SET title = :title, titleGenerated = 1 WHERE owner = :owner AND id = :id AND titleGenerated = 0 AND :messageId = (SELECT id FROM messages WHERE owner = :owner AND conversationId = :id ORDER BY created, id LIMIT 1)")
    suspend fun saveGeneratedTitle(owner: String, id: String, messageId: String, title: String)
    @Query("SELECT * FROM messages WHERE owner = :owner AND conversationId = :conversationId ORDER BY created, id")
    fun observeConversation(owner: String, conversationId: String): Flow<List<ChatMessage>>
    @Query("SELECT * FROM messages WHERE owner = :owner AND conversationId = :id ORDER BY created DESC, id DESC")
    suspend fun conversationHistory(owner: String, id: String): List<ChatMessage>
    @Query("UPDATE messages SET state = 'update_unknown', reply = '资产更新结果待核实，请重新查询资产余额。不会自动重发更新。' WHERE state = 'updating'")
    suspend fun recoverAssetUpdates()
    @Query("UPDATE messages SET state = CASE WHEN state = 'agent_writing' THEN 'agent_unknown' ELSE 'agent_interrupted' END, reply = CASE WHEN state = 'agent_writing' THEN '操作结果待核实，请先查询账目或余额。不会自动重发。' ELSE '对话处理中断，请查看已有结果后继续。' END WHERE state IN ('agent_running', 'agent_writing')")
    suspend fun recoverAgentTurns()
    @Query("DELETE FROM messages WHERE owner = :owner AND conversationId = :id")
    suspend fun deleteConversationMessages(owner: String, id: String)
    @Query("DELETE FROM conversations WHERE owner = :owner AND id = :id")
    suspend fun deleteConversationRow(owner: String, id: String)
    @Transaction suspend fun deleteConversation(owner: String, id: String) {
        deleteConversationMessages(owner, id)
        deleteConversationRow(owner, id)
    }
    @Query("SELECT * FROM messages WHERE owner = :owner ORDER BY created, id") fun observe(owner: String): Flow<List<ChatMessage>>
    @Insert(onConflict = OnConflictStrategy.REPLACE) suspend fun put(message: ChatMessage)
    @Query("DELETE FROM messages WHERE owner = :owner") suspend fun clear(owner: String)
    @Query("UPDATE messages SET state = 'unknown', reply = '保存结果待核实，请到网页账单确认。为避免重复记账，不会自动重发。' WHERE state = 'saving'") suspend fun recoverSaves()
    @Query("UPDATE messages SET state = 'failed', reply = '识别已中断，请重新输入。' WHERE state = 'recognizing'") suspend fun recoverRecognition()
}
@Database(entities = [ChatMessage::class, Conversation::class], version = 3, exportSchema = false)
abstract class ChatDatabase: RoomDatabase() {
    abstract fun messages(): MessageDao
    companion object {
        val MIGRATION_2_3 = object: Migration(2, 3) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE conversations ADD COLUMN titleGenerated INTEGER NOT NULL DEFAULT 0")
                db.execSQL("UPDATE conversations SET title = substr(title, 1, 15)")
            }
        }
        val MIGRATION_1_2 = object: Migration(1, 2) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE messages ADD COLUMN conversationId TEXT NOT NULL DEFAULT ''")
                db.execSQL("CREATE INDEX index_messages_owner_conversationId ON messages(owner, conversationId)")
                db.execSQL("CREATE TABLE conversations (id TEXT NOT NULL PRIMARY KEY, owner TEXT NOT NULL, title TEXT NOT NULL, created INTEGER NOT NULL)")
                db.execSQL("CREATE INDEX index_conversations_owner ON conversations(owner)")
                db.execSQL("INSERT INTO conversations SELECT 'legacy:' || owner, owner, '历史对话', MIN(created) FROM messages GROUP BY owner")
                db.execSQL("UPDATE messages SET conversationId = 'legacy:' || owner")
            }
        }
    }
}

class SessionStore(context: Context) {
    private val prefs = context.getSharedPreferences("connection", Context.MODE_PRIVATE)
    fun selectedConversation(owner: String): String = prefs.getString("conversation:$owner", "").orEmpty()
    fun selectConversation(owner: String, id: String) { prefs.edit().putString("conversation:$owner", id).commit() }
    var base: String get() = prefs.getString("base", "").orEmpty(); set(v) { prefs.edit().putString("base", v).commit() }
    var user: String get() = prefs.getString("user", "").orEmpty(); set(v) { prefs.edit().putString("user", v).commit() }
    private fun key(): SecretKey {
        val ks = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        (ks.getKey("chat_session", null) as? SecretKey)?.let { return it }
        return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore").apply {
            init(KeyGenParameterSpec.Builder("chat_session", KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM).setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE).build())
        }.generateKey()
    }
    var token: String
        get() = try {
            val encoded = prefs.getString("token", null)
            if(encoded == null) "" else {
                val parts = encoded.split(":")
                val cipher = Cipher.getInstance("AES/GCM/NoPadding")
                cipher.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, Base64.decode(parts[0], Base64.NO_WRAP)))
                String(cipher.doFinal(Base64.decode(parts[1], Base64.NO_WRAP)), Charsets.UTF_8)
            }
        } catch(_: Exception) { prefs.edit().remove("token").commit(); "" }
        set(value) {
            if(value.isEmpty()) { prefs.edit().remove("token").commit(); return }
            val cipher = Cipher.getInstance("AES/GCM/NoPadding").apply { init(Cipher.ENCRYPT_MODE, key()) }
            val encrypted = cipher.doFinal(value.toByteArray(Charsets.UTF_8))
            prefs.edit().putString("token", Base64.encodeToString(cipher.iv, Base64.NO_WRAP) + ":" + Base64.encodeToString(encrypted, Base64.NO_WRAP)).commit()
        }
}
