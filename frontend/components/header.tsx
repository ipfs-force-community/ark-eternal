"use client"

import { useWallet } from "@/hooks/use-wallet"
import { config } from "@/lib/config"
import { Wallet, LogOut } from "lucide-react"
import { Button } from "@/components/ui/button"

export function Header() {
  const { isConnected, address, isConnecting, connect, disconnect } = useWallet()

  const handleConnect = async () => {
    try {
      await connect()
    } catch (error) {
      console.error("Failed to connect wallet:", error)
    }
  }

  const formatAddress = (addr: string) => {
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`
  }

  return (
    <header className="bg-white shadow-sm">
      <div className="container mx-auto px-4 py-3 flex items-center justify-between">
        <div className="flex items-center">
          <h1 className="text-2xl font-['Pacifico'] text-primary mr-2">{config.app.name}</h1>
          <span className="text-gray-700 font-medium">{config.app.name}</span>
        </div>

        <div className="flex items-center gap-3">
          {!isConnected ? (
            <Button onClick={handleConnect} disabled={isConnecting} className="flex items-center gap-2">
              <Wallet className="w-4 h-4" />
              {isConnecting ? "Connecting..." : "Connect Wallet"}
            </Button>
          ) : (
            <div className="flex items-center gap-3">
              <div className="bg-gray-100 px-3 py-2 rounded-full flex items-center">
                <Wallet className="w-4 h-4 mr-2 text-primary" />
                <span className="text-gray-700 text-sm">{address && formatAddress(address)}</span>
              </div>
              <Button variant="ghost" size="sm" onClick={disconnect} className="p-2">
                <LogOut className="w-4 h-4" />
              </Button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
