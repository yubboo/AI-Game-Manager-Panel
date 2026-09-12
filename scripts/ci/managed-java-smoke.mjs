import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import readline from 'node:readline'
const binary=process.argv[2]
const major=Number(process.argv[3]||21)
if(!binary)throw new Error('native binary path required')
const root=fs.mkdtempSync(path.join(os.tmpdir(),'agmp-java-workspace-'))
const managed=fs.mkdtempSync(path.join(os.tmpdir(),'agmp-java-managed-'))
const child=spawn(binary,['--root',root,'--managed-root',managed],{stdio:['pipe','pipe','inherit']})
const rl=readline.createInterface({input:child.stdout})
let nextId=1;const pending=new Map()
rl.on('line',line=>{const msg=JSON.parse(line);const p=pending.get(msg.id);if(!p)return;pending.delete(msg.id);msg.error?p.reject(new Error(msg.error.message)):p.resolve(msg.result)})
function call(method,params={}){const id=nextId++;return new Promise((resolve,reject)=>{pending.set(id,{resolve,reject});child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n')})}
try{
 const lease=await call('capability/issue',{runId:'ci-managed-java',tool:'system.java.ensure',approvalHash:'ci-approved',scope:'runtime.java.ensure',payload:{major},ttlMs:120000})
 const info=await call('environment/javaEnsure',{major,capabilityLeaseId:lease.id,capabilityScope:lease.scope,runId:'ci-managed-java',tool:'system.java.ensure'})
 if(!info.available||Number(info.major)!==major)throw new Error(`managed Java mismatch: ${JSON.stringify(info)}`)
 console.log(`Managed Java smoke PASS: ${info.version} @ ${info.executable}`)
}finally{child.kill();rl.close();fs.rmSync(root,{recursive:true,force:true});fs.rmSync(managed,{recursive:true,force:true})}
