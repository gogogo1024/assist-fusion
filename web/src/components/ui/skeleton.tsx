import * as React from 'react'
import { cn } from '../../lib/utils'

export interface SkeletonProps extends Readonly<React.HTMLAttributes<HTMLDivElement>> {
  readonly rounded?: 'sm'|'md'|'lg'|'full'
}

export function Skeleton({ className, rounded = 'md', ...props }: SkeletonProps) {
  let r: string
  switch (rounded) {
    case 'full': r = 'rounded-full'; break
    case 'lg': r = 'rounded-lg'; break
    case 'sm': r = 'rounded'; break
    default: r = 'rounded-md'
  }
  return (
    <div className={cn('animate-pulse bg-muted/60', r, className)} {...props} />
  )
}
