import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
const root=path.resolve('.tmp-test/packages')
const files=[]
function walk(dir){for(const e of fs.readdirSync(dir,{withFileTypes:true})){const p=path.join(dir,e.name);if(e.isDirectory())walk(p);else if(e.name.endsWith('.test.js'))files.push(p)}}
walk(root)
if(!files.length)throw new Error('no compiled core tests found')
const result=spawnSync(process.execPath,['--test',...files],{stdio:'inherit'})
process.exit(result.status??1)
