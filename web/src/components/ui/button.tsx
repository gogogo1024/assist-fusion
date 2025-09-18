import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const buttonVariants = cva(
  'btn inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:opacity-60 disabled:pointer-events-none',
  {
    variants: {
      variant: {
        default: 'btn-primary',
        muted: 'btn-muted',
        success: 'bg-emerald-600 text-white hover:opacity-90',
        warning: 'bg-amber-500 text-zinc-900 hover:opacity-90',
        danger: 'bg-rose-600 text-white hover:opacity-90',
        info: 'bg-sky-600 text-white hover:opacity-90',
        neutral: 'bg-muted text-muted-foreground',
        outline: 'bg-transparent border border-border hover:bg-muted',
        ghost: 'bg-transparent hover:bg-muted',
      },
      size: {
        sm: 'h-8 px-3',
        md: 'h-9 px-4',
        lg: 'h-10 px-5',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        className={cn(buttonVariants({ variant, size }), className)}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = 'Button'

export { Button, buttonVariants }
