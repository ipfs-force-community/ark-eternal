"use client"

import { useState, useEffect } from "react"
import { CheckCircle, AlertCircle, X } from "lucide-react"

interface Toast {
  id: string
  message: string
  type: "success" | "error"
}

interface ToastProps {
  toast: Toast
  onRemove: (id: string) => void
}

function ToastItem({ toast, onRemove }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onRemove(toast.id)
    }, 5000)

    return () => clearTimeout(timer)
  }, [toast.id, onRemove])

  return (
    <div
      className={`flex items-center p-4 rounded-lg shadow-lg transform transition-all duration-300 ${
        toast.type === "success" ? "bg-green-500" : "bg-red-500"
      } text-white`}
    >
      {toast.type === "success" ? <CheckCircle className="w-5 h-5 mr-2" /> : <AlertCircle className="w-5 h-5 mr-2" />}
      <span className="flex-1">{toast.message}</span>
      <button onClick={() => onRemove(toast.id)} className="ml-2 p-1 hover:bg-white/20 rounded">
        <X className="w-4 h-4" />
      </button>
    </div>
  )
}

let toastCounter = 0

export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([])

  const showToast = (message: string, type: "success" | "error" = "success") => {
    const id = `toast-${++toastCounter}`
    setToasts((prev) => [...prev, { id, message, type }])
  }

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id))
  }

  const ToastContainer = () => (
    <div className="fixed top-4 right-4 z-50 space-y-2">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onRemove={removeToast} />
      ))}
    </div>
  )

  return { showToast, ToastContainer }
}
