import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import { Menu, X } from 'lucide-react'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'

export function AppShell() {
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false)

  return (
    <div className="flex h-screen overflow-hidden bg-slate-50">
      {/* Desktop sidebar — always visible on lg+ */}
      <div className="hidden lg:flex">
        <Sidebar />
      </div>

      {/* Mobile sidebar overlay */}
      {mobileSidebarOpen && (
        <div
          className="fixed inset-0 z-40 lg:hidden"
          aria-modal="true"
          role="dialog"
          aria-label="Navigation menu"
        >
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => setMobileSidebarOpen(false)}
            aria-hidden="true"
          />

          {/* Panel */}
          <div className="relative flex h-full z-50">
            <Sidebar />
            <button
              className="absolute top-4 right-4 p-1.5 rounded-lg bg-slate-700 text-slate-300
                hover:bg-slate-600 hover:text-white transition-colors
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
              onClick={() => setMobileSidebarOpen(false)}
              aria-label="Close navigation"
            >
              <X size={18} />
            </button>
          </div>
        </div>
      )}

      {/* Main column */}
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        {/* Mobile header bar */}
        <div className="lg:hidden flex items-center h-16 px-4 bg-white border-b border-slate-200 shrink-0">
          <button
            className="p-2 rounded-lg text-slate-500 hover:bg-slate-100 hover:text-slate-800
              transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
            onClick={() => setMobileSidebarOpen(true)}
            aria-label="Open navigation"
          >
            <Menu size={20} />
          </button>
          <span className="ml-3 font-semibold text-slate-800 text-base">
            eKYC Platform
          </span>
        </div>

        {/* Desktop topbar */}
        <div className="hidden lg:block">
          <Topbar />
        </div>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
