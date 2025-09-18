package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url" // 替换: 原来误用 "url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gogogo1024/assist-fusion/internal/common"
)

const (
	contentTypeJSON      = "application/json"
	headerContentType    = "Content-Type"
	errServerNotReadyFmt = "server not ready at %s"
	docsPath             = "/v1/docs"
	ticketPrefix         = "/v1/tickets/"
)

// test helpers
func waitReady(t *testing.T, baseURL string) {
	t.Helper()
	ready := false
	for i := 0; i < 100; i++ {
		resp, err := http.Get(baseURL + pathTickets)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatalf(errServerNotReadyFmt, baseURL)
	}
}

func putAndDecode[T any](t *testing.T, url string, out *T) {
	t.Helper()
	req, _ := http.NewRequest("PUT", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s err: %v", url, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s resp: %v", url, err)
	}
}

type ticketResp struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	AssignedAt  int64  `json:"assigned_at"`
	ResolvedAt  int64  `json:"resolved_at"`
	EscalatedAt int64  `json:"escalated_at"`
	ReopenedAt  int64  `json:"reopened_at"`
}

func TestTicketAndAIFlow(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18081"}
	h := BuildServer(cfg)
	// start server in background
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18081"

	waitReady(t, baseURL)

	// create ticket
	body := map[string]string{"title": "test", "desc": "hello"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post ticket: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// embeddings
	body2 := map[string]any{"texts": []string{"hello"}, "dim": 4}
	b2, _ := json.Marshal(body2)
	resp2, err := http.Post(baseURL+"/v1/embeddings", contentTypeJSON, bytes.NewReader(b2))
	if err != nil {
		t.Fatalf("post embeddings: %v", err)
	}
	var out struct {
		Vectors [][]float64 `json:"vectors"`
		Dim     int         `json:"dim"`
	}
	dec := json.NewDecoder(resp2.Body)
	_ = dec.Decode(&out)
	resp2.Body.Close()
	if len(out.Vectors) != 1 || out.Dim != 4 {
		t.Fatalf("unexpected embeddings resp: %#v", out)
	}
}

func TestKBFlow(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18082"}
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18082"

	waitReady(t, baseURL)

	// add a doc
	doc := map[string]string{"title": "FAQ 客服", "content": "客服如何升级？请参考SLA"}
	db, _ := json.Marshal(doc)
	resp, err := http.Post(baseURL+"/v1/docs", contentTypeJSON, bytes.NewReader(db))
	if err != nil {
		t.Fatalf("post doc: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 for docs, got %d", resp.StatusCode)
	}

	// search
	resp2, err := http.Get(baseURL + "/v1/search?q=客服")
	if err != nil {
		t.Fatalf("get search: %v", err)
	}
	var out struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	dec := json.NewDecoder(resp2.Body)
	_ = dec.Decode(&out)
	resp2.Body.Close()
	if out.Total < 1 || len(out.Items) < 1 {
		t.Fatalf("expected at least one search result, got: %#v", out)
	}
}

func TestTicketLifecycle(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18083"}
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18083"

	waitReady(t, baseURL)

	// create
	create := map[string]string{"title": "lifecycle", "desc": "demo"}
	b, _ := json.Marshal(create)
	resp, err := http.Post(baseURL+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil || resp.StatusCode != 201 {
		t.Fatalf("create ticket err=%v code=%d", err, resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// list
	resp, err = http.Get(baseURL + pathTickets)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("list err=%v code=%d", err, resp.StatusCode)
	}
	resp.Body.Close()

	// assign/resolve/escalate use a fake id path (we don't parse id from create resp for brevity):
	// This is a minimal smoke; in full test we'd parse the returned ticket ID.
	// Here ensure 404 for non-existent id endpoints are well-behaved.
	for _, p := range []string{"assign", "resolve", "escalate"} {
		req, _ := http.NewRequest("PUT", baseURL+ticketPrefix+"does-not-exist/"+p, nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 404 {
			t.Fatalf("%s expected 404, got err=%v code=%d", p, err, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// helpers for positive lifecycle
func createTicket(t *testing.T, baseURL, title, desc string) ticketResp {
	t.Helper()
	body := map[string]string{"title": title, "desc": desc}
	b, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create ticket err=%v", err)
	}
	defer resp.Body.Close()
	var tk ticketResp
	if err := json.NewDecoder(resp.Body).Decode(&tk); err != nil {
		t.Fatalf("decode create resp: %v", err)
	}
	if tk.ID == "" || tk.Status != "created" || tk.CreatedAt == 0 {
		t.Fatalf("unexpected create ticket: %#v", tk)
	}
	return tk
}

func doAction(t *testing.T, baseURL, id, action string) ticketResp {
	t.Helper()
	var tk ticketResp
	putAndDecode(t, baseURL+ticketPrefix+id+"/"+action, &tk)
	return tk
}

func assertAssigned(t *testing.T, tk ticketResp) {
	t.Helper()
	if tk.Status != "assigned" || tk.AssignedAt == 0 {
		t.Fatalf("unexpected assign ticket: %#v", tk)
	}
}

func assertResolved(t *testing.T, tk ticketResp) {
	t.Helper()
	if tk.Status != "resolved" || tk.ResolvedAt == 0 {
		t.Fatalf("unexpected resolve ticket: %#v", tk)
	}
}

func assertEscalated(t *testing.T, tk ticketResp) {
	t.Helper()
	if tk.Status != "escalated" || tk.EscalatedAt == 0 {
		t.Fatalf("unexpected escalate ticket: %#v", tk)
	}
}

func TestTicketLifecyclePositive(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18084"}
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18084"
	waitReady(t, baseURL)

	// 新的业务：不允许 resolved 后 escalate。
	// 正向链路：created -> assign -> escalate -> resolve。
	tk := createTicket(t, baseURL, "positive", "lifecycle")
	tk = doAction(t, baseURL, tk.ID, "assign")
	assertAssigned(t, tk)
	tk = doAction(t, baseURL, tk.ID, "escalate")
	assertEscalated(t, tk)
	tk = doAction(t, baseURL, tk.ID, "resolve")
	assertResolved(t, tk)

	// 已 resolved 后再次尝试 escalate，期望 409 且状态不变
	req, _ := http.NewRequest("PUT", baseURL+ticketPrefix+tk.ID+"/escalate", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("escalate after resolved err: %v", err)
	}
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 after resolved, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// reopen：将已解决的工单重开
	tk = doAction(t, baseURL, tk.ID, "reopen")
	if tk.Status != "created" || tk.ReopenedAt == 0 || tk.AssignedAt != 0 || tk.ResolvedAt != 0 || tk.EscalatedAt != 0 {
		t.Fatalf("unexpected reopen ticket: %#v", tk)
	}
}

func TestTicketCyclesAPI(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18085"}
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18085"
	waitReady(t, baseURL)

	// create and run through a cycle
	tk := createTicket(t, baseURL, "cycles", "inspect")
	tk = doAction(t, baseURL, tk.ID, "assign")
	tk = doAction(t, baseURL, tk.ID, "escalate")
	tk = doAction(t, baseURL, tk.ID, "resolve")

	// reopen -> new cycle
	tk = doAction(t, baseURL, tk.ID, "reopen")

	// fetch cycles
	resp, err := http.Get(baseURL + ticketPrefix + tk.ID + "/cycles")
	if err != nil {
		t.Fatalf("get cycles err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for cycles, got %d", resp.StatusCode)
	}
	var cyclesOut struct {
		Current int `json:"current"`
		Cycles  []struct {
			CreatedAt   int64  `json:"created_at"`
			AssignedAt  int64  `json:"assigned_at"`
			ResolvedAt  int64  `json:"resolved_at"`
			EscalatedAt int64  `json:"escalated_at"`
			Status      string `json:"status"`
		} `json:"cycles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cyclesOut); err != nil {
		t.Fatalf("decode cycles: %v", err)
	}
	if len(cyclesOut.Cycles) != 2 || cyclesOut.Current != 1 {
		t.Fatalf("unexpected cycles meta: %#v", cyclesOut)
	}
	// first cycle should be resolved
	if cyclesOut.Cycles[0].Status != "resolved" || cyclesOut.Cycles[0].ResolvedAt == 0 {
		t.Fatalf("first cycle not resolved: %#v", cyclesOut.Cycles[0])
	}
	// second cycle just created
	if cyclesOut.Cycles[1].Status != "created" || cyclesOut.Cycles[1].CreatedAt == 0 {
		t.Fatalf("second cycle not created: %#v", cyclesOut.Cycles[1])
	}
}

func TestTicketGetIncludesCycles(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18086"}
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18086"
	waitReady(t, baseURL)

	tk := createTicket(t, baseURL, "withCycles", "detail")
	tk = doAction(t, baseURL, tk.ID, "assign")
	tk = doAction(t, baseURL, tk.ID, "resolve")
	tk = doAction(t, baseURL, tk.ID, "reopen")

	// GET single ticket, should include cycles and currentCycle fields
	resp, err := http.Get(baseURL + ticketPrefix + tk.ID)
	if err != nil {
		t.Fatalf("get ticket err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var detail struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		CurrentCycle int    `json:"current_cycle"`
		Cycles       []struct {
			CreatedAt   int64  `json:"created_at"`
			AssignedAt  int64  `json:"assigned_at"`
			ResolvedAt  int64  `json:"resolved_at"`
			EscalatedAt int64  `json:"escalated_at"`
			Status      string `json:"status"`
		} `json:"cycles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatalf("decode ticket detail: %v", err)
	}
	if len(detail.Cycles) != 2 || detail.CurrentCycle != 1 || detail.Status != "created" {
		t.Fatalf("unexpected ticket detail meta: %#v", detail)
	}
	if detail.Cycles[0].Status != "resolved" || detail.Cycles[0].ResolvedAt == 0 {
		t.Fatalf("first cycle should be resolved: %#v", detail.Cycles[0])
	}
	if detail.Cycles[1].Status != "created" || detail.Cycles[1].CreatedAt == 0 {
		t.Fatalf("second cycle should be created: %#v", detail.Cycles[1])
	}
}

func TestTicketEventsAPI(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18087"}
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})

	baseURL := "http://127.0.0.1:18087"
	waitReady(t, baseURL)

	// create with note
	body := map[string]string{"title": "events", "desc": "audit", "note": "new ticket"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create with note: %v", err)
	}
	var tk ticketResp
	_ = json.NewDecoder(resp.Body).Decode(&tk)
	resp.Body.Close()

	// assign with note
	req, _ := http.NewRequest("PUT", baseURL+ticketPrefix+tk.ID+"/assign", bytes.NewReader([]byte(`{"note":"assigning"}`)))
	req.Header.Set(headerContentType, contentTypeJSON)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("assign with note: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// escalate with note
	req, _ = http.NewRequest("PUT", baseURL+ticketPrefix+tk.ID+"/escalate", bytes.NewReader([]byte(`{"note":"urgent"}`)))
	req.Header.Set(headerContentType, contentTypeJSON)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("escalate with note: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// resolve with note
	req, _ = http.NewRequest("PUT", baseURL+ticketPrefix+tk.ID+"/resolve", bytes.NewReader([]byte(`{"note":"fixed"}`)))
	req.Header.Set(headerContentType, contentTypeJSON)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("resolve with note: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// reopen with note
	req, _ = http.NewRequest("PUT", baseURL+ticketPrefix+tk.ID+"/reopen", bytes.NewReader([]byte(`{"note":"wrong fix"}`)))
	req.Header.Set(headerContentType, contentTypeJSON)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reopen with note: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	resp, err = http.Get(baseURL + ticketPrefix + tk.ID + "/events")
	if err != nil {
		t.Fatalf("get events err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var out struct {
		Events []struct {
			Type string `json:"type"`
			At   int64  `json:"at"`
			Note string `json:"note"`
		} `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(out.Events) != 5 {
		t.Fatalf("expected 5 events (created,assigned,escalated,resolved,reopened), got %d", len(out.Events))
	}
	want := []string{"created", "assigned", "escalated", "resolved", "reopened"}
	for i, w := range want {
		if out.Events[i].Type != w || out.Events[i].At == 0 {
			t.Fatalf("event %d expected %s, got %#v", i, w, out.Events[i])
		}
	}
	// notes
	notes := []string{"new ticket", "assigning", "urgent", "fixed", "wrong fix"}
	for i, n := range notes {
		if out.Events[i].Note != n {
			t.Fatalf("event %d note expected %q, got %q", i, n, out.Events[i].Note)
		}
	}
}

func TestSearchShortQueryAutocomplete(t *testing.T) {
	// 仅在 ES 后端下测试；否则跳过
	if os.Getenv("KB_BACKEND") != "es" {
		t.Skip("skip: KB_BACKEND!=es")
	}
	// 允许本测试在未装 IK(回退 ngram) 或已装 IK 下都工作；只要求能命中
	s, addr := startTestServer(t) // 复用你已有的启动辅助函数（若名称不同请替换）
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_ = s.Shutdown(ctx)
	}()

	base := fmt.Sprintf("http://%s", addr)

	// 准备三条文档，其中 title 覆盖短前缀“安”“安装”“安装指南”
	payloads := []string{
		`{"id":"d1","title":"安装指南","content":"完整安装操作步骤"}`,
		`{"id":"d2","title":"安装故障排查","content":"无法安装时的常见原因"}`,
		`{"id":"d3","title":"升级与安装策略","content":"版本升级兼容说明"}`,
	}
	for _, p := range payloads {
		req, _ := http.NewRequest(http.MethodPost, base+"/v1/docs", strings.NewReader(p))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("add doc err: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status add doc: %d", resp.StatusCode)
		}
	}

	// 给 ES 一点 refresh 时间（回退模式 refresh_interval=5s；测试强制刷新可选改成 ?refresh=true 的写入方式）
	time.Sleep(600 * time.Millisecond)

	type searchResp struct {
		Items []struct {
			ID      string  `json:"id"`
			Title   string  `json:"title"`
			Score   float64 `json:"score"`
			Snippet string  `json:"snippet"`
		} `json:"items"`
		Total int `json:"total"`
	}

	shortQueries := []string{"安", "安装"} // 长度≤4 的短查询，应通过 autocomplete 分支辅助召回
	for _, q := range shortQueries {
		u := base + "/v1/search?q=" + url.QueryEscape(q) + "&limit=5"
		resp, err := http.Get(u)
		if err != nil {
			t.Fatalf("search err (%s): %v", q, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("search status %d body=%s", resp.StatusCode, string(body))
		}
		var sr searchResp
		if err := json.Unmarshal(body, &sr); err != nil {
			t.Fatalf("decode search resp: %v body=%s", err, string(body))
		}
		if len(sr.Items) == 0 {
			t.Fatalf("short query %q expected hits >0", q)
		}
		// 只做基本断言：第一条命中标题中包含“安”或“安装”
		if !strings.Contains(sr.Items[0].Title, "安") {
			t.Errorf("expected first hit title contains 安; got %s", sr.Items[0].Title)
		}
	}

	// 验证较长查询（走常规 multi_match 路径）仍正常
	longQ := "安装指南"
	u := base + "/v1/search?q=" + url.QueryEscape(longQ)
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("long query err: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var sr2 searchResp
	if err := json.Unmarshal(body, &sr2); err != nil {
		t.Fatalf("decode long query resp: %v body=%s", err, string(body))
	}
	if sr2.Total == 0 {
		t.Fatalf("expected long query hits >0")
	}
}

// --- Phase3: KB REST 集成测试新增 ---

func TestKBDocUpdateAndDelete(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18088"} // memory backend
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})
	baseURL := "http://127.0.0.1:18088"
	waitReady(t, baseURL)

	// create doc
	doc := map[string]string{"title": "更新删除测试", "content": "初始内容"}
	b, _ := json.Marshal(doc)
	resp, err := http.Post(baseURL+"/v1/docs", contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create doc err: %v", err)
	}
	var createOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createOut)
	resp.Body.Close()
	if resp.StatusCode != 201 || createOut.ID == "" {
		t.Fatalf("unexpected create status=%d id=%s", resp.StatusCode, createOut.ID)
	}

	// update doc (PUT partial): change title & content
	upd := map[string]string{"title": "更新删除测试-修改", "content": "修改后的内容"}
	ub, _ := json.Marshal(upd)
	req, _ := http.NewRequest(http.MethodPut, baseURL+"/v1/docs/"+createOut.ID, bytes.NewReader(ub))
	req.Header.Set(headerContentType, contentTypeJSON)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update doc err: %v", err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200 on update, got %d", resp2.StatusCode)
	}

	// search for updated title fragment
	resp3, err := http.Get(baseURL + "/v1/search?q=修改&limit=5")
	if err != nil {
		t.Fatalf("search after update err: %v", err)
	}
	var searchOut struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(resp3.Body).Decode(&searchOut)
	resp3.Body.Close()
	if searchOut.Total == 0 || len(searchOut.Items) == 0 {
		t.Fatalf("expected search hit after update")
	}

	// delete doc
	dreq, _ := http.NewRequest(http.MethodDelete, baseURL+"/v1/docs/"+createOut.ID, nil)
	dresp, err := http.DefaultClient.Do(dreq)
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	dresp.Body.Close()
	if dresp.StatusCode != 204 {
		t.Fatalf("expected 204 on delete, got %d", dresp.StatusCode)
	}

	// search again should have zero hits for fragment
	resp4, err := http.Get(baseURL + "/v1/search?q=修改&limit=5")
	if err != nil {
		t.Fatalf("search after delete err: %v", err)
	}
	var searchOut2 struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(resp4.Body).Decode(&searchOut2)
	resp4.Body.Close()
	if searchOut2.Total != 0 || len(searchOut2.Items) != 0 {
		t.Fatalf("expected no hits after delete, got total=%d len=%d", searchOut2.Total, len(searchOut2.Items))
	}
}

func TestKBInfoEndpoint(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18089"} // memory backend
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})
	baseURL := "http://127.0.0.1:18089"
	waitReady(t, baseURL)

	resp, err := http.Get(baseURL + "/v1/kb/info")
	if err != nil {
		t.Fatalf("get kb info err: %v", err)
	}
	var info map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&info)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if info["backend"] != "memory" { // memory backend returns simple message
		t.Fatalf("expected backend=memory, got %#v", info)
	}
}

func TestKBSearchLimitEdgeCases(t *testing.T) {
	cfg := &common.Config{HTTPAddr: ":18090"} // memory backend
	h := BuildServer(cfg)
	go h.Spin()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	})
	baseURL := "http://127.0.0.1:18090"
	waitReady(t, baseURL)

	// seed many docs (>55) containing token BULK
	for i := 0; i < 60; i++ {
		doc := map[string]string{"title": fmt.Sprintf("bulk test %d", i), "content": "bulk test content"}
		b, _ := json.Marshal(doc)
		resp, err := http.Post(baseURL+"/v1/docs", contentTypeJSON, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("seed doc %d err: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 201 {
			t.Fatalf("seed doc %d unexpected status %d", i, resp.StatusCode)
		}
	}

	// limit=0 -> fallback to default (10)
	resp0, err := http.Get(baseURL + "/v1/search?q=bulk&limit=0")
	if err != nil {
		t.Fatalf("limit=0 search err: %v", err)
	}
	var s0 struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(resp0.Body).Decode(&s0)
	resp0.Body.Close()
	if len(s0.Items) == 0 || len(s0.Items) > 10 { // default 10 cap
		t.Fatalf("limit=0 expected 1..10 items, got %d", len(s0.Items))
	}

	// limit very large -> capped at 50
	respBig, err := http.Get(baseURL + "/v1/search?q=bulk&limit=999")
	if err != nil {
		t.Fatalf("limit=999 search err: %v", err)
	}
	var sb struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(respBig.Body).Decode(&sb)
	respBig.Body.Close()
	if len(sb.Items) == 0 || len(sb.Items) > 50 { // enforced upper bound
		t.Fatalf("limit=999 expected 1..50 items, got %d", len(sb.Items))
	}
	if sb.Total < len(sb.Items) { // total should be >= returned items
		t.Fatalf("total %d should be >= returned %d", sb.Total, len(sb.Items))
	}
}
