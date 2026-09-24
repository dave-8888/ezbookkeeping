package net.ezbookkeeping.chat

import android.app.DatePickerDialog
import android.app.TimePickerDialog
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable private fun ChoiceField(label: String, selected: String, choices: List<Choice>, enabled: Boolean, onChange: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    Column {
        Text(label, style = MaterialTheme.typography.labelMedium)
        Box {
            OutlinedButton(onClick = { expanded = true }, enabled = enabled, modifier = Modifier.fillMaxWidth()) {
                Text(choices.find { it.id == selected }?.name ?: "请选择")
            }
            DropdownMenu(expanded, { expanded = false }, modifier = Modifier.heightIn(max = 320.dp)) {
                choices.forEach { choice -> DropdownMenuItem(text = { Text(choice.name) }, onClick = { onChange(choice.id); expanded = false }) }
            }
        }
        if(choices.isEmpty()) Text("暂无可选项，请先在网页版添加", style = MaterialTheme.typography.bodySmall)
    }
}

@Composable fun TransactionEditor(initial: Draft, catalog: Catalog, busy: Boolean, cancel: () -> Unit, save: (Draft) -> Unit, required: List<String> = emptyList()) {
    var type by rememberSaveable { mutableIntStateOf(initial.type) }
    var amount by rememberSaveable { mutableStateOf(TransactionRules.amount(initial.amount)) }
    var account by rememberSaveable { mutableStateOf(initial.account) }
    var category by rememberSaveable { mutableStateOf(initial.category) }
    var epoch by rememberSaveable { mutableLongStateOf(initial.time) }
    var comment by rememberSaveable { mutableStateOf(initial.comment) }
    var tags by rememberSaveable { mutableStateOf(initial.tags.joinToString(",")) }
    var problem by rememberSaveable { mutableStateOf("") }
    var editDetails by rememberSaveable { mutableStateOf(false) }
    var more by rememberSaveable { mutableStateOf(false) }
    val missing = remember(initial, catalog, required) { (TransactionRules.validate(initial, catalog) + required).distinct() }
    val context = LocalContext.current
    val localTime = LocalDateTime.ofInstant(Instant.ofEpochSecond(epoch), ZoneId.systemDefault())
    BackHandler(enabled = !busy, onBack = cancel)
    Column(Modifier.fillMaxSize().padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("补全交易", style = MaterialTheme.typography.headlineSmall)
            if(missing.isNotEmpty()) Text("请补全${missing.joinToString("、")}", color = MaterialTheme.colorScheme.onSurfaceVariant)
            if(editDetails || "交易类型" in missing) ChoiceField("类型", type.toString(), listOf(Choice("3", "支出"), Choice("2", "收入")), !busy) {
                type = it.toInt(); category = ""
            }
            if(editDetails || "金额" in missing) AppField(amount, { amount = it }, label = { Text("金额 · ${catalog.accounts.find { it.id == account }?.currency.orEmpty()}") },
                modifier = Modifier.fillMaxWidth(), enabled = !busy, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true)
            if(editDetails || "账户" in missing) ChoiceField("账户", account, catalog.accounts, !busy) { account = it }
            if(editDetails || "分类" in missing || catalog.categories.none { it.id == category && it.type == type - 1 }) {
                ChoiceField("分类", category, catalog.categories.filter { it.type == type - 1 }, !busy) { category = it }
            }
            Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surface) {
                Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("交易信息", style = MaterialTheme.typography.titleMedium)
                    Text(listOfNotNull(if(type == 2) "收入" else if(type == 3) "支出" else null,
                        amount.takeIf { it.isNotBlank() }?.let { "$it ${catalog.accounts.find { a -> a.id == account }?.currency.orEmpty()}" },
                        catalog.categories.find { it.id == category }?.name,
                        catalog.accounts.find { it.id == account }?.name).joinToString(" · "))
                    TextButton(onClick = { editDetails = !editDetails }, enabled = !busy) { Text(if(editDetails) "收起修改" else "修改已识别信息") }
                    TextButton(onClick = {
                        DatePickerDialog(context, { _, year, month, day ->
                            epoch = localTime.withYear(year).withMonth(1).withDayOfMonth(1).withMonth(month + 1).withDayOfMonth(day).atZone(ZoneId.systemDefault()).toEpochSecond()
                        }, localTime.year, localTime.monthValue - 1, localTime.dayOfMonth).show()
                    }, enabled = !busy) { Text("日期 · ${localTime.format(DateTimeFormatter.ofPattern("yyyy-MM-dd"))}") }
                    TextButton(onClick = {
                        TimePickerDialog(context, { _, hour, minute ->
                            epoch = localTime.withHour(hour).withMinute(minute).atZone(ZoneId.systemDefault()).toEpochSecond()
                        }, localTime.hour, localTime.minute, true).show()
                    }, enabled = !busy) { Text("时间 · ${localTime.format(DateTimeFormatter.ofPattern("HH:mm"))}") }
                }
            }
            TextButton(onClick = { more = !more }, enabled = !busy) { Text(if(more) "收起更多信息" else "更多信息") }
            if(more) {
                AppField(comment, { comment = it }, label = { Text("备注") }, modifier = Modifier.fillMaxWidth(), enabled = !busy)
                if(catalog.tags.isNotEmpty()) Text("标签", style = MaterialTheme.typography.labelMedium)
                catalog.tags.forEach { tag ->
                    val selected = tag.id in tags.split(',')
                    Row(Modifier.fillMaxWidth().heightIn(min = 48.dp).toggleable(selected, enabled = !busy, role = Role.Checkbox) { checked ->
                        tags = tags.split(',').filter { it.isNotEmpty() && it != tag.id }.let { if(checked) it + tag.id else it }.joinToString(",")
                    }, verticalAlignment = Alignment.CenterVertically) {
                        Checkbox(selected, onCheckedChange = null, enabled = !busy)
                        Spacer(Modifier.width(12.dp)); Text(tag.name)
                    }
                }
            }
        }
        if(problem.isNotEmpty()) Text(problem, color = MaterialTheme.colorScheme.error)
        Button(onClick = {
            try {
                val draft = initial.copy(type = type, amount = TransactionRules.parseAmount(amount), account = account, category = category,
                    time = epoch, comment = comment, tags = tags.split(',').filter { it.isNotEmpty() })
                val invalid = TransactionRules.validate(draft, catalog)
                if(invalid.isNotEmpty()) { problem = "请检查：${invalid.joinToString("、")}"; editDetails = true }
                else { problem = ""; save(draft) }
            } catch(_: ArithmeticException) { problem = "金额最多保留两位小数"; editDetails = true }
            catch(_: NumberFormatException) { problem = "请填写有效金额"; editDetails = true }
        }, enabled = !busy, modifier = Modifier.fillMaxWidth()) { Text(if(busy) "正在保存…" else "保存交易") }
        TextButton(onClick = cancel, enabled = !busy, modifier = Modifier.fillMaxWidth()) { Text("返回聊天") }
    }
}
