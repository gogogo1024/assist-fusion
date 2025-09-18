package handler

import (
	"context"
	"strconv"

	"github.com/gogogo1024/assist-fusion/internal/kb"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	"github.com/google/uuid"
)

// KBServiceImpl implements the generated KBService interface using kb.Repo.
type KBServiceImpl struct{ Repo kb.Repo }

func NewKBService(repo kb.Repo) *KBServiceImpl { return &KBServiceImpl{Repo: repo} }

const (
	errMsgKBUnavailable = "kb backend unavailable"
	errMsgTitleRequired = "title required"
	errMsgIDRequired    = "id required"
)

// AddDoc inserts a new document.
func (s *KBServiceImpl) AddDoc(ctx context.Context, title string, content string) (*kcommon.KBDoc, error) {
	if title == "" {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: errMsgTitleRequired}
	}
	d := &kb.Doc{ID: uuid.NewString(), Title: title, Content: content}
	if err := s.Repo.Add(ctx, d); err != nil {
		return nil, &kcommon.ServiceError{Code: "kb_unavailable", Message: errMsgKBUnavailable}
	}
	observability.KBDocCreated.Add(1)
	return &kcommon.KBDoc{Id: d.ID, Title: d.Title, Content: d.Content}, nil
}

// UpdateDoc performs partial update (empty strings are ignored when optional absent semantics not visible here).
func (s *KBServiceImpl) UpdateDoc(ctx context.Context, id string, title string, content string) (*kcommon.KBDoc, error) {
	if id == "" {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: errMsgIDRequired}
	}
	d, ok := s.Repo.Get(ctx, id)
	if !ok {
		d = &kb.Doc{ID: id}
	}
	if title != "" {
		d.Title = title
	}
	if content != "" {
		d.Content = content
	}
	if d.Title == "" {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: errMsgTitleRequired}
	}
	if err := s.Repo.Update(ctx, d); err != nil {
		return nil, &kcommon.ServiceError{Code: "internal_error", Message: "internal"}
	}
	observability.KBDocUpdated.Add(1)
	return &kcommon.KBDoc{Id: d.ID, Title: d.Title, Content: d.Content}, nil
}

func (s *KBServiceImpl) DeleteDoc(ctx context.Context, id string) error {
	if id == "" {
		return &kcommon.ServiceError{Code: "bad_request", Message: errMsgIDRequired}
	}
	if err := s.Repo.Delete(ctx, id); err != nil {
		return &kcommon.ServiceError{Code: "kb_unavailable", Message: errMsgKBUnavailable}
	}
	observability.KBDocDeleted.Add(1)
	return nil
}

func (s *KBServiceImpl) Search(ctx context.Context, query string, limit int32) ([]*kcommon.SearchItem, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	items, total, err := s.Repo.Search(ctx, query, int(limit))
	if err != nil {
		return nil, &kcommon.ServiceError{Code: "kb_unavailable", Message: errMsgKBUnavailable}
	}
	observability.KBSearchRequests.Add(1)
	observability.KBSearchHits.Add(int64(len(items)))
	out := make([]*kcommon.SearchItem, 0, len(items))
	for _, it := range items {
		out = append(out, &kcommon.SearchItem{Id: it.ID, Title: it.Title, Score: it.Score, Snippet: it.Snippet})
	}
	_ = strconv.Itoa(total) // suppress unused (future extension: return total via separate RPC if needed)
	return out, nil
}

func (s *KBServiceImpl) Info(ctx context.Context) (map[string]string, error) {
	// memory backend only for now
	return map[string]string{"backend": "memory"}, nil
}
