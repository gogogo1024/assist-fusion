import * as React from 'react'
import { Tag } from '@arco-design/web-react'
import { cn } from '../../lib/utils'

type BadgeVariant = 'default' | 'info' | 'success' | 'warning' | 'danger' | 'neutral'
type BadgeSize = 'sm' | 'md'

export interface BadgeProps extends Readonly<React.HTMLAttributes<HTMLSpanElement>> {
  readonly variant?: BadgeVariant
  readonly size?: BadgeSize
}

const sizeMap: Record<BadgeSize, 'small' | 'medium'> = {
  sm: 'small',
  md: 'medium'
}

const colorVar: Record<Exclude<BadgeVariant,'default'>, string> = {
  success: 'var(--color-success)',
  warning: 'var(--color-warning)',
  danger: 'var(--color-danger)',
  info: 'var(--color-info)',
  neutral: 'var(--color-neutral)'
}

export function Badge({ className, variant = 'default', size = 'md', children, ...rest }: BadgeProps) {
  const color = variant === 'default' ? undefined : colorVar[variant]
  return <Tag size={sizeMap[size]} color={color} className={cn(className)} {...rest}>{children}</Tag>
}
