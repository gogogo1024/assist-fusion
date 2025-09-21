namespace go ai
include "common.thrift"

struct ChatMessage {
  1: string role,    // system|user|assistant
  2: string content,
}

struct ChatRequest {
  1: list<ChatMessage> messages,
  2: optional string model,
  3: optional i32 max_tokens,
}

struct ChatResponse {
  1: ChatMessage message,
  2: optional common.EmbeddingUsage usage,
}

service AIService {
  /**
   * Embeddings
   * - If req.dim <= 0 server picks default (e.g. 128)
   * - Enforce max batch size in server impl
   */
  common.EmbeddingResponse Embeddings(1: common.EmbeddingRequest req) throws (1: common.ServiceError err)

  /**
   * Simple chat style completion (optional to implement initially)
   */
  ChatResponse Chat(1: ChatRequest req) throws (1: common.ServiceError err)
}
