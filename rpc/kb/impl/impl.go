package impl

import (
	"context"

	"github.com/gogogo1024/assist-fusion/internal/kb"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	kbidl "github.com/gogogo1024/assist-fusion/kitex_gen/kb"
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

// AddDoc implements new request struct contract.
func (s *KBServiceImpl) AddDoc(ctx context.Context, req *kbidl.AddDocRequest) (*kcommon.KBDoc, error) {
	if req == nil || req.Title == "" {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: errMsgTitleRequired}
	}
	d := &kb.Doc{ID: uuid.NewString(), Title: req.Title, Content: req.Content}
	if err := s.Repo.Add(ctx, d); err != nil {
		return nil, &kcommon.ServiceError{Code: "kb_unavailable", Message: errMsgKBUnavailable}
	}
	observability.KBDocCreated.Add(1)
	return &kcommon.KBDoc{Id: d.ID, Title: d.Title, Content: d.Content}, nil
}

// UpdateDoc performs partial update (empty strings are ignored when optional absent semantics not visible here).
func (s *KBServiceImpl) UpdateDoc(ctx context.Context, req *kbidl.UpdateDocRequest) (*kcommon.KBDoc, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: errMsgIDRequired}
	}
	d, ok := s.Repo.Get(ctx, req.Id)
	if !ok {
		d = &kb.Doc{ID: req.Id}
	}
	if req.Title != nil && *req.Title != "" {
		d.Title = *req.Title
	}
	if req.Content != nil && *req.Content != "" {
		d.Content = *req.Content
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

func (s *KBServiceImpl) DeleteDoc(ctx context.Context, req *kbidl.DeleteDocRequest) (*kbidl.DeleteDocResponse, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: errMsgIDRequired}
	}
	if err := s.Repo.Delete(ctx, req.Id); err != nil {
		return nil, &kcommon.ServiceError{Code: "kb_unavailable", Message: errMsgKBUnavailable}
	}
	observability.KBDocDeleted.Add(1)
	return &kbidl.DeleteDocResponse{Ok: true}, nil
}

func (s *KBServiceImpl) Search(ctx context.Context, req *kbidl.SearchRequest) (*kbidl.SearchResponse, error) {
	if req == nil {
		return nil, &kcommon.ServiceError{Code: "bad_request", Message: "request required"}
	}
	limit := int32(10)
	if req.Limit != nil {
		limit = *req.Limit
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	items, total, err := s.Repo.Search(ctx, req.Query, int(limit))
	if err != nil {
		return nil, &kcommon.ServiceError{Code: "kb_unavailable", Message: errMsgKBUnavailable}
	}
	observability.KBSearchRequests.Add(1)
	observability.KBSearchHits.Add(int64(len(items)))
	out := make([]*kcommon.SearchItem, 0, len(items))
	for _, it := range items {
		out = append(out, &kcommon.SearchItem{Id: it.ID, Title: it.Title, Score: it.Score, Snippet: it.Snippet})
	}
	var next *int32
	if req.Offset != nil {
		off := *req.Offset
		consumed := off + int32(len(out))
		if int(consumed) < total {
			n := consumed
			next = &n
		}
	}
	total32 := int32(total)
	return &kbidl.SearchResponse{Items: out, Returned: int32(len(out)), NextOffset: next, Total: &total32}, nil
}

func (s *KBServiceImpl) Info(ctx context.Context) (*kbidl.InfoResponse, error) {
	return &kbidl.InfoResponse{Stats: map[string]string{"backend": "memory"}}, nil
}
