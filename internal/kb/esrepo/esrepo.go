package esrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gogogo1024/assist-fusion/internal/kb"
)

// Config for Elasticsearch repo.
// Addresses: list of http(s) endpoints, e.g. ["http://localhost:9200"].
// Index: index name, default "kb_docs".
// Basic auth optional.
type Config struct {
	Addresses []string
	Index     string
	Username  string
	Password  string
}

type Repo struct {
	cli   *elasticsearch.Client
	index string
}

func New(cfg Config) (*Repo, error) {
	if len(cfg.Addresses) == 0 {
		cfg.Addresses = []string{"http://localhost:9200"}
	}
	if cfg.Index == "" {
		cfg.Index = "kb_docs"
	}
	esCfg := elasticsearch.Config{Addresses: cfg.Addresses}
	if cfg.Username != "" || cfg.Password != "" {
		esCfg.Username = cfg.Username
		esCfg.Password = cfg.Password
	}
	cli, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}
	return &Repo{cli: cli, index: cfg.Index}, nil
}

// ensureIndex creates the index with a minimal mapping if it doesn't exist.
func (r *Repo) ensureIndex(ctx context.Context) error {
	res, err := r.cli.Indices.Exists([]string{r.index})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		return nil
	}
	// 1st attempt: IK analyzers (requires ik plugin). Index analyzer: ik_max_word; search analyzer: ik_smart.
	ikBody := `{
			"settings": {
				"analysis": {
					"analyzer": {
						"cn_index":  {"type": "custom", "tokenizer": "ik_max_word", "filter": ["lowercase"]},
						"cn_search": {"type": "custom", "tokenizer": "ik_smart",    "filter": ["lowercase"]}
					}
				}
			},
			"mappings": {"properties": {
				"title":   {"type": "text", "analyzer": "cn_index",  "search_analyzer": "cn_search"},
				"content": {"type": "text", "analyzer": "cn_index",  "search_analyzer": "cn_search"}
			}}
		}`
	cr := esapi.IndicesCreateRequest{Index: r.index, Body: strings.NewReader(ikBody)}
	cres, err := cr.Do(ctx, r.cli)
	if err == nil && cres != nil && cres.StatusCode < 300 {
		defer cres.Body.Close()
		return nil
	}
	// If IK not installed or creation failed, fallback to ngram-based Chinese-friendly mapping (no plugin needed).
	if cres != nil {
		defer cres.Body.Close()
	}
	ngramBody := `{
			"settings": {
				"refresh_interval": "5s",
				"analysis": {
					"tokenizer": {
						"cn_ngram3":      {"type": "ngram",       "min_gram": 3, "max_gram": 3,  "token_chars": ["letter","digit"]},
						"cn_edge_ngram":  {"type": "edge_ngram",  "min_gram": 2, "max_gram": 12, "token_chars": ["letter","digit"]}
					},
					"analyzer": {
						"cn_index_title":   {"type": "custom", "tokenizer": "cn_ngram3",     "filter": ["lowercase"]},
						"cn_index_content": {"type": "custom", "tokenizer": "cn_ngram3",     "filter": ["lowercase"]},
						"cn_autocomplete":  {"type": "custom", "tokenizer": "cn_edge_ngram", "filter": ["lowercase"]},
						"cn_search":        {"type": "custom", "tokenizer": "standard",      "filter": ["lowercase"]}
					}
				}
			},
			"mappings": {"properties": {
				"title": {
					"type": "text",
					"analyzer": "cn_index_title",
					"search_analyzer": "cn_search",
					"fields": {
						"autocomplete": {"type": "text", "analyzer": "cn_autocomplete", "search_analyzer": "cn_search"}
					}
				},
				"content": {"type": "text", "analyzer": "cn_index_content",  "search_analyzer": "cn_search"}
			}}
		}`
	cr2 := esapi.IndicesCreateRequest{Index: r.index, Body: strings.NewReader(ngramBody)}
	cres2, err2 := cr2.Do(ctx, r.cli)
	if err2 != nil {
		return err2
	}
	defer cres2.Body.Close()
	if cres2.StatusCode >= 300 {
		return fmt.Errorf("create index failed (fallback): %s", cres2.String())
	}
	return nil
}

func (r *Repo) Add(ctx context.Context, d *kb.Doc) error {
	return r.Update(ctx, d)
}

func (r *Repo) Get(ctx context.Context, id string) (*kb.Doc, bool) {
	gr := esapi.GetRequest{Index: r.index, DocumentID: id}
	res, err := gr.Do(ctx, r.cli)
	if err != nil {
		return nil, false
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, false
	}
	if res.StatusCode >= 300 {
		return nil, false
	}
	// minimal parse: use _source passthrough via gjson-like or stdlib; to keep deps minimal, use a tiny manual decode
	type hit struct {
		Source kb.Doc `json:"_source"`
	}
	var h hit
	if err := decodeJSON(res.Body, &h); err != nil {
		return nil, false
	}
	return &h.Source, true
}

func (r *Repo) Search(ctx context.Context, q string, limit int) ([]*kb.Item, int, error) {
	if err := r.ensureIndex(ctx); err != nil {
		return nil, 0, err
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return []*kb.Item{}, 0, nil
	}
	if limit <= 0 {
		limit = 10
	}
	query := buildSearchQuery(q, limit)
	sr := esapi.SearchRequest{Index: []string{r.index}, Body: strings.NewReader(query)}
	res, err := sr.Do(ctx, r.cli)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("search failed: %s", res.String())
	}
	items, total, err := parseSearchResponse(res)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func buildSearchQuery(q string, limit int) string {
	// If query is short (<= 4 runes), add a should clause against title.autocomplete to improve precision
	short := len([]rune(q)) <= 4
	if short {
		return fmt.Sprintf(`{
	"size": %d,
	"query": {
		"bool": {
			"should": [
				{"multi_match": {"query": %q, "fields": ["title^2","content^1"], "type": "best_fields"}},
				{"match": {"title.autocomplete": {"query": %q, "boost": 1.2}}}
			],
			"minimum_should_match": 1
		}
	},
	"highlight": {"fields": {"content": {"fragment_size": 80, "number_of_fragments": 1}}}
}`, limit, q, q)
	}
	return fmt.Sprintf(`{
	"size": %d,
	"query": {"multi_match": {"query": %q, "fields": ["title^2","content^1"], "type": "best_fields"}},
	"highlight": {"fields": {"content": {"fragment_size": 120, "number_of_fragments": 1}}}
}`, limit, q)
}

func parseSearchResponse(res *esapi.Response) ([]*kb.Item, int, error) {
	var resp struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string         `json:"_id"`
				Score  float64        `json:"_score"`
				Source kb.Doc         `json:"_source"`
				HL     map[string]any `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := decodeJSON(res.Body, &resp); err != nil {
		return nil, 0, err
	}
	items := make([]*kb.Item, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		snippet := h.Source.Content
		if v, ok := h.HL["content"]; ok {
			if arr, aok := v.([]any); aok && len(arr) > 0 {
				if s, sok := arr[0].(string); sok {
					snippet = stripTags(s)
				}
			}
		}
		items = append(items, &kb.Item{ID: h.ID, Title: h.Source.Title, Snippet: snippet, Score: h.Score})
	}
	return items, resp.Hits.Total.Value, nil
}

func (r *Repo) Update(ctx context.Context, d *kb.Doc) error {
	if d == nil || d.ID == "" {
		return errors.New("invalid doc")
	}
	if err := r.ensureIndex(ctx); err != nil {
		return err
	}
	payload := fmt.Sprintf(`{"title":%q,"content":%q}`, d.Title, d.Content)
	ir := esapi.IndexRequest{Index: r.index, DocumentID: d.ID, Body: strings.NewReader(payload), Refresh: "true"}
	res, err := ir.Do(ctx, r.cli)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("index failed: %s", res.String())
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	dr := esapi.DeleteRequest{Index: r.index, DocumentID: id, Refresh: "true"}
	res, err := dr.Do(ctx, r.cli)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("delete failed: %s", res.String())
	}
	return nil
}

// Ping performs a lightweight health check against the ES cluster with a short timeout.
// It uses the Info API which does not require the target index to exist and returns quickly.
// Caller should wrap with its own timeout; we still defensively enforce one here.
func (r *Repo) Ping(ctx context.Context) error {
	// ensure we don't block readiness; default to 500ms if parent has no earlier deadline
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
	}
	// Use Info API (HEAD /) via client helper; provide context if supported
	res, err := r.cli.Info(r.cli.Info.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("es info status %d", res.StatusCode)
	}
	return nil
}

// --- tiny helpers (no extra deps) ---

// Info returns minimal KB index information for diagnostics.
// mode: "ik" | "ngram" | "standard"
func (r *Repo) Info(ctx context.Context) (map[string]any, error) {
	// Try to get index settings and infer analyzer mode
	gr := esapi.IndicesGetRequest{Index: []string{r.index}}
	res, err := gr.Do(ctx, r.cli)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return map[string]any{"index": r.index, "mode": "unknown"}, nil
	}
	// Response shape: { "<index>": { "settings": { "index": { "analysis": { ... }}}}}
	var raw map[string]struct {
		Settings struct {
			Index struct {
				Analysis struct {
					Analyzer map[string]struct {
						Tokenizer string `json:"tokenizer"`
					} `json:"analyzer"`
				} `json:"analysis"`
			} `json:"index"`
		} `json:"settings"`
	}
	if err := decodeJSON(res.Body, &raw); err != nil {
		return nil, err
	}
	mode := "standard"
	if v, ok := raw[r.index]; ok {
		for name, an := range v.Settings.Index.Analysis.Analyzer {
			_ = name
			if strings.Contains(strings.ToLower(an.Tokenizer), "ik_") {
				mode = "ik"
				break
			}
			if strings.Contains(strings.ToLower(an.Tokenizer), "ngram") {
				mode = "ngram"
				// keep checking in case IK also present, but typically not
			}
		}
	}
	return map[string]any{"index": r.index, "mode": mode}, nil
}

func decodeJSON(rdr any, out any) error {
	rc, ok := rdr.(interface{ Read([]byte) (int, error) })
	if !ok {
		return fmt.Errorf("invalid reader")
	}
	b := new(strings.Builder)
	buf := make([]byte, 4096)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	dec := jsonNewDecoder(strings.NewReader(b.String()))
	return dec.Decode(out)
}

func stripTags(s string) string {
	res := make([]rune, 0, len(s))
	in := false
	for _, r := range s {
		switch r {
		case '<':
			in = true
		case '>':
			in = false
		default:
			if !in {
				res = append(res, r)
			}
		}
	}
	return string(res)
}

// wrap std json decoder behind a minimal interface to keep imports local to this file.
// We do this to avoid alias conflicts if the repo already imports other json libs elsewhere.

type jsonDecoder interface{ Decode(v any) error }

type jsonDecoderImpl struct{ *json.Decoder }

func (j jsonDecoderImpl) Decode(v any) error { return j.Decoder.Decode(v) }

func jsonNewDecoder(r *strings.Reader) jsonDecoder { return jsonDecoderImpl{json.NewDecoder(r)} }
