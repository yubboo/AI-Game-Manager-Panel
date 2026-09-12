import fs from 'node:fs'

const bat = fs.readFileSync('AGMP-GitHub.bat', 'utf8')
const ps1Buffer = fs.readFileSync('push-agmp.ps1')
const ps1 = ps1Buffer.toString('utf8')
const failures = []

if (!/if\s+"%RC%"=="0"[\s\S]*pause/i.test(bat)) {
  failures.push('AGMP-GitHub.bat must keep the window open after a successful helper run')
}
if (!/\[ERROR\][\s\S]*pause/i.test(bat)) {
  failures.push('AGMP-GitHub.bat must keep the window open after a failed helper run')
}
if (!ps1.includes("1. [一键推送]  安全检查 > 提交 > Push  ← 推荐' -ForegroundColor Green")) {
  failures.push('push-agmp.ps1 must visually highlight menu option 1 as the recommended action')
}
if (!(ps1Buffer[0] === 0xef && ps1Buffer[1] === 0xbb && ps1Buffer[2] === 0xbf)) {
  failures.push('push-agmp.ps1 must remain UTF-8 BOM for Windows PowerShell 5.1')
}
const withoutCrlf = ps1Buffer.toString('binary').replaceAll('\r\n', '')
if (withoutCrlf.includes('\n')) failures.push('push-agmp.ps1 must remain CRLF-only')

if (failures.length) {
  console.error('AGMP GitHub Helper UX Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP GitHub Helper UX Gate PASS (option 1 highlighted; console held open after completion)')
