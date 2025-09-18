import * as React from 'react'
import { cn } from '../../lib/utils'

export interface TabsProps extends Readonly<React.HTMLAttributes<HTMLDivElement>> {
  readonly value?: string
  readonly onValueChange?: (v: string) => void
}

export function Tabs({ className, value, onValueChange, ...props }: TabsProps) {
  return (
    <div className={cn('w-full', className)} data-tabs-value={value} {...props} />
  )
}

export const TabsList = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('inline-flex h-10 items-center justify-center rounded-md bg-muted p-1 text-muted-foreground', className)}
      {...props}
    />
  )
)
TabsList.displayName = 'TabsList'

export interface TabsTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  value: string
  active?: boolean
}

export const TabsTrigger = React.forwardRef<HTMLButtonElement, TabsTriggerProps>(
  ({ className, value: _value, active, ...props }, ref) => (
    <button
      ref={ref}
      className={cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50',
        active ? 'bg-background text-foreground shadow' : 'text-muted-foreground hover:text-foreground',
        className,
      )}
      {...props}
    />
  )
)
TabsTrigger.displayName = 'TabsTrigger'

export function TabsContent({ className, ...props }: Readonly<React.HTMLAttributes<HTMLDivElement>>) {
  return <div className={cn('mt-2', className)} {...props} />
}
