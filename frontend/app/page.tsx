"use client"

import { Header } from "@/components/header"
import { FileManagement } from "@/components/file-management"
import { useToast } from "@/components/toast"
import { config } from "@/lib/config"

export default function HomePage() {
  const { ToastContainer } = useToast()

  return (
    <div className="bg-gray-50 min-h-screen flex flex-col">
      <Header />

      <main className="flex-grow container mx-auto px-4 py-6">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <FileManagement />
          </div>

          <div className="space-y-6">
            {/* Proof Sets Panel - Placeholder */}
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h2 className="text-lg font-medium text-gray-800 mb-4">Proof Sets</h2>
              <p className="text-gray-500 text-sm">Coming soon...</p>
            </div>

            {/* Storage Provider Panel - Placeholder */}
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h2 className="text-lg font-medium text-gray-800 mb-4">Storage Provider</h2>
              <p className="text-gray-500 text-sm">Coming soon...</p>
            </div>
          </div>
        </div>
      </main>

      <footer className="bg-white border-t border-gray-200 py-3">
        <div className="container mx-auto px-4 flex justify-between items-center">
          <div className="flex items-center text-sm text-gray-500">
            <span className="w-4 h-4 flex items-center justify-center mr-1 text-green-500">●</span>
            System Status: Running
          </div>
          <div className="text-sm text-gray-500">
            {config.app.name} v{config.app.version} | Last updated: {new Date().toISOString().split('T')[0]}
          </div>
        </div>
      </footer>

      <ToastContainer />
    </div>
  )
}
