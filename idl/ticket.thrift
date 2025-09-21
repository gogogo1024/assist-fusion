namespace go ticket
include "common.thrift"

struct TicketResponse { 1: common.Ticket ticket }

struct CreateTicketRequest {
  1: string title,
  2: string desc,
  3: optional string note,
}

struct GetTicketRequest { 1: string id }

struct ListTicketsRequest {
  1: optional common.Pagination pagination,
  2: optional list<common.TicketStatus> statuses,
  3: optional i64 created_from,
  4: optional i64 created_to,
}

struct ListTicketsResponse {
  1: list<common.Ticket> tickets,
  2: optional common.PageInfo page_info,
}

struct TicketActionRequest {
  1: string id,
  2: optional string note,
}

struct GetCyclesRequest { 1: string id }
struct GetEventsRequest { 1: string id }

service TicketService {
  TicketResponse CreateTicket(1: CreateTicketRequest req) throws (1: common.ServiceError err)
  TicketResponse GetTicket(1: GetTicketRequest req) throws (1: common.ServiceError err)
  ListTicketsResponse ListTickets(1: ListTicketsRequest req) throws (1: common.ServiceError err)

  TicketResponse Assign(1: TicketActionRequest req) throws (1: common.ServiceError err)
  TicketResponse Resolve(1: TicketActionRequest req) throws (1: common.ServiceError err)
  TicketResponse Escalate(1: TicketActionRequest req) throws (1: common.ServiceError err)
  TicketResponse Reopen(1: TicketActionRequest req) throws (1: common.ServiceError err)

  list<common.TicketCycle> GetCycles(1: GetCyclesRequest req) throws (1: common.ServiceError err)
  list<common.TicketEvent> GetEvents(1: GetEventsRequest req) throws (1: common.ServiceError err)
}
