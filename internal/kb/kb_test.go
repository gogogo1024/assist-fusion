package kb

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

const errAddFmt = "add: %v"

func TestMemoryRepoAddAndSearch(t *testing.T) {
	repo := NewMemoryRepo()
	// add documents
	docs := []Doc{
		{ID: "1", Title: "客服入门", Content: "什么是客服，如何开始"},
		{ID: "2", Title: "升级指南", Content: "如何升级客服流程"},
		{ID: "3", Title: "FAQ", Content: "常见问题：客服、排班、SLA"},
	}
	for i := range docs {
		if err := repo.Add(context.TODO(), &docs[i]); err != nil {
			t.Fatalf("add doc %d: %v", i, err)
		}
	}

	// empty query returns empty
	items, total, err := repo.Search(context.TODO(), " ", 10)
	if err != nil {
		t.Fatalf("search empty: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("expected empty result, got total=%d len=%d", total, len(items))
	}

	// keyword
	items, total, err = repo.Search(context.TODO(), "客服", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total < 2 {
		t.Fatalf("expected at least 2 results, got %d", total)
	}
	// ensure ordered by score desc
	for i := 1; i < len(items); i++ {
		if items[i].Score > items[i-1].Score {
			t.Fatalf("result not sorted desc at %d: %f > %f", i, items[i].Score, items[i-1].Score)
		}
	}

	// limit
	items, total, err = repo.Search(context.TODO(), "客服", 1)
	if err != nil {
		t.Fatalf("search limit: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected len=1, got %d (total=%d)", len(items), total)
	}
}

func TestUpdateAndDeleteIndexConsistency(t *testing.T) {
	repo := NewMemoryRepo()
	d := Doc{ID: "x1", Title: "安装指南", Content: "介绍安装流程"}
	if err := repo.Add(context.TODO(), &d); err != nil {
		t.Fatalf(errAddFmt, err)
	}
	// should hit by "安装"
	_, total, _ := repo.Search(context.TODO(), "安装", 10)
	if total == 0 {
		t.Fatalf("expected hits for 安装 before update")
	}
	// update to different content without "安装"
	d2 := Doc{ID: d.ID, Title: "排错手册", Content: "介绍排错与诊断"}
	if err := repo.Update(context.TODO(), &d2); err != nil {
		t.Fatalf("update: %v", err)
	}
	_, total, _ = repo.Search(context.TODO(), "安装", 10)
	// may still hit via substring fallback only if any field contains query; should be 0 now
	if total != 0 {
		t.Fatalf("expected 0 hit after update for 安装, got %d", total)
	}
	// new query should hit
	_, total, _ = repo.Search(context.TODO(), "排错", 10)
	if total == 0 {
		t.Fatalf("expected hits for 排错 after update")
	}
	// delete
	if err := repo.Delete(context.TODO(), d.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, total, _ = repo.Search(context.TODO(), "排错", 10)
	if total != 0 {
		t.Fatalf("expected 0 hit after delete, got %d", total)
	}
}

func TestUTF8SafeSnippet(t *testing.T) {
	repo := NewMemoryRepo()
	// build a long Chinese content > 200 runes
	long := strings.Repeat("客服系统很重要。", 20) // each sentence ~7 runes, 140+ runes total
	d := Doc{ID: "u1", Title: "关于客服", Content: long}
	if err := repo.Add(context.TODO(), &d); err != nil {
		t.Fatalf(errAddFmt, err)
	}
	items, total, _ := repo.Search(context.TODO(), "客服", 1)
	if total == 0 || len(items) == 0 {
		t.Fatalf("expected hit for 客服")
	}
	sn := items[0].Snippet
	if !utf8.ValidString(sn) {
		t.Fatalf("snippet is not valid utf8")
	}
	// snippet should be at most 120 runes
	if rc := utf8.RuneCountInString(sn); rc > 120 {
		t.Fatalf("snippet rune count %d > 120", rc)
	}
}

func TestNGramConfiguration(t *testing.T) {
	// trigram repo
	repo := NewMemoryRepoWithN(3)
	d := Doc{ID: "n3", Title: "安装指南", Content: "快速开始"}
	if err := repo.Add(context.TODO(), &d); err != nil {
		t.Fatalf(errAddFmt, err)
	}
	// short query of length 2 should trigger fallback (no trigrams), still find via substring
	_, total, _ := repo.Search(context.TODO(), "安装", 10)
	if total == 0 {
		t.Fatalf("expected fallback hit for 安装 with trigram repo")
	}
	// full query with length >=3 should hit via index as well
	_, total, _ = repo.Search(context.TODO(), "安装指南", 10)
	if total == 0 {
		t.Fatalf("expected indexed hit for 安装指南 with trigram repo")
	}
}
