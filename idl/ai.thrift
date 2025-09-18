namespace go ai
include "common.thrift"

service AIService {
  common.EmbeddingResponse Embeddings(1: common.EmbeddingRequest req) throws (1: common.ServiceError err)
}
