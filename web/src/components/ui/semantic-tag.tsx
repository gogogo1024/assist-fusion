import React from 'react'
import { Tag, TagProps } from '@arco-design/web-react'

export type SemanticVariant = 'success' | 'warning' | 'danger' | 'info' | 'neutral' | 'accent' | 'default'

const variantColorVar: Record<SemanticVariant, string | undefined> = {
  success: 'var(--color-success)',
  warning: 'var(--color-warning)',
  danger: 'var(--color-danger)',
  info: 'var(--color-info)',
  neutral: 'var(--color-neutral)',
  accent: 'var(--c-brand-accent)',
  default: undefined
}

export interface SemanticTagProps extends Omit<TagProps, 'color'> {
  variant?: SemanticVariant
  /** allow override while still using variant fallback */
  color?: string
}

/**
 * SemanticTag - 语义化封装，内部使用 CSS 变量以统一来源；可随主题/变量刷新。
 */
export function SemanticTag({ variant = 'default', color, style, ...rest }: Readonly<SemanticTagProps>){
  const resolved = color || variantColorVar[variant]
  return <Tag color={resolved} style={style} {...rest} />
}
