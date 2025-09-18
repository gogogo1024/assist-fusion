namespace go common

/**
 * Shared error + common structs
 */

exception ServiceError {
  1: string code
  2: string message
}

enum TicketStatus {
  CREATED = 0,
  ASSIGNED = 1,
  ESCALATED = 2,
  RESOLVED = 3,
}

struct TicketCycle {
  1: i64 created_at,
  2: i64 assigned_at,
  3: i64 resolved_at,
  4: i64 escalated_at,
  5: TicketStatus status,
}

struct TicketEvent {
  1: string type,
  2: i64 at,
  3: string note,
}

struct Ticket {
  1: string id,
  2: string title,
  3: string desc,
  4: TicketStatus status,
  5: i64 created_at,
  6: i64 assigned_at,
  7: i64 resolved_at,
  8: i64 escalated_at,
  9: i64 reopened_at,
 10: list<TicketCycle> cycles,
 11: i32 current_cycle,
 12: list<TicketEvent> events,
}

struct KBDoc {
  1: string id,
  2: string title,
  3: string content,
}

struct SearchItem {
  1: string id,
  2: string title,
  3: double score,
  4: string snippet,
}

struct EmbeddingRequest {
  1: list<string> texts,
  2: i32 dim,
}

struct EmbeddingResponse {
  1: list<list<double>> vectors,
  2: i32 dim,
}
