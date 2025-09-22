package main

// Canonical gateway integration tests (ports 18201-18211 ONLY)
// Business rule: escalate after resolve => 409 Conflict.
// Keep minimal; add new integration scenarios in separate files if ports exhausted.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogogo1024/assist-fusion/internal/common"
	router "github.com/gogogo1024/assist-fusion/services/gateway/internal/router"
)

const (
	contentTypeJSON       = "application/json"
	headerContentTypeTest = "Content-Type"
	errServerNotReadyFmt  = "server not ready at %s"
	errExpect200Fmt       = "expected 200, got %d"
	docsPath              = "/v1/docs"
	ticketPrefix          = "/v1/tickets/"
	// legacy alias retained for compatibility after route refactor
	pathTickets  = router.PathTickets
	searchURLFmt = "%s/v1/search?q=%s&limit=%d"
)

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

// --- shared RPC backend startup (once) ---
var (
	onceRPC    sync.Once
	rpcStarted bool
)

func setupOnce(t *testing.T) {
	onceRPC.Do(func() {
		_, stops := startAllRPC(t)
		// Set env vars so BuildServer picks direct host:port addresses (DISABLE_CONSUL already set in startAllRPC).
		// startAllRPC already initialized rpc clients.
		// We rely on global clients inside gateway adapter.
		// Stop functions intentionally not deferred here to keep services for entire test package duration.
		rpcStarted = true
		_ = stops // ignored; not stopping within test run to avoid races
	})
	if !rpcStarted {
		// should never happen; defensive
		t.Fatalf("RPC backends not started")
	}
}

// --- helpers ---
func waitReady(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(base + "/health"); err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if r2, err2 := http.Get(base + "/ready"); err2 == nil {
					io.Copy(io.Discard, r2.Body)
					r2.Body.Close()
				}
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf(errServerNotReadyFmt, base)
}

func buildServer(t *testing.T, port string) (base string, stop func()) {
	t.Helper()
	cfg := &common.Config{HTTPAddr: port}
	h := BuildServer(cfg)
	go h.Spin()
	base = "http://127.0.0.1" + port
	waitReady(t, base)
	stop = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		h.Shutdown(ctx)
	}
	return
}

func putAndDecode(t *testing.T, u string, out any) (code int) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, u, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s err=%v", u, err)
	}
	defer resp.Body.Close()
	code = resp.StatusCode
	if out != nil && (code == http.StatusOK || code == http.StatusCreated) {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s: %v", u, err)
		}
	} else {
		io.Copy(io.Discard, resp.Body)
	}
	return
}

func createTicket(t *testing.T, baseURL, title, desc string) ticketResp {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"title": title, "desc": desc})
	resp, err := http.Post(baseURL+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create ticket err=%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create ticket status=%d body=%s", resp.StatusCode, string(raw))
	}
	var tk ticketResp
	if err := json.NewDecoder(resp.Body).Decode(&tk); err != nil {
		t.Fatalf("decode create resp: %v", err)
	}
	if tk.ID == "" || tk.Status != "created" || tk.CreatedAt == 0 {
		t.Fatalf("unexpected created ticket: %#v", tk)
	}
	return tk
}

func doAction(t *testing.T, baseURL, id, action string) (ticketResp, int) {
	t.Helper()
	var tk ticketResp
	code := putAndDecode(t, baseURL+ticketPrefix+id+"/"+action, &tk)
	return tk, code
}

// ---- Tests (ports 18201-18211) ----
func TestTicketAndAIFlow(t *testing.T) { // :18201
	// Ensure shared RPC backends started once.
	setupOnce(t)
	base, stop := buildServer(t, ":18201")
	defer stop()
	b, _ := json.Marshal(map[string]string{"title": "test", "desc": "hello"})
	resp, err := http.Post(base+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post ticket: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	embReq := map[string]any{"texts": []string{"hello"}, "dim": 4}
	eb, _ := json.Marshal(embReq)
	resp2, err := http.Post(base+"/v1/embeddings", contentTypeJSON, bytes.NewReader(eb))
	if err != nil {
		t.Fatalf("post embeddings: %v", err)
	}
	var out struct {
		Vectors [][]float64 `json:"vectors"`
		Dim     int         `json:"dim"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&out)
	resp2.Body.Close()
	if len(out.Vectors) != 1 || out.Dim != 4 {
		t.Fatalf("unexpected embeddings resp: %#v", out)
	}
}

func TestKBFlow(t *testing.T) { // :18202
	setupOnce(t)
	base, stop := buildServer(t, ":18202")
	defer stop()
	doc := map[string]string{"title": "FAQ 客服", "content": "客服如何升级？请参考SLA"}
	db, _ := json.Marshal(doc)
	resp, err := http.Post(base+docsPath, contentTypeJSON, bytes.NewReader(db))
	if err != nil {
		t.Fatalf("post doc: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for docs, got %d", resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	resp2, err := http.Get(base + "/v1/search?q=客服")
	if err != nil {
		t.Fatalf("get search: %v", err)
	}
	var out struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&out)
	resp2.Body.Close()
	if out.Total < 1 {
		t.Fatalf("expected at least one search result, got %#v", out)
	}
}

func TestTicketLifecycleNegativeAndList(t *testing.T) { // :18203
	setupOnce(t)
	base, stop := buildServer(t, ":18203")
	defer stop()
	b, _ := json.Marshal(map[string]string{"title": "lifecycle", "desc": "demo"})
	resp, err := http.Post(base+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create ticket err=%v code=%d", err, resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	resp, err = http.Get(base + pathTickets)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list err=%v code=%d", err, resp.StatusCode)
	}
	resp.Body.Close()
	for _, p := range []string{"assign", "resolve", "escalate"} {
		req, _ := http.NewRequest(http.MethodPut, base+ticketPrefix+"does-not-exist/"+p, nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s expected 404 got err=%v code=%d", p, err, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestTicketLifecyclePositive(t *testing.T) { // :18204
	setupOnce(t)
	base, stop := buildServer(t, ":18204")
	defer stop()
	tk := createTicket(t, base, "positive", "lifecycle")
	tk, _ = doAction(t, base, tk.ID, "assign")
	tk, _ = doAction(t, base, tk.ID, "escalate")
	tk, code := doAction(t, base, tk.ID, "resolve")
	if code != http.StatusOK || tk.Status != "resolved" || tk.ResolvedAt == 0 || tk.EscalatedAt == 0 {
		t.Fatalf("unexpected resolved ticket: %#v", tk)
	}
	_, code = doAction(t, base, tk.ID, "escalate")
	if code != http.StatusConflict {
		t.Fatalf("expected 409 escalate after resolve got %d", code)
	}
	tk, code = doAction(t, base, tk.ID, "reopen")
	if code != http.StatusOK || tk.Status != "created" || tk.ReopenedAt == 0 || tk.AssignedAt != 0 || tk.ResolvedAt != 0 || tk.EscalatedAt != 0 {
		t.Fatalf("unexpected reopen ticket: %#v", tk)
	}
}

func TestTicketCyclesAPI(t *testing.T) { // :18205
	setupOnce(t)
	base, stop := buildServer(t, ":18205")
	defer stop()
	tk := createTicket(t, base, "cycles", "inspect")
	tk, _ = doAction(t, base, tk.ID, "assign")
	tk, _ = doAction(t, base, tk.ID, "escalate")
	tk, _ = doAction(t, base, tk.ID, "resolve")
	tk, _ = doAction(t, base, tk.ID, "reopen")
	resp, err := http.Get(base + ticketPrefix + tk.ID + "/cycles")
	if err != nil {
		t.Fatalf("get cycles err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
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
}

func TestTicketGetIncludesCycles(t *testing.T) { // :18206
	setupOnce(t)
	base, stop := buildServer(t, ":18206")
	defer stop()
	tk := createTicket(t, base, "detail", "view cycles")
	tk, _ = doAction(t, base, tk.ID, "assign")
	tk, _ = doAction(t, base, tk.ID, "resolve")
	tk, _ = doAction(t, base, tk.ID, "reopen")
	resp, err := http.Get(base + ticketPrefix + tk.ID)
	if err != nil {
		t.Fatalf("get ticket err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf(errExpect200Fmt, resp.StatusCode)
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
}

func TestTicketEventsAPI(t *testing.T) { // :18207
	setupOnce(t)
	base, stop := buildServer(t, ":18207")
	defer stop()
	b, _ := json.Marshal(map[string]string{"title": "events", "desc": "audit", "note": "new ticket"})
	resp, err := http.Post(base+pathTickets, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create with note: %v", err)
	}
	var tk ticketResp
	_ = json.NewDecoder(resp.Body).Decode(&tk)
	resp.Body.Close()
	for _, a := range []struct{ A, Note string }{{"assign", "assigning"}, {"escalate", "urgent"}, {"resolve", "fixed"}, {"reopen", "wrong fix"}} {
		req, _ := http.NewRequest(http.MethodPut, base+ticketPrefix+tk.ID+"/"+a.A, bytes.NewReader([]byte(fmt.Sprintf(`{"note":"%s"}`, a.Note))))
		req.Header.Set(headerContentType, contentTypeJSON)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s with note err: %v", a.A, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	resp, err = http.Get(base + ticketPrefix + tk.ID + "/events")
	if err != nil {
		t.Fatalf("get events err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf(errExpect200Fmt, resp.StatusCode)
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
		t.Fatalf("expected 5 events, got %d", len(out.Events))
	}
}

func TestKBDocUpdateAndDelete(t *testing.T) { // :18208
	setupOnce(t)
	base, stop := buildServer(t, ":18208")
	defer stop()
	doc := map[string]string{"title": "更新删除测试", "content": "初始内容"}
	b, _ := json.Marshal(doc)
	resp, err := http.Post(base+docsPath, contentTypeJSON, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create doc err: %v", err)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated || created.ID == "" {
		t.Fatalf("unexpected create status=%d id=%s", resp.StatusCode, created.ID)
	}
	upd := map[string]string{"title": "更新删除测试-修改", "content": "修改后的内容"}
	ub, _ := json.Marshal(upd)
	req, _ := http.NewRequest(http.MethodPut, base+docsPath+"/"+created.ID, bytes.NewReader(ub))
	req.Header.Set(headerContentType, contentTypeJSON)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update doc err: %v", err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf(errExpect200Fmt, resp2.StatusCode)
	}
	resp3, err := http.Get(base + "/v1/search?q=修改&limit=5")
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
	dreq, _ := http.NewRequest(http.MethodDelete, base+docsPath+"/"+created.ID, nil)
	dresp, err := http.DefaultClient.Do(dreq)
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	io.Copy(io.Discard, dresp.Body)
	dresp.Body.Close()
	if dresp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", dresp.StatusCode)
	}
	resp4, err := http.Get(base + "/v1/search?q=修改&limit=5")
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

func TestKBInfoEndpoint(t *testing.T) { // :18209
	setupOnce(t)
	base, stop := buildServer(t, ":18209")
	defer stop()
	resp, err := http.Get(base + "/v1/kb/info")
	if err != nil {
		t.Fatalf("get kb info err: %v", err)
	}
	var info map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&info)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf(errExpect200Fmt, resp.StatusCode)
	}
	if info["backend"] != "memory" {
		t.Fatalf("expected backend=memory, got %#v", info)
	}
}

func TestKBSearchLimitEdgeCases(t *testing.T) { // :18210
	setupOnce(t)
	base, stop := buildServer(t, ":18210")
	defer stop()
	seedBulkDocs(t, base, 60)
	items0, _ := searchItems(t, base, "bulk", 0)
	if len(items0) == 0 || len(items0) > 10 {
		t.Fatalf("limit=0 expected 1..10 items got %d", len(items0))
	}
	itemsMax, totalMax := searchItems(t, base, "bulk", 999)
	if len(itemsMax) == 0 || len(itemsMax) > 50 {
		t.Fatalf("limit=999 expected 1..50 items got %d", len(itemsMax))
	}
	if totalMax < len(itemsMax) {
		t.Fatalf("total %d < returned %d", totalMax, len(itemsMax))
	}
	itemsSmall, totalSmall := searchItems(t, base, "bulk", 5)
	if totalSmall < len(itemsSmall) {
		t.Fatalf("(limit=5) total %d < returned %d", totalSmall, len(itemsSmall))
	}
}

// --- local helpers for KB search limit edge cases ---
func seedBulkDocs(t *testing.T, base string, count int) {
	for i := 0; i < count; i++ {
		doc := map[string]string{"title": fmt.Sprintf("bulk test %d", i), "content": "bulk test content"}
		b, _ := json.Marshal(doc)
		resp, err := http.Post(base+docsPath, contentTypeJSON, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("seed doc %d err: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("seed doc %d status %d", i, resp.StatusCode)
		}
	}
}

func searchItems(t *testing.T, base, q string, limit int) ([]map[string]any, int) {
	url := fmt.Sprintf("%s/v1/search?q=%s&limit=%d", base, q, limit)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("search %s err: %v", url, err)
	}
	var out struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	return out.Items, out.Total
}

func TestSearchShortQueryAutocomplete(t *testing.T) { // :18211
	if os.Getenv("KB_BACKEND") != "es" {
		t.Skip("skip: KB_BACKEND!=es")
	}
	base, stop := buildServer(t, ":18211")
	defer stop()
	payloads := []string{
		`{"id":"d1","title":"安装指南","content":"完整安装操作步骤"}`,
		`{"id":"d2","title":"安装故障排查","content":"无法安装时的常见原因"}`,
		`{"id":"d3","title":"升级与安装策略","content":"版本升级兼容说明"}`,
	}
	for _, p := range payloads {
		req, _ := http.NewRequest(http.MethodPost, base+docsPath, strings.NewReader(p))
		req.Header.Set(headerContentType, contentTypeJSON)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("add doc err: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected add status %d", resp.StatusCode)
		}
	}
	time.Sleep(600 * time.Millisecond)
	type searchResp struct {
		Items []struct{ ID, Title, Content string } `json:"items"`
	}
	resp, err := http.Get(base + "/v1/search?q=安&limit=5")
	if err != nil {
		t.Fatalf("autocomplete search err: %v", err)
	}
	var sr searchResp
	_ = json.NewDecoder(resp.Body).Decode(&sr)
	resp.Body.Close()
	if len(sr.Items) == 0 {
		t.Fatalf("expected at least one suggestion: %#v", sr)
	}
	u := base + "/v1/search?q=" + url.QueryEscape("安") + "&limit=5"
	resp2, err := http.Get(u)
	if err != nil {
		t.Fatalf("escaped query err: %v", err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
}

func TestKBSearchPaginationNextOffset(t *testing.T) { // :18212
	setupOnce(t)
	base, stop := buildServer(t, ":18212")
	defer stop()
	// Seed a number of docs larger than one page; we don't assert exact total, only internal consistency.
	seedDocs(t, base, 23, func(i int) (string, string) { return fmt.Sprintf("page test %02d", i), "paginations" })
	walkPagesDynamicTotal(t, base, "page", 7)
}

// seedDocs seeds n docs using a title/content generator.
func seedDocs(t *testing.T, base string, n int, gen func(i int) (title, content string)) {
	for i := 0; i < n; i++ {
		title, content := gen(i)
		b, _ := json.Marshal(map[string]string{"title": title, "content": content})
		resp, err := http.Post(base+docsPath, contentTypeJSON, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("seed doc %d err: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("seed doc %d status %d", i, resp.StatusCode)
		}
	}
}

// walkPages pages through search results asserting next_offset progression and total consistency.

// walkPagesDynamicTotal pages through results without prior knowledge of total; it validates:
// - total is constant across pages
// - next_offset is cumulative returned count
// - collected items == total on terminal page
func walkPagesDynamicTotal(t *testing.T, base, q string, limit int) {
	var totalObserved *int
	var offset *int
	collected := 0
	for page := 0; ; page++ {
		out := fetchPage(t, base, q, limit, offset)
		assertReturned(t, page, out.Returned, len(out.Items))
		totalObserved = assertTotalStable(t, totalObserved, out.Total)
		collected += out.Returned
		if out.NextOffset == nil {
			if collected != *totalObserved {
				t.Fatalf("collected %d final vs total %d", collected, *totalObserved)
			}
			return
		}
		assertNextOffset(t, page, *out.NextOffset, collected)
		offset = out.NextOffset
		if page > 16 {
			t.Fatalf("too many pages")
		}
	}
}

// helper subset for complexity reduction
func fetchPage(t *testing.T, base, q string, limit int, offset *int) struct {
	Items      []map[string]any `json:"items"`
	Total      int              `json:"total"`
	Returned   int              `json:"returned"`
	NextOffset *int             `json:"next_offset"`
} {
	url := fmt.Sprintf(searchURLFmt, base, q, limit)
	if offset != nil {
		url = fmt.Sprintf("%s&offset=%d", url, *offset)
	}
	var out struct {
		Items      []map[string]any `json:"items"`
		Total      int              `json:"total"`
		Returned   int              `json:"returned"`
		NextOffset *int             `json:"next_offset"`
	}
	getJSON(t, url, &out)
	return out
}

func assertReturned(t *testing.T, page, returned, itemsLen int) {
	if returned == 0 || returned != itemsLen {
		t.Fatalf("page %d returned mismatch", page)
	}
}

func assertTotalStable(t *testing.T, prev *int, current int) *int {
	if prev == nil {
		return &current
	}
	if *prev != current {
		t.Fatalf("total changed %d -> %d", *prev, current)
	}
	return prev
}

func assertNextOffset(t *testing.T, page, got, expect int) {
	if got != expect {
		t.Fatalf("page %d expect next_offset=%d got %d", page, expect, got)
	}
}

// firstPageTotal executes first page and returns returnedCount, totalReported.

// getJSON performs a GET and decodes JSON into out.
func getJSON(t *testing.T, url string, out any) {
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s err: %v", url, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s err: %v", url, err)
	}
}

// End canonical gateway integration test suite.
