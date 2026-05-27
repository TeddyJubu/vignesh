import { NavLink, useLocation } from 'react-router-dom'
import type { PropsWithChildren } from 'react'
import { cn } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  Activity,
  Brain,
  Cable,
  Home,
  Menu,
  Settings,
  Sparkles,
} from 'lucide-react'

type NavItem = {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
}

const nav: Array<{ label: string; items: NavItem[] }> = [
  {
    label: 'Overview',
    items: [{ to: '/', label: 'Overview', icon: Home }],
  },
  {
    label: 'Settings',
    items: [
      { to: '/settings/providers', label: 'Providers', icon: Settings },
      { to: '/settings/instructions', label: 'Instructions', icon: Activity },
    ],
  },
  {
    label: 'Memory',
    items: [
      { to: '/memory/recall', label: 'Recall', icon: Brain },
      { to: '/memory/dreams', label: 'Dreams', icon: Sparkles },
    ],
  },
  {
    label: 'Integrations',
    items: [{ to: '/integrations/composio', label: 'Composio', icon: Cable }],
  },
]

function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="flex h-full flex-col gap-3 px-3 py-4">
      <div className="px-1">
        <div className="text-sm font-semibold tracking-tight">
          AI Receptionist
        </div>
        <div className="text-xs text-muted-foreground">Dashboard</div>
      </div>

      <Separator />

      <div className="flex flex-col gap-4">
        {nav.map((section) => (
          <div key={section.label} className="flex flex-col gap-2">
            <div className="px-1 text-xs font-medium text-muted-foreground">
              {section.label.toUpperCase()}
            </div>
            <div className="flex flex-col gap-1">
              {section.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  onClick={onNavigate}
                  className={({ isActive }) =>
                    cn(
                      'group flex items-center gap-2 rounded-md px-2 py-2 text-sm transition-colors',
                      isActive
                        ? 'bg-accent text-accent-foreground'
                        : 'text-foreground/80 hover:bg-accent/60 hover:text-foreground',
                    )
                  }
                >
                  <item.icon className="h-4 w-4 opacity-80" />
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-auto px-1 text-xs text-muted-foreground">
        <span className="font-medium">API</span> under <code>/api</code>
      </div>
    </nav>
  )
}

export function AppShell({ children }: PropsWithChildren) {
  const { pathname } = useLocation()
  const current = nav
    .flatMap((s) => s.items)
    .find((i) => i.to === pathname)?.label

  return (
    <div className="min-h-svh bg-background">
      <div className="mx-auto flex w-full max-w-6xl gap-0">
        <aside className="sticky top-0 hidden h-svh w-64 shrink-0 border-r md:block">
          <SidebarNav />
        </aside>

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
            <div className="flex items-center justify-between gap-2 px-4 py-3">
              <div className="flex items-center gap-2">
                <Sheet>
                  <SheetTrigger
                    className={cn(buttonVariants({ variant: 'ghost', size: 'icon' }), 'md:hidden')}
                    aria-label="Open menu"
                  >
                    <Menu className="h-4 w-4" />
                  </SheetTrigger>
                  <SheetContent side="left" className="p-0">
                    <SidebarNav />
                  </SheetContent>
                </Sheet>

                <div className="flex flex-col">
                  <div className="text-sm font-semibold leading-4">
                    {current ?? 'Dashboard'}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {pathname}
                  </div>
                </div>
              </div>

              <Tooltip>
                <TooltipTrigger className={cn(buttonVariants({ variant: 'secondary', size: 'sm' }))}>
                  <span className="hidden sm:inline">Local</span>
                  <span className="sm:hidden">Local</span>
                </TooltipTrigger>
                <TooltipContent>
                  This UI expects the Go app to serve the API.
                </TooltipContent>
              </Tooltip>
            </div>
          </header>

          <main className="min-w-0 flex-1 px-4 py-6">{children}</main>
        </div>
      </div>
    </div>
  )
}

