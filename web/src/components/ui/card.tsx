import * as React from 'react'
import { cn } from '../../lib/utils'

export function Card({ className, ...props }: Readonly<React.HTMLAttributes<HTMLDivElement>>) {
  return <div className={cn('card p-4', className)} {...props} />
}

export function CardHeader({ className, ...props }: Readonly<React.HTMLAttributes<HTMLDivElement>>) {
  return <div className={cn('mb-3', className)} {...props} />
}

export function CardTitle({ className, children, ...props }: Readonly<React.HTMLAttributes<HTMLHeadingElement>>) {
  return <h3 className={cn('text-lg font-semibold', className)} {...props}>{children ?? ' '}</h3>
}

export function CardContent({ className, ...props }: Readonly<React.HTMLAttributes<HTMLDivElement>>) {
  return <div className={cn('', className)} {...props} />
}
