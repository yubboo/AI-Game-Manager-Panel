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
  const result = await dialog.showOpenDialog({ title: options?.title || '选择目录', defaultPath: options?.initial || undefined, properties: ['openDirectory', 'createDirectory'] })
  return result.canceled ? '' : (result.filePaths[0] || '')
})
ipcMain.handle('agmp:open-path', async (_event, target?: string) => (!target || !path.isAbsolute(target)) ? '路径无效' : shell.openPath(target))

let coreProcess: ChildProcess | null = null
let coreOrigin = ''
let isQuitting = false

function getFreePort(): Promise<number> { return new Promise((resolve, reject) => { const s = createServer(); s.unref(); s.once('error', reject); s.listen(0, '127.0.0.1', () => { const a = s.address(); if (!a || typeof a === 'string') return s.close(() => reject(new Error('无法分配本地 Core 端口'))); const p = a.port; s.close(e => e ? reject(e) : resolve(p)) }) }) }

function runtimePaths() {
  const base = isDev ? path.join(projectRoot, 'runtime') : path.join(app.getPath('userData'), 'runtime')
  const data = path.join(base, 'data'), workspace = path.join(base, 'workspace'), native = path.join(base, 'native')
  for (const dir of [data, workspace, native]) fs.mkdirSync(dir, { recursive: true })
  return { data, workspace, native }
}

function coreSpec(port: number) {
  const override = process.env.AGMP_ELECTRON_CORE?.trim()
  if (override) return { command: override, args: [], cwd: projectRoot }
  if (isDev) return { command: process.execPath, args: ['--experimental-strip-types', path.join(projectRoot, 'apps', 'server', 'src', 'main.ts')], cwd: projectRoot }
  const entry = path.join(process.resourcesPath, 'core', 'apps', 'server', 'src', 'main.js')
  return { command: process.execPath, args: [entry], cwd: path.join(process.resourcesPath, 'core') }
}

async function waitForCore(origin: string, timeoutMs = 30_000) { const started = Date.now(); let last = ''; while (Date.now() - started < timeoutMs) { if (coreProcess?.exitCode != null) throw new Error(`TypeScript Core 已退出，退出码 ${coreProcess.exitCode}`); try { const r = await fetch(`${origin}/api/v1/health`, { signal: AbortSignal.timeout(1500) }); if (r.ok) return; last = `HTTP ${r.status}` } catch (e) { last = e instanceof Error ? e.message : String(e) } await new Promise(r => setTimeout(r, 250)) } throw new Error(`等待 TypeScript Core 启动超时：${last || '未知错误'}`) }

async function startCore() {
  const port = await getFreePort(), paths = runtimePaths(); coreOrigin = `http://127.0.0.1:${port}`; const spec = coreSpec(port)
  const nativeBinary = isDev ? path.join(projectRoot, 'crates', 'target', 'release', process.platform === 'win32' ? 'agmp-native.exe' : 'agmp-native') : path.join(process.resourcesPath, 'native', process.platform === 'win32' ? 'agmp-native.exe' : 'agmp-native')
  coreProcess = spawn(spec.command, spec.args, { cwd: spec.cwd, env: { ...process.env, AGMP_PORT: String(port), AGMP_DATA: paths.data, AGMP_WORKSPACE: paths.workspace, AGMP_NATIVE_ROOT: paths.native, AGMP_NATIVE_RUNTIME: nativeBinary, AGMP_DESKTOP_FRAMEWORK: 'electron' }, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] })
  coreProcess.stdout?.on('data', c => process.stdout.write(`[AGMP TS Core] ${c}`)); coreProcess.stderr?.on('data', c => process.stderr.write(`[AGMP TS Core] ${c}`))
  await waitForCore(coreOrigin)
}
function stopCore() { const p = coreProcess; coreProcess = null; if (p?.exitCode === null) p.kill() }

async function createMainWindow() {
  const win = new BrowserWindow({ width: 1280, height: 840, minWidth: 960, minHeight: 640, show: false, title: 'AI游戏管理器面板 · Electron', backgroundColor: '#101311', webPreferences: { preload: path.join(currentDir, 'preload.cjs'), nodeIntegration: false, contextIsolation: true, sandbox: true, webSecurity: true, devTools: isDev } })
  win.webContents.setWindowOpenHandler(({ url }) => { if (/^https?:\/\//i.test(url)) void shell.openExternal(url); return { action: 'deny' } })
  win.webContents.on('will-navigate', (event, url) => { try { if (new URL(url).origin === coreOrigin) return } catch {} event.preventDefault() })
  win.once('ready-to-show', () => win.show()); await win.loadURL(coreOrigin); return win
}

function installMenu() { const template: MenuItemConstructorOptions[] = [
  { label: '文件', submenu: [{ label: '新建窗口', accelerator: 'Ctrl+Shift+N', click: () => { void createMainWindow() } }, { type: 'separator' }, { label: '退出', role: 'quit' }] },
  { label: '编辑', submenu: [{ label: '撤销', role: 'undo' }, { label: '重做', role: 'redo' }, { type: 'separator' }, { label: '剪切', role: 'cut' }, { label: '复制', role: 'copy' }, { label: '粘贴', role: 'paste' }, { label: '全选', role: 'selectAll' }] },
  { label: '视图', submenu: [{ label: '重新加载', role: 'reload' }, ...(isDev ? [{ label: '开发者工具', role: 'toggleDevTools' as const }] : []), { type: 'separator' }, { label: '实际大小', role: 'resetZoom' }, { label: '放大', role: 'zoomIn' }, { label: '缩小', role: 'zoomOut' }, { label: '全屏', role: 'togglefullscreen' }] },
  { label: '帮助', submenu: [{ label: '关于AI游戏管理器面板', click: () => { void dialog.showMessageBox({ type: 'info', title: '关于AI游戏管理器面板', message: 'AI游戏管理器面板', detail: `版本 ${app.getVersion()}\nTypeScript Agent + Rust Native\n\n配置什么厂商模型，小鱼就使用什么模型的真实能力。`, buttons: ['确定'] }) } }] },
]; Menu.setApplicationMenu(Menu.buildFromTemplate(template)) }

if (!app.requestSingleInstanceLock()) app.quit(); else {
  app.on('second-instance', () => { const w = BrowserWindow.getAllWindows()[0]; if (w) { if (w.isMinimized()) w.restore(); w.show(); w.focus() } })
  app.whenReady().then(async () => { installMenu(); session.defaultSession.setPermissionRequestHandler((_wc, _p, cb) => cb(false)); session.defaultSession.setPermissionCheckHandler(() => false); try { await startCore(); await createMainWindow() } catch (e) { console.error('[Electron] 启动失败:', e); stopCore(); app.exit(1) } })
}
app.on('window-all-closed', () => { if (process.platform !== 'darwin') app.quit() })
app.on('before-quit', () => { isQuitting = true; stopCore() })
void isQuitting
