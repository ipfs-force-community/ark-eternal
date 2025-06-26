export interface FileInfo {
  file_name: string
  size: string
  upload_time: string
  status: "completed" | "pending" | "failed"
}

export interface UploadRequest {
  user_address: string
  file_name: string
  resource_url: string
}

export interface ProofSet {
  id: string
  name: string
  created_at: string
  status: "active" | "pending" | "failed"
  file_count: number
}

export interface WalletState {
  isConnected: boolean
  address: string | null
  isConnecting: boolean
}
