import * as React from 'react'
import { cn } from '../../lib/utils'

export interface EmptyStateProps extends Readonly<React.HTMLAttributes<HTMLDivElement>> {
  readonly icon?: React.ReactNode
  readonly title?: string
  readonly description?: string
}

export function EmptyState({ icon, title, description, className, ...props }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center text-center border border-dashed border-border rounded-lg p-8 bg-background/40', className)} {...props}>
      {icon && <div className="mb-3 text-muted-foreground">{icon}</div>}
      {title && <div className="text-sm font-medium text-foreground">{title}</div>}
      {description && <div className="text-xs text-muted-foreground mt-1">{description}</div>}
    </div>
  )
}
