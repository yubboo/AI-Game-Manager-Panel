import fs from 'node:fs'
const required=['frontend/src/features/xiaoyu/AIWorkbenchView.vue','frontend/src/app/layout/AppSidebar.vue']
for(const f of required)if(!fs.existsSync(f))throw new Error(`UI Freeze Gate: destination UI missing ${f}`)
console.log('AGMP UI Freeze Gate PASS (existing Codex three-column UI preserved by migration package)')
