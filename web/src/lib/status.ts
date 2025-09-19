// Unified status/variant helpers
// Central place to map backend status strings or stat keys into visual variants & colors.

export type Variant = 'default' | 'success' | 'warning' | 'danger' | 'neutral' | 'info'

// Map ticket runtime status to variant used by Tags / bars.
export function statusToVariant(status: string | undefined | null): Variant {
  const v = String(status || '').toLowerCase()
  switch (v) {
    case 'resolved': return 'success'
    case 'escalated': return 'warning'
    case 'closed': return 'neutral'
    case 'canceled': return 'danger'
    case 'waiting':
    case 'in_progress': return 'info'
    default: return 'default'
  }
}

// Map statistic key (aggregated counters) to variant.
export function statKeyToVariant(key: string): Variant {
  switch (key) {
    case 'resolved': return 'success'
    case 'escalated': return 'warning'
    case 'waiting': return 'info'
    case 'closed': return 'neutral'
    case 'canceled': return 'danger'
    case 'in_progress': return 'info'
    case 'assigned': return 'info'
    case 'created': return 'info'
    default: return 'default'
  }
}

// Convert variant to CSS variable color for Tag component (foreground token).
export function variantToTagColor(v: Variant): string | undefined {
  switch (v) {
    case 'success': return 'var(--color-success)'
    case 'warning': return 'var(--color-warning)'
    case 'danger': return 'var(--color-danger)'
    case 'info': return 'var(--color-info)'
    case 'neutral': return 'var(--color-neutral)'
    default: return undefined
  }
}

// Bar (top mini bar) background class mapping for statistic cards.
export function variantBarClass(v: Variant): string {
  switch (v) {
    case 'success': return 'bg-emerald-500'
    case 'warning': return 'bg-amber-500'
    case 'danger': return 'bg-rose-500'
    case 'info': return 'bg-sky-500'
    case 'neutral': return 'bg-muted'
    default: return 'bg-primary'
  }
}

// Left bar class for table rows
export function variantLeftBarClass(v: Variant): string {
  switch (v) {
    case 'success': return 'bar-success'
    case 'warning': return 'bar-warning'
    case 'danger': return 'bar-danger'
    case 'info': return 'bar-info'
    default: return 'bar-neutral'
  }
}

// Tint (background highlight) class when row is active
export function variantTintClass(v: Variant): string {
  switch (v) {
    case 'success': return 'tint-success'
    case 'warning': return 'tint-warning'
    case 'danger': return 'tint-danger'
    case 'info': return 'tint-info'
    default: return 'tint-neutral'
  }
}
