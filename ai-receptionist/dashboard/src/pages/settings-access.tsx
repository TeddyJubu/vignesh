import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { apiFetch } from '@/lib/api'
import { useApiState } from '@/lib/use-api'
import { Page, PageHeader } from '@/components/page'

function normalizePhone(raw: string) {
  return raw.replace(/\D+/g, '')
}

type Role = 'admin' | 'manager' | 'client'

type AccessRoleRow = {
  phone: string
  role: Role | string
  permissions?: Record<string, boolean>
  created_at?: string
}

type AllowlistSettings = {
  allow_all: boolean
  allow_list: string[]
}

const managerPermKeys = [
  'settings',
  'instructions',
  'dreams',
  'providers',
  'memory',
  'access',
] as const

function permsToLabel(perms?: Record<string, boolean>) {
  if (!perms) return '—'
  const enabled = managerPermKeys.filter((k) => perms[k])
  return enabled.length ? enabled.join(', ') : 'none'
}

async function loadRoles(role: Role) {
  const res = await apiFetch<{ roles: AccessRoleRow[] }>(`/access/roles?role=${role}`)
  return res.roles
}

async function loadAllowlist() {
  const res = await apiFetch<AllowlistSettings>('/access/allowlist')
  return res
}

export function SettingsAccessPage() {
  const admins = useApiState(() => loadRoles('admin'), [])
  const managers = useApiState(() => loadRoles('manager'), [])
  const clients = useApiState(() => loadRoles('client'), [])
  const allowlist = useApiState(loadAllowlist, [])

  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // dialogs
  const [adminOpen, setAdminOpen] = useState(false)
  const [adminPhone, setAdminPhone] = useState('')

  const [managerOpen, setManagerOpen] = useState(false)
  const [managerPhone, setManagerPhone] = useState('')
  const [managerPerms, setManagerPerms] = useState<Record<string, boolean>>({})

  const [allowOpen, setAllowOpen] = useState(false)
  const allowAll = allowlist.data?.allow_all ?? true
  const allowList = allowlist.data?.allow_list ?? []
  const [draftAllowAll, setDraftAllowAll] = useState<boolean>(allowAll)
  const [draftAllowList, setDraftAllowList] = useState<string>(allowList.join('\n'))

  const normalizedAdmin = useMemo(() => normalizePhone(adminPhone), [adminPhone])
  const normalizedManager = useMemo(() => normalizePhone(managerPhone), [managerPhone])

  async function upsertRole(payload: { phone: string; role: Role; permissions?: Record<string, boolean> }) {
    await apiFetch('/access/roles', { method: 'PUT', json: payload })
  }

  async function deleteRole(phone: string) {
    await apiFetch('/access/roles', { method: 'DELETE', json: { phone } })
  }

  async function saveAllowlist(next: AllowlistSettings) {
    await apiFetch('/access/allowlist', { method: 'PUT', json: next })
  }

  async function addAdmin() {
    setBusy(true)
    setError(null)
    try {
      await upsertRole({ phone: normalizedAdmin, role: 'admin' })
      setAdminOpen(false)
      setAdminPhone('')
      await admins.refresh()
    } catch (e: any) {
      setError(e?.message ?? 'Could not add administrator')
    } finally {
      setBusy(false)
    }
  }

  async function addOrUpdateManager() {
    setBusy(true)
    setError(null)
    try {
      await upsertRole({
        phone: normalizedManager,
        role: 'manager',
        permissions: managerPerms,
      })
      setManagerOpen(false)
      setManagerPhone('')
      setManagerPerms({})
      await managers.refresh()
    } catch (e: any) {
      setError(e?.message ?? 'Could not save manager')
    } finally {
      setBusy(false)
    }
  }

  async function remove(role: Role, phone: string) {
    setBusy(true)
    setError(null)
    try {
      await deleteRole(phone)
      if (role === 'admin') await admins.refresh()
      if (role === 'manager') await managers.refresh()
      if (role === 'client') await clients.refresh()
    } catch (e: any) {
      setError(e?.message ?? 'Could not remove role')
    } finally {
      setBusy(false)
    }
  }

  async function openEditManager(row: AccessRoleRow) {
    setManagerPhone(row.phone)
    setManagerPerms(row.permissions ?? {})
    setManagerOpen(true)
  }

  async function applyAllowlist() {
    setBusy(true)
    setError(null)
    try {
      const nextList = draftAllowList
        .split('\n')
        .map((l) => normalizePhone(l.trim()))
        .filter(Boolean)
      await saveAllowlist({ allow_all: draftAllowAll, allow_list: nextList })
      setAllowOpen(false)
      await allowlist.refresh()
    } catch (e: any) {
      setError(e?.message ?? 'Could not save allowlist')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Page>
      <PageHeader
        title="Access"
        description="Manage dashboard roles and WhatsApp inbound allowlist."
        right={
          <div className="flex items-center gap-2">
            <Badge variant="secondary">
              WhatsApp inbound: {allowAll ? 'Allow all' : 'Allow list'}
            </Badge>
            <Button variant="secondary" onClick={() => void Promise.all([admins.refresh(), managers.refresh(), clients.refresh(), allowlist.refresh()])}>
              Reload
            </Button>
          </div>
        }
      />

      {error && (
        <Alert>
          <AlertTitle>Update failed</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {(admins.error || managers.error || clients.error || allowlist.error) && (
        <Alert>
          <AlertTitle>Could not load access settings</AlertTitle>
          <AlertDescription>
            This page expects the API to expose <code>GET /api/access/roles</code> and <code>GET /api/access/allowlist</code>.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between gap-2">
            <CardTitle className="text-base">Roles</CardTitle>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="admins" className="w-full">
              <TabsList className="grid h-10 w-full grid-cols-3">
                <TabsTrigger value="admins">Administrators</TabsTrigger>
                <TabsTrigger value="managers">Managers</TabsTrigger>
                <TabsTrigger value="clients">Clients</TabsTrigger>
              </TabsList>

              <TabsContent value="admins" className="mt-4 space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-sm text-muted-foreground">Admins have full dashboard access.</div>
                  <Dialog open={adminOpen} onOpenChange={setAdminOpen}>
                    <DialogTrigger render={<Button size="sm">Add admin</Button>} />
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Add administrator</DialogTitle>
                      </DialogHeader>
                      <div className="space-y-2">
                        <Label htmlFor="admin-phone">Phone</Label>
                        <Input
                          id="admin-phone"
                          value={adminPhone}
                          onChange={(e) => setAdminPhone(e.target.value)}
                          placeholder="e.g. 6590013157"
                          inputMode="tel"
                        />
                        <div className="text-xs text-muted-foreground">
                          Normalized as <code>{normalizedAdmin || '—'}</code>
                        </div>
                      </div>
                      <DialogFooter>
                        <Button onClick={() => void addAdmin()} disabled={busy || normalizedAdmin.length < 7}>
                          {busy ? 'Saving…' : 'Save'}
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>

                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Phone</TableHead>
                      <TableHead className="w-[1%]" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(admins.data ?? []).map((row) => (
                      <TableRow key={row.phone}>
                        <TableCell className="font-mono text-xs">{row.phone}</TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={() => void remove('admin', row.phone)}
                            disabled={busy}
                          >
                            Remove
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                    {!admins.loading && (admins.data?.length ?? 0) === 0 ? (
                      <TableRow>
                        <TableCell colSpan={2} className="text-sm text-muted-foreground">
                          No administrators yet.
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </TabsContent>

              <TabsContent value="managers" className="mt-4 space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-sm text-muted-foreground">
                    Managers authenticate via OTP and are limited by permissions.
                  </div>
                  <Dialog open={managerOpen} onOpenChange={setManagerOpen}>
                    <DialogTrigger
                      render={
                        <Button
                          size="sm"
                          onClick={() => {
                            setManagerPhone('')
                            setManagerPerms({})
                            setManagerOpen(true)
                          }}
                        >
                          Add manager
                        </Button>
                      }
                    />
                    <DialogContent className="sm:max-w-xl">
                      <DialogHeader>
                        <DialogTitle>Manager access</DialogTitle>
                      </DialogHeader>

                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label htmlFor="manager-phone">Phone</Label>
                          <Input
                            id="manager-phone"
                            value={managerPhone}
                            onChange={(e) => setManagerPhone(e.target.value)}
                            placeholder="e.g. 6580286424"
                            inputMode="tel"
                          />
                          <div className="text-xs text-muted-foreground">
                            Normalized as <code>{normalizedManager || '—'}</code>
                          </div>
                        </div>

                        <div className="space-y-2">
                          <div className="text-sm font-medium">Permissions</div>
                          <div className="grid grid-cols-2 gap-2">
                            {managerPermKeys.map((key) => (
                              <label key={key} className="flex items-center gap-2 text-sm">
                                <input
                                  type="checkbox"
                                  checked={Boolean(managerPerms[key])}
                                  onChange={(e) =>
                                    setManagerPerms((p) => ({
                                      ...p,
                                      [key]: e.target.checked,
                                    }))
                                  }
                                />
                                <span className="capitalize">{key}</span>
                              </label>
                            ))}
                          </div>
                        </div>
                      </div>

                      <DialogFooter>
                        <Button
                          onClick={() => void addOrUpdateManager()}
                          disabled={busy || normalizedManager.length < 7}
                        >
                          {busy ? 'Saving…' : 'Save'}
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>

                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Phone</TableHead>
                      <TableHead>Permissions</TableHead>
                      <TableHead className="w-[1%]" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(managers.data ?? []).map((row) => (
                      <TableRow key={row.phone}>
                        <TableCell className="font-mono text-xs">{row.phone}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {permsToLabel(row.permissions)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            <Button size="sm" variant="secondary" onClick={() => void openEditManager(row)} disabled={busy}>
                              Edit
                            </Button>
                            <Button size="sm" variant="destructive" onClick={() => void remove('manager', row.phone)} disabled={busy}>
                              Remove
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                    {!managers.loading && (managers.data?.length ?? 0) === 0 ? (
                      <TableRow>
                        <TableCell colSpan={3} className="text-sm text-muted-foreground">
                          No managers yet.
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </TabsContent>

              <TabsContent value="clients" className="mt-4 space-y-3">
                <div className="text-sm text-muted-foreground">
                  Clients are auto-captured when Julia processes a number. This list is read-only.
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Phone</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="w-[1%]" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(clients.data ?? []).map((row) => (
                      <TableRow key={row.phone}>
                        <TableCell className="font-mono text-xs">{row.phone}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">{row.created_at ?? '—'}</TableCell>
                        <TableCell className="text-right">
                          <Button size="sm" variant="secondary" onClick={() => void remove('client', row.phone)} disabled={busy}>
                            Remove
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                    {!clients.loading && (clients.data?.length ?? 0) === 0 ? (
                      <TableRow>
                        <TableCell colSpan={3} className="text-sm text-muted-foreground">
                          No clients yet.
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-2">
            <CardTitle className="text-base">WhatsApp inbound</CardTitle>
            <Dialog
              open={allowOpen}
              onOpenChange={(open) => {
                setAllowOpen(open)
                if (open) {
                  setDraftAllowAll(allowAll)
                  setDraftAllowList(allowList.join('\n'))
                }
              }}
            >
              <DialogTrigger
                render={
                  <Button size="sm" variant="secondary">
                    Edit
                  </Button>
                }
              />
              <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>Inbound allowlist</DialogTitle>
                </DialogHeader>

                <div className="space-y-3">
                  <div className="space-y-2">
                    <div className="text-sm font-medium">Mode</div>
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="radio"
                        name="allow-mode"
                        checked={draftAllowAll}
                        onChange={() => setDraftAllowAll(true)}
                      />
                      Allow all numbers
                    </label>
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="radio"
                        name="allow-mode"
                        checked={!draftAllowAll}
                        onChange={() => setDraftAllowAll(false)}
                      />
                      Allow only these numbers
                    </label>
                  </div>

                  {!draftAllowAll ? (
                    <div className="space-y-2">
                      <Label htmlFor="allow-list">Allowed numbers (one per line)</Label>
                      <textarea
                        id="allow-list"
                        className="min-h-48 w-full rounded-md border bg-background px-3 py-2 font-mono text-xs outline-none focus:ring-2 focus:ring-ring"
                        value={draftAllowList}
                        onChange={(e) => setDraftAllowList(e.target.value)}
                        placeholder="6590013157\n6580286424"
                      />
                      <div className="text-xs text-muted-foreground">
                        Inputs are normalized to digits only on save.
                      </div>
                    </div>
                  ) : null}
                </div>

                <DialogFooter>
                  <Button onClick={() => void applyAllowlist()} disabled={busy}>
                    {busy ? 'Saving…' : 'Save'}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-sm text-muted-foreground">
              {allowAll ? (
                <>All inbound numbers are allowed.</>
              ) : (
                <>
                  Only <span className="font-medium text-foreground">{allowList.length}</span>{' '}
                  number(s) are allowed.
                </>
              )}
            </div>

            {!allowAll ? (
              <div className="space-y-2">
                <div className="text-xs font-medium text-muted-foreground">Allowed</div>
                <div className="max-h-64 space-y-1 overflow-auto rounded-md border bg-muted/20 p-2 font-mono text-xs">
                  {allowList.length ? (
                    allowList.map((p) => <div key={p}>{p}</div>)
                  ) : (
                    <div className="text-muted-foreground">No numbers in list.</div>
                  )}
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </Page>
  )
}

