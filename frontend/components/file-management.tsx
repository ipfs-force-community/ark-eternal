"use client"

import { useState, useEffect, useCallback } from "react"
import { useWallet } from "@/hooks/use-wallet"
import { useToast } from "@/components/toast"
import { apiService } from "@/lib/api"
import type { FileInfo } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Save, FileText, ExternalLink } from "lucide-react"

export function FileManagement() {
  const [files, setFiles] = useState<FileInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [pageUrl, setPageUrl] = useState("")
  const [fileName, setFileName] = useState("")

  const { isConnected, address } = useWallet()
  const { showToast } = useToast()

  const fetchFiles = useCallback(async () => {
    if (!address) return

    setLoading(true)
    try {
      const fileList = await apiService.getFiles(address)
      setFiles(fileList)
    } catch {
      showToast("Failed to load files", "error")
    } finally {
      setLoading(false)
    }
  }, [address, showToast])

  const handleUpload = async () => {
    if (!address || !pageUrl.trim() || !fileName.trim()) {
      showToast("Please fill in all fields", "error")
      return
    }

    setUploading(true)
    try {
      await apiService.uploadFile({
        user_address: address,
        file_name: fileName.trim(),
        resource_url: pageUrl.trim(),
      })

      showToast("File uploaded successfully", "success")
      setPageUrl("")
      setFileName("")
      await fetchFiles()
    } catch {
      showToast("Upload failed", "error")
    } finally {
      setUploading(false)
    }
  }

  const handleFileClick = async (file: FileInfo) => {
    if (!address) return

    try {
      const htmlContent = await apiService.downloadFile(address, file.file_name)
      const newWindow = window.open("", "_blank")
      if (newWindow) {
        newWindow.document.write(htmlContent)
        newWindow.document.close()
      } else {
        showToast("Unable to open new window", "error")
      }
    } catch {
      showToast("Failed to download file", "error")
    }
  }

  const handleDownload = async (cid: string) => {
    if (!cid) return

    try {
      const url = await apiService.downloadFileByCID(cid)
      window.open(url, "_blank")
    } catch {
      showToast("Failed to download file", "error")
    }
  }

  useEffect(() => {
    if (isConnected && address) {
      fetchFiles()
    }
  }, [isConnected, address, fetchFiles])

  const getStatusColor = (status: FileInfo["status"]) => {
    switch (status) {
      case "completed":
        return "bg-green-100 text-green-800"
      case "pending":
        return "bg-yellow-100 text-yellow-800"
      case "failed":
        return "bg-red-100 text-red-800"
      default:
        return "bg-gray-100 text-gray-800"
    }
  }

  const getStatusText = (status: FileInfo["status"]) => {
    switch (status) {
      case "completed":
        return "Completed"
      case "pending":
        return "Pending"
      case "failed":
        return "Failed"
      default:
        return "Unknown"
    }
  }

  // Helper function to truncate the root string
  const truncateRoot = (root: string, startLength: number = 6, endLength: number = 6): string => {
    if (!root || root.length <= startLength + endLength) {
      return root; // If the root is short, return it as is
    }
    return `${root.slice(0, startLength)}...${root.slice(-endLength)}`;
  };


  if (!isConnected) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="text-center py-8">
          <p className="text-gray-500">Please connect your wallet to manage files</p>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <h2 className="text-lg font-medium text-gray-800 mb-4">File Management</h2>

      {/* Upload Form */}
      <div className="space-y-4 mb-6">
        <div>
          <Label htmlFor="page-url">Page URL</Label>
          <Input
            id="page-url"
            type="url"
            value={pageUrl}
            onChange={(e) => setPageUrl(e.target.value)}
            placeholder="Enter the page URL to save"
          />
        </div>
        <div>
          <Label htmlFor="file-name">File Name</Label>
          <Input
            id="file-name"
            value={fileName}
            onChange={(e) => setFileName(e.target.value)}
            placeholder="Enter the file name to save"
          />
        </div>
        <Button onClick={handleUpload} disabled={uploading || !pageUrl.trim() || !fileName.trim()} className="w-full">
          <Save className="w-4 h-4 mr-2" />
          {uploading ? "Saving..." : "Save Page"}
        </Button>
      </div>

      {/* File List */}
      <div className="overflow-x-auto">
        {loading ? (
          <div className="text-center py-8">
            <p className="text-gray-500">Loading files...</p>
          </div>
        ) : (
          <table className="min-w-full">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="py-3 px-4 text-left text-sm font-medium text-gray-500">File Name</th>
                <th className="py-3 px-4 text-left text-sm font-medium text-gray-500">Root</th>
                <th className="py-3 px-4 text-left text-sm font-medium text-gray-500">Size</th>
                <th className="py-3 px-4 text-left text-sm font-medium text-gray-500">Upload Time</th>
                <th className="py-3 px-4 text-left text-sm font-medium text-gray-500">Status</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-gray-500">
                    Loading files...
                  </td>
                </tr>
              ) : files === null ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-gray-500">
                    No files found
                  </td>
                </tr>
              ) : (
                files.map((file, index) => (
                  <tr key={index} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="py-3 px-4 text-sm">
                      <button
                        onClick={() => handleFileClick(file)}
                        className="text-gray-700 hover:text-primary flex items-center"
                      >
                        <FileText className="w-4 h-4 mr-2" />
                        {file.file_name}
                        <ExternalLink className="w-3 h-3 ml-1" />
                      </button>
                    </td>
                    <td className="py-3 px-4 text-sm">
                      <a
                        href="#"
                        onClick={(e) => {
                          e.preventDefault();
                          handleDownload(file.root);
                        }}
                        className="text-gray-700 hover:text-blue-500 hover:underline flex items-center transition-colors duration-200"
                      >
                        {truncateRoot(file.root)}
                      </a>
                    </td>
                    <td className="py-3 px-4 text-sm text-gray-600">{file.size}</td>
                    <td className="py-3 px-4 text-sm text-gray-600">{file.upload_time}</td>
                    <td className="py-3 px-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                          file.status
                        )}`}
                      >
                        {getStatusText(file.status)}
                      </span>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
