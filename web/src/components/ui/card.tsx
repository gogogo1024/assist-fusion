import * as React from 'react'
import { Card as ArcoCard } from '@arco-design/web-react'
import { cn } from '../../lib/utils'

// 简化后的 Card：移除自定义 variant (contrast/glass)，直接使用 Arco 默认样式。
// 仅保留 title (可为 ReactNode) 与原生 div 属性。避免与 HTMLAttributes 中的 title(string) 冲突需 Omit。
interface CardProps extends Omit<React.HTMLAttributes<HTMLDivElement>, 'title'> {
  readonly title?: React.ReactNode
}

export function Card({ className, title, children, ...rest }: CardProps) {
  return (
    <ArcoCard
      className={cn('p-4', className)}
      title={title}
      {...rest}
    >
      {children}
    </ArcoCard>
  )
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
