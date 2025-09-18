package kb

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type Doc struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Item struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

type Repo interface {
	// Add inserts a new document. If the same ID already exists, this behaves as an upsert
	// and will update the existing document atomically (equivalent to calling Update).
	Add(ctx context.Context, d *Doc) error
	// Get returns the document by id if present.
	Get(ctx context.Context, id string) (*Doc, bool)
	// Search finds matching documents for query q and returns (items, total).
	// items are sorted by score desc and truncated to 'limit'; total is the untruncated size.
	Search(ctx context.Context, q string, limit int) ([]*Item, int, error)
	// Update replaces the document with the same ID and updates indexes accordingly (upsert).
	// If the document does not exist, it will be inserted. The operation is atomic.
	Update(ctx context.Context, d *Doc) error
	// Delete removes a document and cleans up indexes.
	Delete(ctx context.Context, id string) error
}

type memoryRepo struct {
	mu   sync.RWMutex
	docs map[string]*Doc
	// inverted indexes counted by bigram -> docID -> count
	indexTitle map[string]map[string]int
	indexBody  map[string]map[string]int
	// per-doc ngram frequencies for proper update/delete bookkeeping
	gramsTitleByDoc map[string]map[string]int
	gramsBodyByDoc  map[string]map[string]int
	// n-gram size, default 2 (bigrams)
	ngramN int
}

func NewMemoryRepo() Repo {
	return &memoryRepo{
		docs:            map[string]*Doc{},
		indexTitle:      map[string]map[string]int{},
		indexBody:       map[string]map[string]int{},
		gramsTitleByDoc: map[string]map[string]int{},
		gramsBodyByDoc:  map[string]map[string]int{},
		ngramN:          2,
	}
}

// NewMemoryRepoWithN returns a memory repo configured to use n-grams of size n (n>=2 recommended).
func NewMemoryRepoWithN(n int) Repo {
	if n < 2 {
		n = 2
	}
	return &memoryRepo{
		docs:            map[string]*Doc{},
		indexTitle:      map[string]map[string]int{},
		indexBody:       map[string]map[string]int{},
		gramsTitleByDoc: map[string]map[string]int{},
		gramsBodyByDoc:  map[string]map[string]int{},
		ngramN:          n,
	}
}

// Add inserts a document; when the ID already exists, it performs an upsert while keeping
// the inverted index consistent (no leakage from the previous content).
// Concurrency: obtains a write lock; safe for concurrent use with Search/Update/Delete.
func (m *memoryRepo) Add(ctx context.Context, d *Doc) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// If doc exists, treat as Update to keep index consistent
	if _, ok := m.docs[d.ID]; ok {
		return m.upsertLocked(d)
	}
	m.docs[d.ID] = d
	// compute ngram counts
	tCounts := countNGrams(d.Title, m.ngramN)
	bCounts := countNGrams(d.Content, m.ngramN)
	// update inverted indexes
	for g, c := range tCounts {
		if m.indexTitle[g] == nil {
			m.indexTitle[g] = map[string]int{}
		}
		m.indexTitle[g][d.ID] += c
	}
	for g, c := range bCounts {
		if m.indexBody[g] == nil {
			m.indexBody[g] = map[string]int{}
		}
		m.indexBody[g][d.ID] += c
	}
	// store per-doc counts
	m.gramsTitleByDoc[d.ID] = tCounts
	m.gramsBodyByDoc[d.ID] = bCounts
	return nil
}

func (m *memoryRepo) Search(ctx context.Context, q string, limit int) ([]*Item, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return []*Item{}, 0, nil
	}
	if limit <= 0 {
		limit = 10
	}
	// bigram score first
	grams := toNGrams(q, m.ngramN)
	items := itemsFromIndex(grams, m.docs, m.indexTitle, m.indexBody)
	if len(items) == 0 { // fallback
		items = itemsFromSubstring(m.docs, q)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	total := len(items)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, total, nil
}

// Get returns a copy-safe pointer to the document and a boolean flag.
// Note: returns the stored pointer; callers must treat it as read-only.
func (m *memoryRepo) Get(ctx context.Context, id string) (*Doc, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.docs[id]
	return d, ok
}

func scoreDoc(d *Doc, q string) float64 {
	titleLower := strings.ToLower(d.Title)
	contentLower := strings.ToLower(d.Content)
	score := 0.0
	if strings.Contains(titleLower, q) {
		score += 2
	}
	if strings.Contains(contentLower, q) {
		score += 1
	}
	return score
}

func makeSnippet(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// toNGrams generates lowercased overlapping n-grams after stripping spaces and punctuation.
func toNGrams(s string, n int) []string {
	if n < 2 {
		n = 2
	}
	norm := normalizeRunes(s)
	if len(norm) < n {
		return nil
	}
	grams := make([]string, 0, len(norm)-n+1)
	for i := 0; i <= len(norm)-n; i++ {
		grams = append(grams, string(norm[i:i+n]))
	}
	return grams
}

func normalizeRunes(s string) []rune {
	b := strings.Builder{}
	for _, r := range strings.ToLower(s) {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		b.WriteRune(r)
	}
	return []rune(b.String())
}

func countNGrams(s string, n int) map[string]int {
	grams := toNGrams(s, n)
	if len(grams) == 0 {
		return map[string]int{}
	}
	m := make(map[string]int, len(grams))
	for _, g := range grams {
		m[g]++
	}
	return m
}

// itemsFromIndex builds items using inverted index scores.
func itemsFromIndex(grams []string, docs map[string]*Doc, idxTitle, idxBody map[string]map[string]int) []*Item {
	if len(grams) == 0 {
		return nil
	}
	scores := buildScores(grams, idxTitle, idxBody, len(docs))
	items := make([]*Item, 0, len(scores))
	for id, s := range scores {
		if s <= 0 {
			continue
		}
		if d, ok := docs[id]; ok {
			items = append(items, &Item{ID: d.ID, Title: d.Title, Snippet: makeSnippet(d.Content, 120), Score: s})
		}
	}
	return items
}

func buildScores(grams []string, idxTitle, idxBody map[string]map[string]int, numDocs int) map[string]float64 {
	if numDocs <= 0 || len(grams) == 0 {
		return map[string]float64{}
	}
	uniq := dedupStrings(grams)
	idf := computeIDF(uniq, idxTitle, idxBody, numDocs)
	return accumulateScores(uniq, idf, idxTitle, idxBody)
}

func dedupStrings(ss []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

func computeIDF(uniq map[string]struct{}, idxTitle, idxBody map[string]map[string]int, numDocs int) map[string]float64 {
	idf := make(map[string]float64, len(uniq))
	for g := range uniq {
		dfSet := map[string]struct{}{}
		if postings, ok := idxTitle[g]; ok {
			for docID := range postings {
				dfSet[docID] = struct{}{}
			}
		}
		if postings, ok := idxBody[g]; ok {
			for docID := range postings {
				dfSet[docID] = struct{}{}
			}
		}
		df := float64(len(dfSet))
		idf[g] = 1.0 + math.Log((1.0+float64(numDocs))/(1.0+df))
	}
	return idf
}

func accumulateScores(uniq map[string]struct{}, idf map[string]float64, idxTitle, idxBody map[string]map[string]int) map[string]float64 {
	scores := map[string]float64{}
	for g := range uniq {
		w := idf[g]
		if postings, ok := idxTitle[g]; ok {
			for docID, c := range postings {
				scores[docID] += float64(2*c) * w
			}
		}
		if postings, ok := idxBody[g]; ok {
			for docID, c := range postings {
				scores[docID] += float64(1*c) * w
			}
		}
	}
	return scores
}

func itemsFromSubstring(docs map[string]*Doc, q string) []*Item {
	out := []*Item{}
	for _, d := range docs {
		score := scoreDoc(d, q)
		if score <= 0 {
			continue
		}
		out = append(out, &Item{ID: d.ID, Title: d.Title, Snippet: makeSnippet(d.Content, 120), Score: score})
	}
	return out
}

// Update implements Repo.Update.
func (m *memoryRepo) Update(ctx context.Context, d *Doc) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.upsertLocked(d)
}

// upsertLocked performs an upsert for document d while the caller holds the write lock.
// Steps:
// 1) Remove previous per-doc n-gram counts from the inverted indexes (if any).
// 2) Recompute n-grams for the new content and add them back to the indexes.
// 3) Replace the stored document and refresh per-doc n-gram caches.
// Invariants after return:
//   - For any gram g: indexTitle[g][id] == gramsTitleByDoc[id][g] (or both absent)
//     indexBody[g][id]  == gramsBodyByDoc[id][g]  (or both absent)
//   - docs[id] == d
//
// Big-O: O(G_old + G_new), where G_* is the number of n-grams in old/new content.
// Note: ctx is unused here; cancellation should be handled by the caller before locking.
func (m *memoryRepo) upsertLocked(d *Doc) error {
	m.removeDocFromIndexNoLock(d.ID)
	m.addDocToIndexNoLock(d)
	return nil
}

// removeDocFromIndexNoLock subtracts the stored per-doc n-gram counts from the field indexes.
// No-op if the document has no cached grams (i.e., not present).
func (m *memoryRepo) removeDocFromIndexNoLock(id string) {
	m.removeFieldIndex(id, m.gramsTitleByDoc[id], m.indexTitle)
	m.removeFieldIndex(id, m.gramsBodyByDoc[id], m.indexBody)
}

// removeFieldIndex applies negative deltas for a document across all grams of a single field,
// removing empty postings and gram keys when counts drop to zero.
func (m *memoryRepo) removeFieldIndex(id string, grams map[string]int, index map[string]map[string]int) {
	if grams == nil {
		return
	}
	for g, c := range grams {
		if postings := index[g]; postings != nil {
			postings[id] -= c
			if postings[id] <= 0 {
				delete(postings, id)
			}
			if len(postings) == 0 {
				delete(index, g)
			}
		}
	}
}

// addDocToIndexNoLock (re)computes per-doc gram counts and adds them to indexes and caches.
func (m *memoryRepo) addDocToIndexNoLock(d *Doc) {
	tCounts := countNGrams(d.Title, m.ngramN)
	bCounts := countNGrams(d.Content, m.ngramN)
	for g, c := range tCounts {
		if m.indexTitle[g] == nil {
			m.indexTitle[g] = map[string]int{}
		}
		m.indexTitle[g][d.ID] += c
	}
	for g, c := range bCounts {
		if m.indexBody[g] == nil {
			m.indexBody[g] = map[string]int{}
		}
		m.indexBody[g][d.ID] += c
	}
	m.docs[d.ID] = d
	m.gramsTitleByDoc[d.ID] = tCounts
	m.gramsBodyByDoc[d.ID] = bCounts
}

// Delete implements Repo.Delete.
func (m *memoryRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.docs[id]; !ok {
		return nil
	}
	m.removeDocFromIndexNoLock(id)
	delete(m.docs, id)
	delete(m.gramsTitleByDoc, id)
	delete(m.gramsBodyByDoc, id)
	return nil
}
