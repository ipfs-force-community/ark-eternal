import { config } from "./config"
import type { FileInfo, UploadRequest } from "@/types"

class ApiService {
  private baseUrl: string

  constructor() {
    this.baseUrl = config.api.baseUrl
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`

    try {
      const response = await fetch(url, {
        headers: {
          "Content-Type": "application/json",
          ...options.headers,
        },
        ...options,
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      return await response.json()
    } catch (error) {
      console.error(`API request failed: ${url}`, error)
      throw error
    }
  }

  async getFiles(userAddress: string): Promise<FileInfo[]> {
    return this.request<FileInfo[]>(`${config.api.endpoints.files}?user_address=${encodeURIComponent(userAddress)}`)
  }

  async uploadFile(data: UploadRequest): Promise<any> {
    return this.request(config.api.endpoints.upload, {
      method: "POST",
      body: JSON.stringify(data),
    })
  }

  async downloadFile(userAddress: string, fileName: string): Promise<string> {
    const url = `${this.baseUrl}${config.api.endpoints.download}?user_address=${encodeURIComponent(userAddress)}&file_name=${encodeURIComponent(fileName)}`

    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`Download failed: ${response.status}`)
    }

    return response.text()
  }

  async downloadFileByCID(cid: string): Promise<string> {
    // Construct the URL using the CID as a query parameter
    const url = `${this.baseUrl}/${encodeURIComponent(cid)}`
  
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`Download failed: ${response.status}`)
    }
  
    return url
  }
}

export const apiService = new ApiService()
