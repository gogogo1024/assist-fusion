namespace go ticket
include "common.thrift"

service TicketService {
  common.Ticket CreateTicket(1:string title, 2:string desc, 3:optional string note) throws (1: common.ServiceError err)
  common.Ticket GetTicket(1:string id) throws (1: common.ServiceError err)
  list<common.Ticket> ListTickets() throws (1: common.ServiceError err)
  common.Ticket Assign(1:string id, 2:optional string note) throws (1: common.ServiceError err)
  common.Ticket Resolve(1:string id, 2:optional string note) throws (1: common.ServiceError err)
  common.Ticket Escalate(1:string id, 2:optional string note) throws (1: common.ServiceError err)
  common.Ticket Reopen(1:string id, 2:optional string note) throws (1: common.ServiceError err)
  list<common.TicketCycle> GetCycles(1:string id) throws (1: common.ServiceError err)
  list<common.TicketEvent> GetEvents(1:string id) throws (1: common.ServiceError err)
}
