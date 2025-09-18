const base = ''

// 轻量Toast
function toast(msg, ok = true) {
  const el = document.getElementById('toast')
  if (!el) return alert(msg)
  el.textContent = msg
  el.style.background = ok ? '#134e4a' : '#7f1d1d'
  el.hidden = false
  setTimeout(() => { el.hidden = true }, 2000)
}

function show(el, data) {
  el.textContent = JSON.stringify(data, null, 2)
}

function fmt(ts) {
  if (!ts) return '-'
  try { return new Date(ts * 1000).toLocaleString() } catch { return String(ts) }
}

function badge(status) {
  const s = String(status || '').toLowerCase()
  let color = '#a6b0cf'
  if (s === 'resolved') color = '#22c55e'
  else if (s === 'escalated') color = '#f97316'
  else if (s === 'assigned') color = '#3b82f6'
  let label = s
  if (window.t) {
    // try status keys first, then event keys
    const viaStats = t(`stats.${s}`)
    if (viaStats && !String(viaStats).startsWith('stats.')) {
      label = viaStats
    } else {
      const viaEvents = t(`events.${s}`)
      label = (viaEvents && !String(viaEvents).startsWith('events.')) ? viaEvents : s
    }
  }
  return `<span class="badge" style="background:${color}">${label || '-'}</span>`
}

async function req(method, path, body) {
  const res = await fetch(base + path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let data
  try { data = JSON.parse(text) } catch { data = text }
  return { status: res.status, ok: res.ok, data, headers: Object.fromEntries(res.headers.entries()) }
}

function byId(id) { return document.getElementById(id) }

// 诊断
byId('btnReady').addEventListener('click', async () => {
  const out = byId('readyOut')
  const r = await req('GET', '/ready')
  show(out, r)
})

// 工单列表
let allTickets = []
let page = 1
const pageSize = 10

function applyFilter(data) {
  const status = byId('fltStatus').value
  const kw = (byId('fltKeyword').value || '').toLowerCase().trim()
  return data.filter(t => {
    const okStatus = !status || String(t.status) === status
    const okKW = !kw || (String(t.title||'').toLowerCase().includes(kw) || String(t.desc||'').toLowerCase().includes(kw))
    return okStatus && okKW
  }).sort((a,b) => (b.created_at||0) - (a.created_at||0))
}

function renderPaged(data) {
  const total = data.length
  const pages = Math.max(1, Math.ceil(total / pageSize))
  if (page > pages) page = pages
  const start = (page-1)*pageSize
  const items = data.slice(start, start+pageSize)
  const wrap = byId('ticketsList')
  const rows = items.map(t => `
    <tr>
      <td><code>${t.id}</code></td>
      <td>${t.title || '-'}</td>
      <td>${badge(t.status)}</td>
      <td>${fmt(t.created_at)}</td>
  <td><button class="btn-small" data-id="${t.id}">${window.t ? t('actions.view') : '查看'}</button></td>
    </tr>`).join('')
  const th = (k, fb) => window.t ? t(k) : fb
  wrap.innerHTML = `<table class="table">
    <thead><tr><th>${th('fields.id','ID')}</th><th>${th('fields.title','标题')}</th><th>${th('fields.status','状态')}</th><th>${th('fields.createdAt','创建时间')}</th><th>${th('columns.operations','操作')}</th></tr></thead>
    <tbody>${rows || `<tr><td colspan="5">${th('common.noData','暂无数据')}</td></tr>`}</tbody>
  </table>`
  wrap.querySelectorAll('button[data-id]')
    .forEach(btn => btn.addEventListener('click', () => { byId('tId').value = btn.dataset.id; byId('btnGet').click() }))
  byId('pageInfo').textContent = window.t ? t('common.pageInfo', { page, pages, total }) : `第 ${page} / ${pages} 页（共 ${total} 条）`
  byId('prevPage').disabled = page <= 1
  byId('nextPage').disabled = page >= pages
}

async function renderList() {
  const wrap = byId('ticketsList')
  wrap.innerHTML = window.t ? t('common.loading') : '加载中...'
  const r = await req('GET', '/v1/tickets')
  if (!r.ok || !Array.isArray(r.data)) { wrap.textContent = window.t ? t('common.loadFailed') : '加载失败'; return }
  allTickets = r.data
  renderPaged(applyFilter(allTickets))
}
byId('btnList').addEventListener('click', renderList)
renderList()

byId('btnApplyFilter').addEventListener('click', () => { page=1; renderPaged(applyFilter(allTickets)) })
byId('btnResetFilter').addEventListener('click', () => { byId('fltStatus').value=''; byId('fltKeyword').value=''; page=1; renderPaged(applyFilter(allTickets)) })
byId('prevPage').addEventListener('click', () => { if (page>1){ page--; renderPaged(applyFilter(allTickets)) } })
byId('nextPage').addEventListener('click', () => { page++; renderPaged(applyFilter(allTickets)) })

// 工单
byId('btnCreate').addEventListener('click', async () => {
  const title = byId('tTitle').value || '演示工单'
  const desc = byId('tDesc').value || '这是一个示例描述'
  const r = await req('POST', '/v1/tickets', { title, desc })
  show(byId('ticketOut'), r)
  if (r?.ok && r?.data?.id) byId('tId').value = r.data.id
  renderList()
  let msg
  if (r.ok) {
    msg = window.t ? t('common.success') : '成功'
  } else {
    msg = window.t ? t('common.failed') : '失败'
  }
  toast(msg, r.ok)
})

async function action(act) {
  const id = byId('tId').value.trim()
  if (!id) { alert(window.t ? t('common.pleaseTicketId') : '请先填入工单ID'); return }
  const note = byId('tNote').value || ''
  const btn = document.getElementById(`btn${act.charAt(0).toUpperCase()+act.slice(1)}`)
  if (btn) { btn.disabled = true; btn.insertAdjacentHTML('beforeend', '<span class="spinner"></span>') }
  try {
    const r = await req('PUT', `/v1/tickets/${id}/${act}`, { note })
    show(byId('ticketOut'), r)
    renderList()
    const actLabel = window.t ? t('actions.' + act) : act
    const okMsg = window.t ? actLabel + ' ' + t('common.success') : act + ' 成功'
    const failMsg = window.t ? actLabel + ' ' + t('common.failed') + '：' + r.status : act + ' 失败：' + r.status
    toast(r.ok ? okMsg : failMsg, r.ok)
  } catch (e) {
    const actLabel = window.t ? t('actions.' + act) : act
    const errText = (e && e.message) ? e.message : e
    const msg2 = window.t ? actLabel + ' ' + t('common.failed') + '：' + errText : act + ' 失败：' + errText
    toast(msg2, false)
  } finally {
    if (btn) { btn.disabled = false; const sp = btn.querySelector('.spinner'); if (sp) sp.remove() }
  }
}
byId('btnAssign').addEventListener('click', () => action('assign'))
byId('btnEscalate').addEventListener('click', () => action('escalate'))
byId('btnStart').addEventListener('click', () => action('start'))
byId('btnWait').addEventListener('click', () => action('wait'))
byId('btnResolve').addEventListener('click', () => action('resolve'))
byId('btnClose').addEventListener('click', () => action('close'))
byId('btnCancel').addEventListener('click', () => action('cancel'))
byId('btnReopen').addEventListener('click', () => action('reopen'))
byId('btnGet').addEventListener('click', async () => {
  const id = byId('tId').value.trim()
  if (!id) { alert(window.t ? t('common.pleaseTicketId') : '请先填入工单ID'); return }
  const r = await req('GET', `/v1/tickets/${id}`)
  show(byId('ticketOut'), r)
})
byId('btnCycles').addEventListener('click', async () => {
  const id = byId('tId').value.trim()
  if (!id) { alert(window.t ? t('common.pleaseTicketId') : '请先填入工单ID'); return }
  const r = await req('GET', `/v1/tickets/${id}/cycles`)
  show(byId('extraOut'), r)
})
byId('btnEvents').addEventListener('click', async () => {
  const id = byId('tId').value.trim()
  if (!id) { alert(window.t ? t('common.pleaseTicketId') : '请先填入工单ID'); return }
  const r = await req('GET', `/v1/tickets/${id}/events`)
  // 渲染时间线
  const tl = byId('eventsTL')
  tl.innerHTML = ''
  const items = Array.isArray(r?.data?.events) ? r.data.events : []
  if (!items.length) { tl.textContent = window.t ? t('common.noData') : '暂无数据'; return }
  tl.innerHTML = items.map(ev => `
    <div class="tl-item">
      <div class="tl-dot"></div>
      <div class="tl-content">
        <div class="tl-head">${badge(ev.type)} <span class="ts">${fmt(ev.at)}</span></div>
        ${ev.note ? `<div class="tl-note">${ev.note}</div>` : ''}
      </div>
    </div>`).join('')
  toast(window.t ? t('common.loadedEvents') : '已加载事件')
})

// 知识库
byId('btnAddDoc').addEventListener('click', async () => {
  const title = byId('kbTitle').value || 'FAQ'
  const content = byId('kbContent').value || '示例内容'
  const r = await req('POST', '/v1/docs', { title, content })
  show(byId('kbOut'), r)
  if (r?.ok && r?.data?.id) byId('kbId').value = r.data.id
})
byId('btnUpdateDoc').addEventListener('click', async () => {
  const id = byId('kbId').value.trim()
  if (!id) { alert(window.t ? t('common.pleaseDocId') : '请先填入文档ID'); return }
  const title = byId('kbTitle2').value || undefined
  const content = byId('kbContent2').value || undefined
  const r = await req('PUT', `/v1/docs/${id}`, { title, content })
  show(byId('kbOut'), r)
})
byId('btnDeleteDoc').addEventListener('click', async () => {
  const id = byId('kbId').value.trim()
  if (!id) { alert(window.t ? t('common.pleaseDocId') : '请先填入文档ID'); return }
  const r = await req('DELETE', `/v1/docs/${id}`)
  show(byId('kbOut'), r)
})
byId('btnSearch').addEventListener('click', async () => {
  const q = byId('kbQ').value || ''
  const r = await req('GET', `/v1/search?q=${encodeURIComponent(q)}&limit=10`)
  show(byId('kbOut'), r)
})
byId('btnKBInfo').addEventListener('click', async () => {
  const r = await req('GET', '/v1/kb/info')
  show(byId('kbOut'), r)
})

// 域指标输出
byId('btnDomainMetrics').addEventListener('click', async () => {
  const res = await fetch('/metrics/domain')
  const text = await res.text()
  byId('metricsOut').textContent = text
})

// 主管看板逻辑
let supTimer = null
function supStopTimer(){ if (supTimer){ clearInterval(supTimer); supTimer=null } }
function supStartTimer(){
  const auto = byId('supAuto')?.checked
  supStopTimer()
  if (!auto) return
  const iv = parseInt(byId('supInterval')?.value||'10', 10) * 1000
  supTimer = setInterval(supRefresh, iv)
}

function groupStats(list){
  const stat = { total: list.length }
  for (const t of list){
    const s = String(t.status||'').toLowerCase()
    stat[s] = (stat[s]||0)+1
  }
  return stat
}

function renderStats(s){
  const el = byId('supStats')
  if (!el) return
  const keys = ['total','created','assigned','in_progress','waiting','escalated','resolved','closed','canceled']
  el.innerHTML = keys.map(k => `
    <div class="stat">
      <div class="label">${window.t ? t('stats.'+k) : k}</div>
      <div class="value">${s[k]||0}</div>
    </div>`).join('')
}

function renderUnassigned(list){
  const wrap = byId('supUnassigned')
  if (!wrap) return
  const rows = list.map(t => `
    <tr>
      <td><code>${t.id}</code></td>
      <td>${t.title||'-'}</td>
      <td>${badge(t.status)}</td>
      <td>${fmt(t.created_at)}</td>
      <td>
        <input data-id="${t.id}" class="assignee" placeholder="${window.t ? t('fields.assignee') : '指派人'}" style="min-width:100px"/>
        <button class="btn-small assign" data-id="${t.id}">${window.t ? t('actions.assign') : '指派'}</button>
      </td>
    </tr>`).join('')
  const th = (k, fb) => window.t ? t(k) : fb
  wrap.innerHTML = `<table class="table">
    <thead><tr><th>${th('fields.id','ID')}</th><th>${th('fields.title','标题')}</th><th>${th('fields.status','状态')}</th><th>${th('fields.createdAt','创建时间')}</th><th>${th('columns.operations','操作')}</th></tr></thead>
    <tbody>${rows || `<tr><td colspan="5">${th('common.none','暂无')}</td></tr>`}</tbody>
  </table>`
  wrap.querySelectorAll('button.assign').forEach(btn => btn.addEventListener('click', async () => {
    const id = btn.dataset.id
    const input = wrap.querySelector(`input.assignee[data-id="${id}"]`)
    const assignee = input?.value?.trim()
    const r = await req('PUT', `/v1/tickets/${id}/assign`, { assignee })
    toast(r.ok ? '指派成功' : `指派失败：${r.status}`, r.ok)
    supRefresh()
  }))
}

function renderOverdue(list){
  const wrap = byId('supOverdue')
  if (!wrap) return
  const rows = list.map(t => `
    <tr>
      <td><code>${t.id}</code></td>
      <td>${t.title||'-'}</td>
      <td>${badge(t.status)}</td>
      <td>${fmt(t.due_at)}</td>
      <td>${fmt(t.created_at)}</td>
    </tr>`).join('')
  const th2 = (k, fb) => window.t ? t(k) : fb
  wrap.innerHTML = `<table class="table">
    <thead><tr><th>${th2('fields.id','ID')}</th><th>${th2('fields.title','标题')}</th><th>${th2('fields.status','状态')}</th><th>${th2('fields.dueAt','截止')}</th><th>${th2('fields.createdAt','创建')}</th></tr></thead>
    <tbody>${rows || `<tr><td colspan="5">${th2('common.none','暂无')}</td></tr>`}</tbody>
  </table>`
}

async function supRefresh(){
  const btn = byId('supRefresh'); if (btn){ btn.disabled = true; btn.insertAdjacentHTML('beforeend', '<span class="spinner"></span>') }
  try{
    const r = await req('GET', '/v1/tickets')
    const list = Array.isArray(r.data)? r.data : []
    const stats = groupStats(list)
    renderStats(stats)
    const terminal = new Set(['resolved','closed','canceled'])
    const unassigned = list.filter(t => !t.assignee)
    const now = Math.floor(Date.now()/1000)
    const overdue = list.filter(t => t.due_at && t.due_at < now && !terminal.has(String(t.status||'').toLowerCase()))
    renderUnassigned(unassigned.slice(0,50))
    renderOverdue(overdue.slice(0,50))
  } finally {
    if (btn){ btn.disabled=false; const sp=btn.querySelector('.spinner'); if (sp) sp.remove() }
    supStartTimer()
  }
}

byId('supRefresh')?.addEventListener('click', supRefresh)
byId('supAuto')?.addEventListener('change', supStartTimer)
byId('supInterval')?.addEventListener('change', supStartTimer)

// 视图切换：默认显示坐席
function switchTab(hash) {
  const sections = ['agent', 'supervisor', 'kb', 'diag']
  sections.forEach(id => {
    const sec = document.getElementById(id)
    if (sec) sec.style.display = (hash === `#${id}` || (!hash && id==='agent')) ? 'block' : 'none'
  })
  // 激活态
  const tabs = [
    {id:'tabAgent', h:'#agent'},
    {id:'tabKB', h:'#kb'},
    {id:'tabSupervisor', h:'#supervisor'},
    {id:'tabDiag', h:'#diag'},
  ]
  tabs.forEach(t => {
    const el = document.getElementById(t.id)
    if (!el) return
    const active = (hash || '#agent') === t.h
    el.classList.toggle('active', active)
  })
  if ((hash || '#agent') === '#supervisor') {
    supRefresh()
  } else {
    supStopTimer()
  }
}
window.addEventListener('hashchange', () => switchTab(location.hash))
switchTab(location.hash)

// 坐席KB建议：默认用当前工单标题+描述
async function searchAgentKB() {
  const id = byId('tId').value.trim()
  let q = byId('kbQAgent').value || ''
  if (!q && id) {
    const r = await req('GET', `/v1/tickets/${id}`)
    const t = r?.data || {}
    q = `${t.title || ''} ${t.desc || ''}`.trim()
    byId('kbQAgent').value = q
  }
  if (!q) { toast(window.t ? t('common.pleaseKeyword') : '请输入关键字'); return }
  const r2 = await req('GET', `/v1/search?q=${encodeURIComponent(q)}&limit=5`)
  show(byId('kbOutAgent'), r2)
    let msg
    if (r2.ok) {
      msg = window.t ? t('common.loadedSuggestions') : '已加载建议'
    } else {
      msg = window.t ? t('common.loadSuggestionsFailed') : '建议加载失败'
    }
    toast(msg, r2.ok)
}
byId('btnSearchAgent').addEventListener('click', searchAgentKB)

// 引导条处理
const guideKey = 'agentGuideHidden'
if (localStorage.getItem(guideKey) === '1') {
  const g = byId('agentGuide'); if (g) g.style.display = 'none'
}
byId('btnCloseGuide')?.addEventListener('click', () => {
  localStorage.setItem(guideKey, '1')
  const g = byId('agentGuide'); if (g) g.style.display = 'none'
})
