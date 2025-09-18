/* global i18next */
;(function(){
  // Resources
  const resources = {
    'zh-CN': {
      translation: {
        nav: { agent: '坐席', supervisor: '主管', kb: '知识库', diag: '诊断' },
        actions: {
          refresh: '刷新', assign: '指派', start: '开始', wait: '等待', escalate: '升级', resolve: '解决', close: '关闭', cancel: '取消', reopen: '重开',
          create: '创建', add: '新增', update: '更新', delete: '删除', search: '搜索', backendInfo: '后端信息',
          checkReady: '检查 /ready', viewMetrics: '查看 /metrics/domain', view: '查看'
        },
        fields: {
          id: 'ID', title: '标题', desc: '描述', status: '状态', assignee: '指派人', note: '备注（可选）', ticketId: '工单ID', docId: '文档ID',
          content: '内容', query: '搜索关键字', dueAt: '截止', createdAt: '创建时间'
        },
        placeholders: {
          title: '标题',
          desc: '描述',
          keyword: '标题/描述 关键词',
          note: '备注（可选）',
          assignee: '指派人',
          kbTitle: '标题',
          kbContent: '内容',
          kbId: '文档ID',
          kbTitleNew: '新标题（可选）',
          kbContentNew: '新内容（可选）',
          kbQuery: '搜索关键字',
        },
        columns: { operations: '操作' },
        supervisor: { unassigned: '未分配工单', overdue: '逾期工单（非终态）' },
        stats: {
          total: '总数', created: '已创建', assigned: '已指派', in_progress: '进行中', waiting: '等待中', escalated: '已升级',
          resolved: '已解决', closed: '已关闭', canceled: '已取消'
        },
        common: {
          loading: '加载中...', loadFailed: '加载失败', noData: '暂无数据', none: '暂无', all: '全部',
          pageInfo: '第 {{page}} / {{pages}} 页（共 {{total}} 条）',
          success: '成功', failed: '失败',
          pleaseTicketId: '请先填写工单ID',
          pleaseDocId: '请先填写文档ID',
          pleaseKeyword: '请输入关键字',
          loadedEvents: '已加载事件',
          loadedSuggestions: '已加载建议',
          loadSuggestionsFailed: '建议加载失败'
        },
        events: { created: '已创建', assigned: '已指派', escalated: '已升级', resolved: '已解决', reopened: '已重开', waiting: '等待中', started: '已开始', closed: '已关闭', canceled: '已取消' }
      }
    },
    en: {
      translation: {
        nav: { agent: 'Agent', supervisor: 'Supervisor', kb: 'Knowledge Base', diag: 'Diagnostics' },
        actions: {
          refresh: 'Refresh', assign: 'Assign', start: 'Start', wait: 'Wait', escalate: 'Escalate', resolve: 'Resolve', close: 'Close', cancel: 'Cancel', reopen: 'Reopen',
          create: 'Create', add: 'Add', update: 'Update', delete: 'Delete', search: 'Search', backendInfo: 'Backend Info',
          checkReady: 'Check /ready', viewMetrics: 'View /metrics/domain', view: 'View'
        },
        fields: {
          id: 'ID', title: 'Title', desc: 'Description', status: 'Status', assignee: 'Assignee', note: 'Note (optional)', ticketId: 'Ticket ID', docId: 'Doc ID',
          content: 'Content', query: 'Search keyword', dueAt: 'Due', createdAt: 'Created At'
        },
        placeholders: {
          title: 'Title',
          desc: 'Description',
          keyword: 'Title/Description keywords',
          note: 'Note (optional)',
          assignee: 'Assignee',
          kbTitle: 'Title',
          kbContent: 'Content',
          kbId: 'Doc ID',
          kbTitleNew: 'New title (optional)',
          kbContentNew: 'New content (optional)',
          kbQuery: 'Search keyword'
        },
        columns: { operations: 'Operations' },
        supervisor: { unassigned: 'Unassigned', overdue: 'Overdue (non-terminal)' },
        stats: {
          total: 'Total', created: 'Created', assigned: 'Assigned', in_progress: 'In Progress', waiting: 'Waiting', escalated: 'Escalated',
          resolved: 'Resolved', closed: 'Closed', canceled: 'Canceled'
        },
        common: {
          loading: 'Loading...', loadFailed: 'Load failed', noData: 'No data', none: 'None', all: 'All',
          pageInfo: 'Page {{page}} / {{pages}} ({{total}} items)',
          success: 'succeeded', failed: 'failed',
          pleaseTicketId: 'Please enter Ticket ID first',
          pleaseDocId: 'Please enter Document ID first',
          pleaseKeyword: 'Please enter a keyword',
          loadedEvents: 'Events loaded',
          loadedSuggestions: 'Suggestions loaded',
          loadSuggestionsFailed: 'Failed to load suggestions'
        },
        events: { created: 'Created', assigned: 'Assigned', escalated: 'Escalated', resolved: 'Resolved', reopened: 'Reopened', waiting: 'Waiting', started: 'Started', closed: 'Closed', canceled: 'Canceled' }
      }
    }
  }

  // Initialize
  i18next
    .use(window.i18nextBrowserLanguageDetector)
    .init({
      resources,
      fallbackLng: 'zh-CN',
      supportedLngs: ['zh-CN','en','zh'],
      detection: { order: ['querystring','localStorage','navigator'], caches: ['localStorage'], lookupQuerystring: 'lang' },
      interpolation: { escapeValue: false },
    }, () => {
      translateStatic()
      translateFilterOptions()
      const sel = document.getElementById('langSel')
      if (sel) sel.value = i18next.language
      document.documentElement.setAttribute('lang', i18next.language || 'zh-CN')
    })

  // global t
  window.t = (k, opts) => i18next.t(k, opts)

  window.translateStatic = function translateStatic(){
    const setText = (id, key) => { const el = document.getElementById(id); if (el) el.textContent = i18next.t(key) }
    setText('tabAgent', 'nav.agent')
    setText('tabSupervisor', 'nav.supervisor')
    setText('tabKB', 'nav.kb')
    setText('tabDiag', 'nav.diag')
    setText('btnReady', 'actions.checkReady')
    setText('btnDomainMetrics', 'actions.viewMetrics')
    const h1 = document.querySelector('header h1'); if (h1) h1.textContent = 'AssistFusion'
    // Guide
    const g = document.getElementById('agentGuide')
    if (g){
      const div = g.querySelector('div'); if (div) div.textContent = i18next.t('guide.tip')
      const btn = document.getElementById('btnCloseGuide'); if (btn) btn.textContent = i18next.t('guide.close')
    }
    // Supervisor headings
    const sup = document.getElementById('supervisor')
    if (sup){
      const h2 = sup.querySelector('h2'); if (h2) h2.textContent = i18next.t('nav.supervisor')
    }
    // Placeholders
    const setPh = (id, key) => { const el = document.getElementById(id); if (el) el.placeholder = i18next.t(key) }
    setPh('fltKeyword', 'placeholders.keyword')
    setPh('tTitle', 'placeholders.title')
    setPh('tDesc', 'placeholders.desc')
    setPh('tNote', 'placeholders.note')
    setPh('kbTitle', 'placeholders.kbTitle')
    setPh('kbContent', 'placeholders.kbContent')
    setPh('kbId', 'placeholders.kbId')
    setPh('kbTitle2', 'placeholders.kbTitleNew')
    setPh('kbContent2', 'placeholders.kbContentNew')
    setPh('kbQ', 'placeholders.kbQuery')
    setPh('kbQAgent', 'placeholders.kbQuery')
  }

  function translateFilterOptions(){
    const sel = document.getElementById('fltStatus')
    if (!sel) return
    for (const opt of sel.options){
      const v = opt.value
      if (!v) opt.textContent = i18next.t('common.all')
      else opt.textContent = i18next.t(`stats.${String(v).toLowerCase()}`)
    }
  }

  // Language selector
  window.addEventListener('DOMContentLoaded', () => {
    const sel = document.getElementById('langSel')
    if (sel){
      sel.addEventListener('change', () => {
        i18next.changeLanguage(sel.value).then(() => {
          translateStatic(); translateFilterOptions();
        })
      })
    }
  })
})()
