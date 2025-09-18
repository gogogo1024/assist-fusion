import * as React from 'react'
import { cn } from '../../lib/utils'

type BadgeVariant = 'default' | 'info' | 'success' | 'warning' | 'danger' | 'neutral'
type BadgeSize = 'sm' | 'md'

export interface BadgeProps extends Readonly<React.HTMLAttributes<HTMLSpanElement>> {
  readonly variant?: BadgeVariant
  readonly size?: BadgeSize
}

const variantClass: Record<BadgeVariant, string> = {
  default: 'badge',
  info: 'badge bg-sky-500/20 text-sky-300 border-sky-700',
  success: 'badge bg-emerald-500/20 text-emerald-300 border-emerald-700',
  warning: 'badge bg-amber-500/20 text-amber-200 border-amber-700',
  danger: 'badge bg-rose-500/20 text-rose-300 border-rose-700',
  neutral: 'badge bg-muted text-muted-foreground border-border',
}

const sizeClass: Record<BadgeSize, string> = {
  sm: 'px-2 py-0.5 text-[11px]',
  md: 'px-2 py-0.5 text-xs',
}

export function Badge({ className, variant = 'default', size = 'md', ...props }: BadgeProps) {
  return <span className={cn(variantClass[variant], sizeClass[size], className)} {...props} />
}
