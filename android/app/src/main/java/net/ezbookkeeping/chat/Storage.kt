package net.ezbookkeeping.chat

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import androidx.room.*
import kotlinx.coroutines.flow.Flow
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

@Entity(tableName = "messages", indices = [Index("owner")])
data class ChatMessage(@PrimaryKey val id: String, val owner: String, val created: Long = System.currentTimeMillis(), val text: String,
    val state: String = "recognizing", val reply: String = "正在识别…", val draft: String? = null, val transactionId: String? = null)
@Dao interface MessageDao {
    @Query("SELECT * FROM messages WHERE owner = :owner ORDER BY created, id") fun observe(owner: String): Flow<List<ChatMessage>>
    @Insert(onConflict = OnConflictStrategy.REPLACE) suspend fun put(message: ChatMessage)
    @Query("DELETE FROM messages WHERE owner = :owner") suspend fun clear(owner: String)
    @Query("UPDATE messages SET state = 'unknown', reply = '保存结果待核实，请到网页账单确认。为避免重复记账，不会自动重发。' WHERE state = 'saving'") suspend fun recoverSaves()
    @Query("UPDATE messages SET state = 'failed', reply = '识别已中断，请重新输入。' WHERE state = 'recognizing'") suspend fun recoverRecognition()
}
@Database(entities = [ChatMessage::class], version = 1, exportSchema = false)
abstract class ChatDatabase: RoomDatabase() { abstract fun messages(): MessageDao }

class SessionStore(context: Context) {
    private val prefs = context.getSharedPreferences("connection", Context.MODE_PRIVATE)
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
