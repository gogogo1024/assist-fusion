namespace go kb
include "common.thrift"

service KBService {
  common.KBDoc AddDoc(1:string title, 2:string content) throws (1: common.ServiceError err)
  common.KBDoc UpdateDoc(1:string id, 2:optional string title, 3:optional string content) throws (1: common.ServiceError err)
  void DeleteDoc(1:string id) throws (1: common.ServiceError err)
  list<common.SearchItem> Search(1:string query, 2:i32 limit) throws (1: common.ServiceError err)
  map<string,string> Info() throws (1: common.ServiceError err)
}
