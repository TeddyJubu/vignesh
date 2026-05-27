import type { PropsWithChildren } from 'react'

export function Page({ children }: PropsWithChildren) {
  return <div className="flex w-full flex-col gap-6">{children}</div>
}

export function PageHeader({
  title,
  description,
  right,
}: {
  title: string
  description?: string
  right?: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        <div className="text-xl font-semibold tracking-tight">{title}</div>
        {description ? (
          <div className="mt-1 text-sm text-muted-foreground">{description}</div>
        ) : null}
      </div>
      {right ? <div className="shrink-0">{right}</div> : null}
    </div>
  )
}

