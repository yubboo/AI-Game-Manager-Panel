/// <reference types="vite/client" />

interface Window {
  agmpElectron?: Readonly<{
    desktop: true
    framework: 'electron'
    electronVersion: string
    chromiumVersion: string
    nodeVersion: string
    platform: string
    selectDirectory: (title?: string, initial?: string) => Promise<string>
    openPath: (target: string) => Promise<string>
  }>
}
