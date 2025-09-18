import * as React from 'react'
import { cn } from '../../lib/utils'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

export const Input = React.forwardRef<HTMLInputElement, Readonly<InputProps>>(
  ({ className, ...props }, ref) => {
    return (
      <input
        ref={ref}
        className={cn('input h-9', className)}
        {...props}
      />
    )
  }
)
Input.displayName = 'Input'
