import { contextBridge, ipcRenderer } from 'electron'

contextBridge.exposeInMainWorld('agmpElectron', Object.freeze({
  desktop: true,
  framework: 'electron',
  electronVersion: process.versions.electron,
  chromiumVersion: process.versions.chrome,
  nodeVersion: process.versions.node,
  platform: process.platform,
  selectDirectory: (title?: string, initial?: string) => ipcRenderer.invoke('agmp:select-directory', { title, initial }) as Promise<string>,
  openPath: (target: string) => ipcRenderer.invoke('agmp:open-path', target) as Promise<string>,
}))
