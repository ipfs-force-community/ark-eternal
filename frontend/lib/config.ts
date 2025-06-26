export const config = {
  api: {
    baseUrl: process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:12345",
    endpoints: {
      files: "/files",
      upload: "/upload",
      download: "/download",
    },
  },
  app: {
    name: "Ark Eternal",
    version: "1.0.0",
  },
} as const

export type Config = typeof config
