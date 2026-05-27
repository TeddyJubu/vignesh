import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Page, PageHeader } from '@/components/page'
import { cn } from '@/lib/utils'
import { apiFetch } from '@/lib/api'
import type { DreamProposal } from '@/lib/models'
import { useApiState } from '@/lib/use-api'

function statusVariant(status: string) {
  if (status === 'applied') return 'default' as const
  if (status === 'rejected') return 'secondary' as const
  return 'outline' as const
}

export function MemoryDreamsPage() {
  const state = useApiState<DreamProposal[]>(() => apiFetch('/dreams'), [])
  const [applying, setApplying] = useState<string | null>(null)

  const proposals = state.data ?? []
  const count = proposals.length
  const pending = proposals.filter((p) => p.status === 'proposed').length

  const subtitle = useMemo(() => {
    if (state.loading) return 'Loading…'
    if (!count) return 'No proposals'
    if (!pending) return `${count} total`
    return `${pending} pending · ${count} total`
  }, [state.loading, count, pending])

  async function apply(id: string) {
    setApplying(id)
    try {
      await apiFetch(`/dreams/${encodeURIComponent(id)}/apply`, { method: 'POST' })
      await state.refresh()
    } finally {
      setApplying(null)
    }
  }

  return (
    <Page>
      <PageHeader
        title="Dreams"
        description="Review proposed instruction updates. Nothing is applied automatically."
        right={<Badge variant="secondary">{subtitle}</Badge>}
      />

      {state.error && (
        <Alert>
          <AlertTitle>Could not load dream proposals</AlertTitle>
          <AlertDescription>
            The API should expose <code>GET /api/dreams</code> and{' '}
            <code>POST /api/dreams/:id/apply</code>.
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-2">
          <CardTitle className="text-base">Proposals</CardTitle>
          <Button variant="secondary" size="sm" onClick={() => void state.refresh()}>
            Refresh
          </Button>
        </CardHeader>
        <CardContent>
          {count ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[170px]">Created</TableHead>
                    <TableHead>Title</TableHead>
                    <TableHead className="w-[130px]">Status</TableHead>
                    <TableHead className="w-[140px] text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {proposals.map((p) => (
                    <TableRow key={p.id}>
                      <TableCell className="align-top text-sm text-muted-foreground">
                        {new Date(p.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="align-top">
                        <div className="text-sm font-medium">{p.title}</div>
                        {p.rationale ? (
                          <div className="mt-1 text-xs text-muted-foreground">
                            {p.rationale}
                          </div>
                        ) : null}
                      </TableCell>
                      <TableCell className="align-top">
                        <Badge variant={statusVariant(p.status)}>{p.status}</Badge>
                      </TableCell>
                      <TableCell className="align-top text-right">
                        <Dialog>
                          <DialogTrigger className={cn(buttonVariants({ variant: "secondary", size: "sm" }))}>
                              Review
                          </DialogTrigger>
                          <DialogContent className="max-w-3xl">
                            <DialogHeader>
                              <DialogTitle>{p.title}</DialogTitle>
                              <DialogDescription>
                                Review the diff, then apply if approved.
                              </DialogDescription>
                            </DialogHeader>

                            <Separator />

                            <div className="space-y-2">
                              <div className="text-sm font-medium">Diff</div>
                              <pre className="max-h-[55vh] overflow-auto rounded-md border bg-muted p-3 text-xs leading-5">
                                {p.diff ?? 'No diff provided.'}
                              </pre>
                            </div>

                            <div className="flex justify-end gap-2">
                              <Button
                                onClick={() => void apply(p.id)}
                                disabled={p.status !== 'proposed' || applying === p.id}
                              >
                                {applying === p.id ? 'Applying…' : 'Apply'}
                              </Button>
                            </div>
                          </DialogContent>
                        </Dialog>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-sm text-muted-foreground">
              No proposals yet. When Graphiti is running, it can write proposals for review.
            </div>
          )}
        </CardContent>
      </Card>
    </Page>
  )
}

