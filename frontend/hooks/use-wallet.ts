"use client"

import { useState, useEffect, useCallback } from "react"
import type { WalletState } from "@/types"

declare global {
  interface Window {
    ethereum?: any
  }
}

export function useWallet() {
  const [wallet, setWallet] = useState<WalletState>({
    isConnected: false,
    address: null,
    isConnecting: false,
  })

  const checkConnection = useCallback(async () => {
    if (typeof window.ethereum === "undefined") return

    try {
      const accounts = await window.ethereum.request({ method: "eth_accounts" })
      if (accounts.length > 0) {
        setWallet({
          isConnected: true,
          address: accounts[0],
          isConnecting: false,
        })
      }
    } catch (error) {
      console.error("Failed to check wallet connection:", error)
    }
  }, [])

  const connect = useCallback(async () => {
    if (typeof window.ethereum === "undefined") {
      throw new Error("MetaMask is not installed")
    }

    setWallet((prev) => ({ ...prev, isConnecting: true }))

    try {
      const accounts = await window.ethereum.request({ method: "eth_requestAccounts" })
      setWallet({
        isConnected: true,
        address: accounts[0],
        isConnecting: false,
      })
    } catch (error) {
      setWallet((prev) => ({ ...prev, isConnecting: false }))
      throw error
    }
  }, [])

  const disconnect = useCallback(() => {
    setWallet({
      isConnected: false,
      address: null,
      isConnecting: false,
    })
  }, [])

  useEffect(() => {
    checkConnection()

    if (window.ethereum) {
      window.ethereum.on("accountsChanged", (accounts: string[]) => {
        if (accounts.length === 0) {
          disconnect()
        } else {
          setWallet((prev) => ({ ...prev, address: accounts[0] }))
        }
      })
    }

    return () => {
      if (window.ethereum) {
        window.ethereum.removeAllListeners("accountsChanged")
      }
    }
  }, [checkConnection, disconnect])

  return {
    ...wallet,
    connect,
    disconnect,
  }
}
