package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/log"
	"github.com/mayswind/financially_free/pkg/models"
)

type conversationAccount struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
	Hidden    bool   `json:"hidden"`
	Liability bool   `json:"liability"`
}
type conversationState struct {
	Accounts      []conversationAccount `json:"accounts,omitempty"`
	IncludeHidden bool                  `json:"includeHidden"`
	BlockedWrite  bool                  `json:"blockedWrite"`
}
type conversationToolResult struct {
	Data     any
	Reply    string
	Details  string
	Verified string
	Unknown  bool
}
type conversationToolRuntime struct {
	c              *core.WebContext
	state          *conversationState
	zone           *time.Location
	defaultAccount string
	messageID      string
	beforeWrite    func() error
	attempted      bool
}

// Invoke existing authenticated business handlers inside the service. No loopback HTTP,
// model-supplied URL, client-supplied UID, or duplicate copy of write validation.
func (t *conversationToolRuntime) invoke(handler func(*core.WebContext) (any, *errs.Error), query url.Values, body any) (any, error) {
	clone := core.WrapWebContext(t.c.Context.Copy())
	req := t.c.Request.Clone(t.c.Request.Context())
	u := *req.URL
	u.RawQuery = query.Encode()
	req.URL = &u
	req.Body = nil
	req.Method = http.MethodGet
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req.Method = http.MethodPost
		req.Body = io.NopCloser(bytes.NewReader(raw))
		req.ContentLength = int64(len(raw))
		req.Header = req.Header.Clone()
		req.Header.Set("Content-Type", "application/json")
	}
	clone.Request = req
	value, err := handler(clone)
	if err != nil {
		return nil, err
	}
	return value, nil
}
func strictToolArgs(raw json.RawMessage, v any) error {
	if len(raw) > 16384 || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return errors.New("工具参数必须是对象")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return errors.New("工具参数不完整或格式不正确，请重新确认")
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("工具参数格式不正确")
	}
	return nil
}
func conversationMoney(n int64) string {
	r := new(big.Rat).SetFrac(big.NewInt(n), big.NewInt(100))
	return r.FloatString(2)
}
func conversationAmount(value string) (int64, error) {
	if value == "" || strings.ContainsAny(value, "eE/ ") {
		return 0, errors.New("请提供十进制金额")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || (len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > 2)) {
		return 0, errors.New("金额最多两位小数")
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, errors.New("金额无效")
	}
	r.Mul(r, big.NewRat(100, 1))
	if !r.IsInt() || !r.Num().IsInt64() {
		return 0, errors.New("金额无效")
	}
	n := r.Num().Int64()
	if n < -99999999999 || n > 99999999999 {
		return 0, errors.New("金额超出范围")
	}
	return n, nil
}
func (t *conversationToolRuntime) allAccounts() ([]conversationAccount, error) {
	value, err := t.invoke(Accounts.AccountListHandler, url.Values{"visible_only": {"false"}}, nil)
	if err != nil {
		return nil, err
	}
	roots, ok := value.(models.AccountInfoResponseSlice)
	if !ok {
		return nil, errors.New("账户数据异常")
	}
	categoryOrder := map[models.AccountCategory]int{1: 0, 2: 1, 8: 2, 3: 3, 4: 4, 5: 5, 6: 6, 9: 7, 7: 8}
	sort.SliceStable(roots, func(i, j int) bool { return categoryOrder[roots[i].Category] < categoryOrder[roots[j].Category] })
	rows := []conversationAccount{}
	var visit func(*models.AccountInfoResponse, string, bool)
	visit = func(a *models.AccountInfoResponse, parent string, hidden bool) {
		name := a.Name
		if parent != "" {
			name = parent + " / " + name
		}
		hidden = hidden || a.Hidden
		if a.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			for _, child := range a.SubAccounts {
				visit(child, name, hidden)
			}
			return
		}
		rows = append(rows, conversationAccount{ID: strconv.FormatInt(a.Id, 10), Name: name, Balance: a.Balance, Currency: a.Currency, Hidden: hidden, Liability: a.IsLiability})
	}
	for _, a := range roots {
		visit(a, "", false)
	}
	return rows, nil
}
func (t *conversationToolRuntime) accountResult(rows []conversationAccount) conversationToolResult {
	type sum struct{ assets, debt int64 }
	totals := map[string]sum{}
	accounts := []any{}
	details := []string{}
	for i, a := range rows {
		v := totals[a.Currency]
		if a.Liability {
			v.debt -= a.Balance
		} else {
			v.assets += a.Balance
		}
		totals[a.Currency] = v
		accounts = append(accounts, map[string]any{"id": a.ID, "ordinal": i + 1, "name": a.Name, "balance": conversationMoney(a.Balance), "currency": a.Currency, "hidden": a.Hidden, "liability": a.Liability})
		hidden := ""
		if a.Hidden {
			hidden = "（隐藏）"
		}
		details = append(details, fmt.Sprintf("%d. %s%s\n%s %s", i+1, a.Name, hidden, conversationMoney(a.Balance), a.Currency))
	}
	scope := "不含隐藏账户"
	if t.state.IncludeHidden {
		scope = "含隐藏账户"
	}
	lines := []string{"当前资产 · " + scope}
	currencies := []string{}
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	for _, currency := range currencies {
		v := totals[currency]
		lines = append(lines, fmt.Sprintf("资产 %s %s\n负债 %s %s\n净资产 %s %s", conversationMoney(v.assets), currency, conversationMoney(v.debt), currency, conversationMoney(v.assets-v.debt), currency))
	}
	if len(rows) == 0 {
		lines = append(lines, "暂无资产账户")
	}
	summary := strings.Join(lines, "\n")
	return conversationToolResult{Data: map[string]any{"status": "success", "accounts": accounts, "include_hidden": t.state.IncludeHidden, "summary": summary}, Reply: summary, Details: strings.Join(details, "\n\n")}
}
func (t *conversationToolRuntime) execute(name string, args json.RawMessage) (conversationToolResult, error) {
	switch name {
	case "list_accounts":
		var a struct {
			Hidden *bool `json:"include_hidden"`
		}
		if err := strictToolArgs(args, &a); err != nil {
			return conversationToolResult{}, err
		}
		rows, err := t.allAccounts()
		if err != nil {
			return conversationToolResult{}, err
		}
		if a.Hidden != nil {
			t.state.IncludeHidden = *a.Hidden
		}
		t.state.Accounts = []conversationAccount{}
		for _, r := range rows {
			if t.state.IncludeHidden || !r.Hidden {
				t.state.Accounts = append(t.state.Accounts, r)
			}
		}
		t.state.BlockedWrite = false
		return t.accountResult(t.state.Accounts), nil
	case "query_transactions":
		return t.query(args)
	case "get_transaction_options":
		var a struct{}
		if err := strictToolArgs(args, &a); err != nil {
			return conversationToolResult{}, err
		}
		accounts, err := t.allAccounts()
		if err != nil {
			return conversationToolResult{}, err
		}
		cats, err := t.invoke(TransactionCategories.CategoryListHandler, nil, nil)
		if err != nil {
			return conversationToolResult{}, err
		}
		tags, err := t.invoke(TransactionTags.TagListHandler, nil, nil)
		if err != nil {
			return conversationToolResult{}, err
		}
		visible := []conversationAccount{}
		for _, a := range accounts {
			if !a.Hidden {
				visible = append(visible, a)
			}
		}
		flatCategories := []any{}
		grouped, ok := cats.(map[models.TransactionCategoryType]models.TransactionCategoryInfoResponseSlice)
		if !ok {
			return conversationToolResult{}, errors.New("分类数据异常")
		}
		for _, roots := range grouped {
			for _, parent := range roots {
				if parent.Hidden {
					continue
				}
				for _, child := range parent.SubCategories {
					if !child.Hidden && (child.Type == 1 || child.Type == 2) {
						flatCategories = append(flatCategories, map[string]any{"id": strconv.FormatInt(child.Id, 10), "name": parent.Name + " / " + child.Name, "type": int(child.Type) + 1})
					}
				}
			}
		}
		accountData := t.accountResult(visible).Data.(map[string]any)["accounts"]
		return conversationToolResult{Data: map[string]any{"status": "success", "default_account_id": t.defaultAccount, "accounts": accountData, "categories": flatCategories, "tags": tags}}, nil
	case "set_account_balance":
		return t.setBalance(args)
	case "create_transaction":
		return t.create(args)
	default:
		return conversationToolResult{}, errors.New("不支持该操作")
	}
}
func (t *conversationToolRuntime) startWrite() error {
	if t.attempted {
		return errors.New("本条消息已尝试过写入，请核实后另发消息")
	}
	t.attempted = true
	if err := t.beforeWrite(); err != nil {
		return err
	}
	return nil
}
func (t *conversationToolRuntime) setBalance(args json.RawMessage) (conversationToolResult, error) {
	var a struct {
		ID       string `json:"account_id"`
		Ordinal  *int   `json:"ordinal"`
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := strictToolArgs(args, &a); err != nil {
		return conversationToolResult{}, err
	}
	amount, err := conversationAmount(a.Amount)
	if err != nil {
		return conversationToolResult{}, err
	}
	var target *conversationAccount
	if a.Ordinal != nil {
		if *a.Ordinal < 1 || *a.Ordinal > len(t.state.Accounts) {
			return conversationToolResult{}, errors.New("请先查询资产列表并确认序号")
		}
		target = &t.state.Accounts[*a.Ordinal-1]
	}
	if a.ID != "" {
		var found *conversationAccount
		for i := range t.state.Accounts {
			if t.state.Accounts[i].ID == a.ID {
				found = &t.state.Accounts[i]
				break
			}
		}
		if found == nil || (target != nil && target.ID != found.ID) {
			return conversationToolResult{}, errors.New("账户与列表不匹配，请重新确认")
		}
		target = found
	}
	if target == nil || a.Currency != target.Currency {
		return conversationToolResult{}, errors.New("请明确账户及正确币种")
	}
	rows, err := t.allAccounts()
	if err != nil {
		return conversationToolResult{}, err
	}
	var current *conversationAccount
	for i := range rows {
		if rows[i].ID == target.ID {
			current = &rows[i]
			break
		}
	}
	if current == nil || *current != *target {
		return conversationToolResult{}, errors.New("账户信息或余额已变化，请重新查询并确认后操作")
	}
	if err = t.startWrite(); err != nil {
		return conversationToolResult{}, err
	}
	_, err = t.invoke(AccountBalances.ModifySingleAccountBalanceHandler, nil, map[string]any{"id": target.ID, "balance": amount, "expectedBalance": target.Balance})
	if err != nil {
		return t.writeFailure(err, "余额更新", false), nil
	}
	rows, err = t.allAccounts()
	if err != nil {
		return t.writeFailure(err, "余额更新", true), nil
	}
	verified := false
	for _, row := range rows {
		if row.ID == target.ID && row.Balance == amount && row.Currency == target.Currency {
			verified = true
		}
	}
	if !verified {
		return t.writeFailure(errors.New("readback mismatch"), "余额更新", true), nil
	}
	t.state.BlockedWrite = false
	receipt := fmt.Sprintf("已核实：%s 余额 %s → %s %s", target.Name, conversationMoney(target.Balance), conversationMoney(amount), target.Currency)
	t.state.Accounts = []conversationAccount{}
	for _, row := range rows {
		if t.state.IncludeHidden || !row.Hidden {
			t.state.Accounts = append(t.state.Accounts, row)
		}
	}
	output := t.accountResult(t.state.Accounts)
	output.Verified = receipt
	output.Reply = receipt
	output.Data = map[string]any{"status": "success", "verified": true, "summary": receipt, "accounts": output.Data}
	return output, nil
}
func (t *conversationToolRuntime) writeFailure(err error, label string, accepted bool) conversationToolResult {
	log.Errorf(t.c, "[conversation] message:%s write:%s accepted:%t error:%v", t.messageID, label, accepted, err)
	rejected := false
	var apiErr *errs.Error
	if errors.As(err, &apiErr) && apiErr.HttpStatusCode >= 400 && apiErr.HttpStatusCode < 500 && !accepted {
		rejected = true
	}
	t.state.Accounts = nil
	if rejected {
		t.state.BlockedWrite = false
		return conversationToolResult{Data: map[string]any{"status": "error", "message": "服务拒绝了操作，请核对账户与参数后重新确认", "reason": err.Error()}, Reply: label + "未完成，请核对后重新确认。"}
	}
	t.state.BlockedWrite = true
	message := label + "结果待核实，请先查询核对，系统不会重复提交。"
	return conversationToolResult{Data: map[string]any{"status": "unknown", "message": message}, Reply: message, Verified: message, Unknown: true}
}
func (t *conversationToolRuntime) create(args json.RawMessage) (conversationToolResult, error) {
	var a struct {
		Type     int      `json:"type"`
		Amount   string   `json:"amount"`
		ID       string   `json:"account_id"`
		Category string   `json:"category_id"`
		Currency string   `json:"currency"`
		Time     string   `json:"time"`
		Comment  string   `json:"comment"`
		Tags     []string `json:"tag_ids"`
	}
	if err := strictToolArgs(args, &a); err != nil {
		return conversationToolResult{}, err
	}
	amount, err := conversationAmount(a.Amount)
	if err != nil || amount == 0 {
		return conversationToolResult{}, errors.New("请提供有效金额")
	}
	if (a.Type != 2 && a.Type != 3) || a.Category == "" || len(a.Comment) > 255 {
		return conversationToolResult{}, errors.New("请明确收入或支出类型及分类")
	}
	accounts, err := t.allAccounts()
	if err != nil {
		return conversationToolResult{}, err
	}
	valid := false
	name := ""
	for _, row := range accounts {
		if row.ID == a.ID && !row.Hidden && row.Currency == a.Currency {
			valid = true
			name = row.Name
		}
	}
	if !valid {
		return conversationToolResult{}, errors.New("请获取可用账户并确认币种")
	}
	when := time.Now().In(t.zone)
	if a.Time != "" {
		when, err = time.ParseInLocation("2006-01-02T15:04:05", a.Time, t.zone)
		if err != nil {
			return conversationToolResult{}, errors.New("日期时间格式不正确")
		}
	}
	_, offset := when.Zone()
	body := map[string]any{"type": a.Type, "sourceAmount": amount, "sourceAccountId": a.ID, "categoryId": a.Category, "time": when.Unix(), "utcOffset": offset / 60, "comment": a.Comment, "tagIds": a.Tags, "clientSessionId": "conversation-" + t.messageID}
	if err = t.startWrite(); err != nil {
		return conversationToolResult{}, err
	}
	value, err := t.invoke(Transactions.TransactionCreateHandler, nil, body)
	if err != nil {
		return t.writeFailure(err, "记账", false), nil
	}
	raw, _ := json.Marshal(value)
	var response struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &response) != nil || response.ID == "" {
		return t.writeFailure(errors.New("missing transaction id"), "记账", true), nil
	}
	value, err = t.invoke(Transactions.TransactionGetHandler, url.Values{"id": {response.ID}}, nil)
	if err != nil {
		return t.writeFailure(err, "记账", true), nil
	}
	raw, _ = json.Marshal(value)
	var saved struct {
		ID      string `json:"id"`
		Amount  int64  `json:"sourceAmount"`
		Account string `json:"sourceAccountId"`
		Type    int    `json:"type"`
	}
	if json.Unmarshal(raw, &saved) != nil || saved.ID != response.ID || saved.Amount != amount || saved.Account != "0" || saved.Type != a.Type {
		return t.writeFailure(errors.New("readback mismatch"), "记账", true), nil
	}
	label := "支出"
	if a.Type == 2 {
		label = "收入"
	}
	receipt := fmt.Sprintf("已记账并核实：%s %s %s · %s", label, conversationMoney(amount), a.Currency, name)
	t.state.Accounts = nil
	t.state.BlockedWrite = false
	return conversationToolResult{Data: map[string]any{"status": "success", "verified": true, "transaction_id": response.ID, "summary": receipt}, Reply: receipt, Verified: receipt}, nil
}
func (t *conversationToolRuntime) query(args json.RawMessage) (conversationToolResult, error) {
	var a struct {
		Start   string `json:"start_date"`
		End     string `json:"end_date"`
		Type    int    `json:"type"`
		Account string `json:"account_id"`
		Keyword string `json:"keyword"`
		Offset  int    `json:"offset"`
	}
	if err := strictToolArgs(args, &a); err != nil {
		return conversationToolResult{}, err
	}
	start, e1 := time.ParseInLocation("2006-01-02", a.Start, t.zone)
	end, e2 := time.ParseInLocation("2006-01-02", a.End, t.zone)
	if e1 != nil || e2 != nil || end.Before(start) || end.Sub(start) > 367*24*time.Hour || a.Offset < 0 || (a.Type != 2 && a.Type != 3) {
		return conversationToolResult{}, errors.New("请指定有效的日期范围和收入或支出")
	}
	max := end.AddDate(0, 0, 1).Unix()*1000 - 1
	min := start.Unix() * 1000
	all := models.TransactionInfoResponseSlice{}
	for {
		value, err := t.invoke(Transactions.TransactionListHandler, url.Values{"type": {strconv.Itoa(a.Type)}, "min_time": {strconv.FormatInt(min, 10)}, "max_time": {strconv.FormatInt(max, 10)}, "count": {"50"}, "account_ids": {a.Account}, "keyword": {a.Keyword}}, nil)
		if err != nil {
			return conversationToolResult{}, err
		}
		page, ok := value.(*models.TransactionInfoPageWrapperResponse)
		if !ok {
			return conversationToolResult{}, errors.New("查询数据异常")
		}
		all = append(all, page.Items...)
		if len(all) > 5000 {
			return conversationToolResult{}, errors.New("记录较多，请缩小查询日期范围")
		}
		if page.NextTimeSequenceId == nil {
			break
		}
		if *page.NextTimeSequenceId >= max || *page.NextTimeSequenceId < min {
			return conversationToolResult{}, errors.New("查询分页异常")
		}
		max = *page.NextTimeSequenceId
	}
	totals := map[string]int64{}
	for _, r := range all {
		currency := r.Currency
		if currency == "" && r.SourceAccount != nil {
			currency = r.SourceAccount.Currency
		}
		if currency == "" {
			return conversationToolResult{}, errors.New("查询缺少币种")
		}
		totals[currency] += r.SourceAmount
	}
	values := map[string]string{}
	keys := []string{}
	for k, v := range totals {
		values[k] = conversationMoney(v)
		keys = append(keys, k)
	}
	sort.Strings(keys)
	label := "支出"
	if a.Type == 2 {
		label = "收入"
	}
	lines := []string{a.Start + " 至 " + a.End + " · " + label}
	if len(all) == 0 {
		lines = append(lines, "暂无"+label+"记录")
	}
	for _, k := range keys {
		lines = append(lines, "合计 "+values[k]+" "+k)
	}
	lines = append(lines, fmt.Sprintf("共 %d 笔", len(all)))
	transactions := []any{}
	details := []string{}
	limit := a.Offset + 50
	if limit > len(all) {
		limit = len(all)
	}
	for i := a.Offset; i < limit; i++ {
		r := all[i]
		currency, accountName := r.Currency, ""
		if r.SourceAccount != nil {
			accountName = r.SourceAccount.Name
			if currency == "" {
				currency = r.SourceAccount.Currency
			}
		}
		category := ""
		if r.Category != nil {
			category = r.Category.Name
		}
		at := time.Unix(r.Time, 0).In(t.zone).Format("2006-01-02 15:04")
		transactions = append(transactions, map[string]any{"id": strconv.FormatInt(r.Id, 10), "time": at, "amount": conversationMoney(r.SourceAmount), "currency": currency, "account": accountName, "category": category, "comment": r.Comment})
		details = append(details, fmt.Sprintf("%s · %s %s\n%s · %s\n%s", at, conversationMoney(r.SourceAmount), currency, category, accountName, r.Comment))
	}
	var next any
	if limit < len(all) {
		next = limit
	}
	t.state.BlockedWrite = false
	return conversationToolResult{Data: map[string]any{"status": "success", "start_date": a.Start, "end_date": a.End, "timezone": t.zone.String(), "account_id": a.Account, "total_count": len(all), "totals": values, "transactions": transactions, "next_offset": next}, Reply: strings.Join(lines, "\n"), Details: strings.Join(details, "\n\n")}, nil
}
