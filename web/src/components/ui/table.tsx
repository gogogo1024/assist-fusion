import * as React from 'react'
import { Table as ArcoTable } from '@arco-design/web-react'
import { cn } from '../../lib/utils'

// 兼容层：旧的 <Table><TableHeader/><TableBody/>... 结构被频繁使用时，可选择渐进迁移。
// 这里提供一个轻量包装：如果传入 children 里已经是 <ArcoTable columns data> 模式，可直接改业务端；
// 暂时维持旧导出，内部用原生 <table>，后续再统一替换成真正的 ArcoTable columns API。

export const Table = React.forwardRef<HTMLDivElement, Readonly<React.HTMLAttributes<HTMLDivElement>>>(
  ({ className, children, ...rest }, ref) => (
    <div ref={ref} className={cn('relative w-full overflow-auto table-wrapper', className)} {...rest}>
      {children}
    </div>
  )
)
Table.displayName = 'Table'

// 保留语义标签，方便逐步把整块替换成 <ArcoTable /> 时全局搜：
export const TableHeader = (props: Readonly<React.HTMLAttributes<HTMLTableSectionElement>>) => <thead {...props} />
export const TableBody = (props: Readonly<React.HTMLAttributes<HTMLTableSectionElement>>) => <tbody {...props} />
export const TableFooter = (props: Readonly<React.HTMLAttributes<HTMLTableSectionElement>>) => <tfoot {...props} />
export const TableRow = (props: Readonly<React.HTMLAttributes<HTMLTableRowElement>>) => <tr {...props} />
export const TableHead = (props: Readonly<React.ThHTMLAttributes<HTMLTableCellElement>>) => <th {...props} />
export const TableCell = (props: Readonly<React.TdHTMLAttributes<HTMLTableCellElement>>) => <td {...props} />
export const TableCaption = (props: Readonly<React.HTMLAttributes<HTMLTableCaptionElement>>) => <caption {...props} />

// 额外导出一个真正的 Arco Table 便于新代码直接使用
export const ATable = ArcoTable
