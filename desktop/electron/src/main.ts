import { app, BrowserWindow, dialog, ipcMain, Menu, session, shell, type MenuItemConstructorOptions } from 'electron'
import { spawn, type ChildProcess } from 'node:child_process'
import { createServer } from 'node:net'
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const electronRoot = path.resolve(currentDir, '..')
const projectRoot = path.resolve(electronRoot, '..', '..')
const isDev = process.argv.includes('--agmp-dev') || !app.isPackaged

app.setAppUserModelId('com.aigamemanager.panel')

ipcMain.handle('agmp:select-directory', async (_event, options?: { title?: string; initial?: string }) => {
  const result = await dialog.showOpenDialog({
    title: options?.title || '选择目录',
    defaultPath: options?.initial || undefined,
    properties: ['openDirectory', 'createDirectory'],
  })
  return result.canceled ? '' : (result.filePaths[0] || '')
})

ipcMain.handle('agmp:open-path', async (_event, target?: string) => {
  if (!target || !path.isAbsolute(target)) return '路径无效'
  return shell.openPath(target)
})

let coreProcess: ChildProcess | null = null
let coreOrigin = ''
let isQuitting = false

function getFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = createServer()
    server.unref()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (!address || typeof address === 'string') {
        server.close()
        reject(new Error('无法分配 Electron 本地 Core 端口'))
        return
      }
      const port = address.port
      server.close(error => error ? reject(error) : resolve(port))
    })
  })
}

function prepareRuntimeRoot() {
  if (isDev) return projectRoot

  const runtimeRoot = path.join(app.getPath('userData'), 'runtime')
  const sourceConfigs = path.join(process.resourcesPath, 'defaults', 'configs')
  const targetConfigs = path.join(runtimeRoot, 'configs')
  fs.mkdirSync(targetConfigs, { recursive: true })

  if (fs.existsSync(sourceConfigs)) {
    for (const entry of fs.readdirSync(sourceConfigs, { withFileTypes: true })) {
      if (!entry.isFile() || !entry.name.endsWith('.json')) continue
      const source = path.join(sourceConfigs, entry.name)
      const target = path.join(targetConfigs, entry.name)
      if (!fs.existsSync(target)) fs.copyFileSync(source, target)
    }
  }

  return runtimeRoot
}

function coreCommand(port: number) {
  const listen = `127.0.0.1:${port}`
  const override = process.env.AGMP_ELECTRON_CORE?.trim()
  if (override) {
    return { command: override, args: ['--listen', listen], cwd: projectRoot }
  }

  if (isDev) {
    return { command: 'go', args: ['run', '-tags', 'agmp_dev_license', './cmd/aigame-manager-web', '--listen', listen], cwd: projectRoot }
  }

  const executable = process.platform === 'win32' ? 'AI-Game-Manager-Core.exe' : 'AI-Game-Manager-Core'
  return {
    command: path.join(process.resourcesPath, 'core', executable),
    args: ['--listen', listen],
    cwd: path.dirname(path.join(process.resourcesPath, 'core', executable)),
  }
}

async function waitForCore(origin: string, timeoutMs = 30_000) {
  const startedAt = Date.now()
  let lastError = ''
  while (Date.now() - startedAt < timeoutMs) {
    if (coreProcess?.exitCode !== null && coreProcess?.exitCode !== undefined) {
      throw new Error(`AI Game Manager Panel Core 已退出，退出码 ${coreProcess.exitCode}`)
    }
    try {
      const response = await fetch(`${origin}/api/v1/health`, { signal: AbortSignal.timeout(1500) })
      if (response.ok) return
      lastError = `HTTP ${response.status}`
    } catch (error) {
      lastError = error instanceof Error ? error.message : String(error)
    }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  throw new Error(`等待 AI Game Manager Panel Core 启动超时：${lastError || '未知错误'}`)
}

async function startCore() {
  const port = await getFreePort()
  coreOrigin = `http://127.0.0.1:${port}`
  const spec = coreCommand(port)
  const runtimeRoot = prepareRuntimeRoot()

  coreProcess = spawn(spec.command, spec.args, {
    cwd: spec.cwd,
    env: {
      ...process.env,
      AGMP_DESKTOP_FRAMEWORK: 'electron',
      AGMP_ROOT: runtimeRoot,
      AGMP_XIAOYU_RUNTIME: isDev
        ? (process.env.AGMP_XIAOYU_RUNTIME || '')
        : path.join(process.resourcesPath, 'core', 'AI-Game-Manager-XiaoYu.exe'),
    },
    windowsHide: true,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  coreProcess.stdout?.on('data', chunk => process.stdout.write(`[AI Game Manager Panel Core] ${chunk}`))
  coreProcess.stderr?.on('data', chunk => process.stderr.write(`[AI Game Manager Panel Core] ${chunk}`))
  coreProcess.once('error', error => {
    if (!isQuitting) console.error('[Electron] AI Game Manager Panel Core 启动失败:', error)
  })

  await waitForCore(coreOrigin)
}

function stopCore() {
  if (!coreProcess) return
  const child = coreProcess
  coreProcess = null
  if (child.exitCode === null) child.kill()
}

async function createMainWindow() {
  const preload = path.join(currentDir, 'preload.cjs')
  const window = new BrowserWindow({
    width: 1280,
    height: 840,
    minWidth: 960,
    minHeight: 640,
    show: false,
    title: 'AI游戏管理器面板 · Electron',
    backgroundColor: '#101311',
    webPreferences: {
      preload,
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true,
      webSecurity: true,
      devTools: isDev,
    },
  })

  window.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:\/\//i.test(url)) void shell.openExternal(url)
    return { action: 'deny' }
  })

  window.webContents.on('will-navigate', (event, url) => {
    try {
      const destination = new URL(url)
      if (destination.origin === coreOrigin) return
    } catch {
      // Invalid navigation is denied below.
    }
    event.preventDefault()
  })

  window.once('ready-to-show', () => window.show())
  await window.loadURL(coreOrigin)
  return window
}


function installChineseApplicationMenu() {
  const template: MenuItemConstructorOptions[] = [
    {
      label: '文件',
      submenu: [
        { label: '新建窗口', accelerator: 'Ctrl+Shift+N', click: () => { void createMainWindow() } },
        { type: 'separator' },
        { label: '退出', accelerator: 'Alt+F4', role: 'quit' },
      ],
    },
    {
      label: '编辑',
      submenu: [
        { label: '撤销', accelerator: 'Ctrl+Z', role: 'undo' },
        { label: '重做', accelerator: 'Ctrl+Y', role: 'redo' },
        { type: 'separator' },
        { label: '剪切', accelerator: 'Ctrl+X', role: 'cut' },
        { label: '复制', accelerator: 'Ctrl+C', role: 'copy' },
        { label: '粘贴', accelerator: 'Ctrl+V', role: 'paste' },
        { label: '删除', role: 'delete' },
        { type: 'separator' },
        { label: '全选', accelerator: 'Ctrl+A', role: 'selectAll' },
      ],
    },
    {
      label: '视图',
      submenu: [
        { label: '重新加载', accelerator: 'Ctrl+R', role: 'reload' },
        { label: '强制重新加载', accelerator: 'Ctrl+Shift+R', role: 'forceReload' },
        ...(isDev ? [{ label: '开发者工具', accelerator: 'F12', role: 'toggleDevTools' as const }] : []),
        { type: 'separator' },
        { label: '实际大小', accelerator: 'Ctrl+0', role: 'resetZoom' },
        { label: '放大', accelerator: 'Ctrl+Plus', role: 'zoomIn' },
        { label: '缩小', accelerator: 'Ctrl+-', role: 'zoomOut' },
        { type: 'separator' },
        { label: '全屏', accelerator: 'F11', role: 'togglefullscreen' },
      ],
    },
    {
      label: '窗口',
      submenu: [
        { label: '最小化', role: 'minimize' },
        { label: '关闭', accelerator: 'Ctrl+W', role: 'close' },
      ],
    },
    {
      label: '帮助',
      submenu: [
        {
          label: '关于AI游戏管理器面板',
          click: () => {
            void dialog.showMessageBox({
              type: 'info',
              title: '关于AI游戏管理器面板',
              message: 'AI游戏管理器面板',
              detail: `版本 ${app.getVersion()}\nGo Core + Vue 3 + Wails / Electron / Web\n\n游戏服务器统一管理工作台。`,
              buttons: ['确定'],
              noLink: true,
            })
          },
        },
      ],
    },
  ]
  Menu.setApplicationMenu(Menu.buildFromTemplate(template))
}

const lockAcquired = app.requestSingleInstanceLock()
if (!lockAcquired) {
  app.quit()
} else {
  app.on('second-instance', () => {
    const window = BrowserWindow.getAllWindows()[0]
    if (!window) return
    if (window.isMinimized()) window.restore()
    window.show()
    window.focus()
  })

  app.whenReady().then(async () => {
    installChineseApplicationMenu()
    session.defaultSession.setPermissionRequestHandler((_webContents, _permission, callback) => callback(false))
    session.defaultSession.setPermissionCheckHandler(() => false)

    try {
      await startCore()
      await createMainWindow()
    } catch (error) {
      console.error('[Electron] 启动失败:', error)
      stopCore()
      app.exit(1)
    }
  })
}

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit()
})

app.on('before-quit', () => {
  isQuitting = true
  stopCore()
})
