import fs from 'node:fs'; import path from 'node:path'
const root=process.cwd(), bad=[]; const skip=new Set(['.git','node_modules','.tmp-test','dist','target','runtime'])
function walk(dir){for(const e of fs.readdirSync(dir,{withFileTypes:true})){if(skip.has(e.name))continue;const p=path.join(dir,e.name),r=path.relative(root,p).replaceAll('\\','/');if(e.isDirectory())walk(p);else if(e.name.endsWith('.go')||['go.mod','go.sum','wails.json'].includes(e.name))bad.push(r)}}
walk(root); for(const d of ['cmd','internal','rust'])if(fs.existsSync(path.join(root,d)))bad.push(`${d}/ (legacy core directory)`)
if(bad.length){console.error('AGMP No-Go Gate FAIL');bad.forEach(x=>console.error(' - '+x));process.exit(1)} console.log('AGMP No-Go Gate PASS (TypeScript Agent + Rust Native; Go core = 0)')
