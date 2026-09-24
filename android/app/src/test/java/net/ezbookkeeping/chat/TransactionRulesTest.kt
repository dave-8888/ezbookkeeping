package net.ezbookkeeping.chat

import org.junit.Assert.*
import org.junit.Test
import org.json.JSONObject

class TransactionRulesTest {
    private val catalog = Catalog(
        listOf(Choice("a", "现金", currency = "CNY"), Choice("b", "银行卡", currency = "CNY")),
        listOf(Choice("food", "餐饮", 2, "expense"), Choice("salary", "工资", 1, "income")),
        listOf(Choice("tag", "日常")), "b")
    @Test fun expenseUsesDefaultAndMinorUnits() {
        val (d, missing) = TransactionRules.recognize(JSONObject("""{"type":3,"sourceAmount":3050}"""), "午饭30.5", catalog)
        assertTrue(missing.isEmpty()); assertEquals("b", d.account); assertEquals("food", d.category)
        assertEquals(3050L, d.amount); assertEquals("30.50", TransactionRules.amount(d.amount))
        assertEquals("午饭30.5", d.comment)
    }
    @Test fun incomePreservesExplicitFields() {
        val (d, missing) = TransactionRules.recognize(JSONObject("""{"type":2,"sourceAmount":800000,"sourceAccountId":"a","categoryId":"salary","time":1700000000,"tagIds":["tag","missing"]}"""), "工资", catalog)
        assertTrue(missing.isEmpty()); assertEquals(2, d.type); assertEquals("a", d.account)
        assertEquals(listOf("tag"), d.tags); assertEquals(1700000000L, d.time)
    }
    @Test fun missingTypeRequiresForm() {
        val (_, missing) = TransactionRules.recognize(JSONObject("""{"sourceAmount":3000}"""), "30", catalog)
        assertTrue(missing.contains("交易类型"))
    }
    @Test fun categoryCanInferType() {
        val (d, missing) = TransactionRules.recognize(JSONObject("""{"sourceAmount":100,"categoryId":"salary"}"""), "收入", catalog)
        assertEquals(2, d.type); assertTrue(missing.isEmpty())
    }
    @Test fun parentCategoryResolvesToChild() {
        val (d, missing) = TransactionRules.recognize(JSONObject("""{"type":3,"sourceAmount":100,"categoryId":"expense"}"""), "支出", catalog)
        assertEquals("food", d.category); assertTrue(missing.isEmpty())
    }
    @Test fun mismatchRequiresCorrection() {
        val (_, missing) = TransactionRules.recognize(JSONObject("""{"type":3,"sourceAmount":100,"categoryId":"salary"}"""), "支出", catalog)
        assertTrue(missing.contains("分类"))
    }
    @Test fun missingAccountsAndAmountBlockSave() {
        val (_, missing) = TransactionRules.recognize(JSONObject("""{"type":3}"""), "午饭", Catalog())
        assertTrue(missing.containsAll(listOf("账户", "金额", "分类")))
    }
    @Test fun zeroAmountBlocksSave() { assertTrue(TransactionRules.validate(Draft(amount = 0), catalog).contains("金额")) }
    @Test(expected = IllegalArgumentException::class) fun transferIsNeverExpense() {
        TransactionRules.recognize(JSONObject("""{"type":4,"sourceAmount":100}"""), "转账", catalog)
    }
    @Test fun payloadAndRestorationKeepSubmissionIdentity() {
        val draft = Draft(amount = 100, account = "a", category = "food")
        val json = draft.json()
        assertEquals(draft, Draft.from(json))
        assertEquals("0", json.getString("destinationAccountId")); assertEquals(0L, json.getLong("destinationAmount"))
        assertFalse(json.getBoolean("hideAmount")); assertEquals(0, json.getJSONArray("pictureIds").length())
        assertEquals(draft.session, Draft.from(JSONObject(json.toString())).session)
    }
    @Test fun decimalInputIsExact() { assertEquals(1001L, TransactionRules.parseAmount("10.01")); assertEquals(-1001L, TransactionRules.parseAmount("-10.01")) }
    @Test(expected = ArithmeticException::class) fun fractionsOfCentRejected() { TransactionRules.parseAmount("0.001") }
}
