import { createMachine } from 'xstate'

// 业务状态
export type TicketStatus = 'created'|'assigned'|'in_progress'|'waiting'|'escalated'|'resolved'|'closed'|'canceled'

// 事件定义
type Ev =
  | { type: 'ASSIGN' }
  | { type: 'START' }
  | { type: 'WAIT' }
  | { type: 'ESCALATE' }
  | { type: 'RESOLVE' }
  | { type: 'REOPEN' }
  | { type: 'CLOSE' }
  | { type: 'CANCEL' }

// 终态集合（禁止某些动作，如 ESCALATE、START、WAIT）
const terminal = new Set<TicketStatus>(['resolved','closed','canceled'])

export const ticketMachine = createMachine({
  id: 'ticket',
  initial: 'created',
  states: {
    created: {
      on: { ASSIGN: 'assigned', START: 'in_progress', WAIT: 'waiting', ESCALATE: 'escalated', CLOSE: 'closed', CANCEL: 'canceled' }
    },
    assigned: {
      on: { START: 'in_progress', WAIT: 'waiting', ESCALATE: 'escalated', RESOLVE: 'resolved', CLOSE: 'closed', CANCEL: 'canceled' }
    },
    in_progress: {
      on: { WAIT: 'waiting', RESOLVE: 'resolved', ESCALATE: 'escalated', CLOSE: 'closed', CANCEL: 'canceled' }
    },
    waiting: {
      on: { START: 'in_progress', RESOLVE: 'resolved', ESCALATE: 'escalated', CLOSE: 'closed', CANCEL: 'canceled' }
    },
    escalated: {
      on: { RESOLVE: 'resolved', CLOSE: 'closed', CANCEL: 'canceled' }
    },
    resolved: {
      on: { REOPEN: 'created' }
    },
    closed: {},
    canceled: {},
  }
})

// 前端校验：基于当前状态与目标动作，判断是否允许
export function canTransition(status: TicketStatus, action: 'assign'|'start'|'wait'|'escalate'|'resolve'|'reopen'|'close'|'cancel'){
  const cfg: any = (ticketMachine as any).config || {}
  const states: any = cfg.states || {}
  const state: any = states[status] || {}
  const on = state.on || {}
  const map: Record<string,string> = {
    assign: 'ASSIGN', start: 'START', wait: 'WAIT', escalate: 'ESCALATE', resolve: 'RESOLVE', reopen: 'REOPEN', close: 'CLOSE', cancel: 'CANCEL'
  }
  // 特殊后端规则：终态禁止 escalate、start、wait（返回409）；close/cancel 对终态重复返回409
  if (terminal.has(status)){
    if (action==='escalate' || action==='start' || action==='wait') return false
  }
  return !!on?.[map[action]]
}
