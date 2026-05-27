import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Page, PageHeader } from '@/components/page'
import { apiFetch } from '@/lib/api'
import type { RecallResult } from '@/lib/models'
import { Search } from 'lucide-react'

export function MemoryRecallPage() {
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<RecallResult | null>(null)

  const hasResults = (result?.items?.length ?? 0) > 0

  const title = useMemo(() => {
    if (!result) return 'No query yet'
    if (!hasResults) return 'No matches'
    return `${result.items.length} matches`
  }, [result, hasResults])

  async function run() {
    setLoading(true)
    setError(null)
    try {
      const next = await apiFetch<RecallResult>(`/memory/recall?q=${encodeURIComponent(q)}`)
      setResult(next)
    } catch (e) {
      if (typeof e === 'object' && e && 'message' in e) {
        setError(String((e as any).message))
      } else {
        setError(e instanceof Error ? e.message : String(e))
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Page>
      <PageHeader
        title="Recall"
        description="Search memory via the Graphiti sidecar (proxied through Go)."
        right={<Badge variant="secondary">Graphiti</Badge>}
      />

      {error && (
        <Alert>
          <AlertTitle>Recall failed</AlertTitle>
          <AlertDescription>
            <div>
              The API should expose <code>GET /api/memory/recall</code> and proxy to the sidecar.
            </div>
            <div className="mt-2 text-xs text-muted-foreground">
              {error}
            </div>
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Query</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1 space-y-2">
            <div className="text-sm font-medium">Search</div>
            <Input
              value={q}
              placeholder="Try: 'preferred appointment times', 'lead budget', 'dog-friendly'…"
              onChange={(e) => setQ(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') void run()
              }}
            />
          </div>
          <Button onClick={() => void run()} disabled={!q || loading}>
            <Search className="mr-2 h-4 w-4" />
            Search
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-2">
          <CardTitle className="text-base">Results</CardTitle>
          <Badge variant="secondary">{title}</Badge>
        </CardHeader>
        <CardContent>
          {result?.snippet ? (
            <div className="mb-4 rounded-md border bg-muted p-3 text-xs leading-5">
              {result.snippet}
            </div>
          ) : null}
          {hasResults ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[140px]">Score</TableHead>
                    <TableHead>Memory</TableHead>
                    <TableHead className="w-[140px]">Source</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {result!.items.map((item, idx) => (
                    <TableRow key={`${item.created_at ?? ''}-${idx}`}>
                      <TableCell className="align-top">
                        <span className="text-sm text-muted-foreground">
                          {item.score ?? '—'}
                        </span>
                      </TableCell>
                      <TableCell className="align-top">
                        <div className="text-sm whitespace-pre-wrap">{item.text}</div>
                        {item.created_at ? (
                          <div className="mt-1 text-xs text-muted-foreground">
                            {new Date(item.created_at).toLocaleString()}
                          </div>
                        ) : null}
                      </TableCell>
                      <TableCell className="align-top">
                        <span className="text-sm text-muted-foreground">
                          {item.source ?? '—'}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-sm text-muted-foreground">
              Run a search to see results.
            </div>
          )}
        </CardContent>
      </Card>
    </Page>
  )
}

