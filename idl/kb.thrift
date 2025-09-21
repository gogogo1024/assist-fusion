namespace go kb
include "common.thrift"

struct AddDocRequest {
  1: string title,
  2: string content,
  3: optional map<string,string> tags,
}

struct UpdateDocRequest {
  1: string id,
  2: optional string title,
  3: optional string content,
  4: optional map<string,string> tags,
}

struct DeleteDocRequest { 1: string id }
struct DeleteDocResponse { 1: bool ok }

struct SearchRequest {
  1: string query,
  2: optional i32 limit,       // default 10, cap 50 (server side)
  3: optional i32 offset,      // for incremental pagination
  4: optional bool with_snippet,
}

struct SearchResponse {
  1: list<common.SearchItem> items,
  2: i32 returned,
  3: optional i32 next_offset,
}

struct InfoResponse { 1: map<string,string> stats }

service KBService {
  common.KBDoc AddDoc(1: AddDocRequest req) throws (1: common.ServiceError err)
  common.KBDoc UpdateDoc(1: UpdateDocRequest req) throws (1: common.ServiceError err)
  DeleteDocResponse DeleteDoc(1: DeleteDocRequest req) throws (1: common.ServiceError err)
  SearchResponse Search(1: SearchRequest req) throws (1: common.ServiceError err)
  InfoResponse Info() throws (1: common.ServiceError err)
}
