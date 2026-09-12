import fs from 'node:fs'
const required=['apps/server/src/app.ts','apps/server/src/main.ts','desktop/electron/src/main.ts','desktop/electron/src/preload.cts','desktop/electron/tsconfig.json','packages/agent/src/index.ts','packages/agent/src/run-manager.ts','packages/model/src/store.ts','packages/tools/src/approval.ts','packages/tools/src/native.ts','packages/minecraft/src/index.ts','packages/instances/src/index.ts','skills/minecraft-server/SKILL.md','crates/native-runtime/src/capability.rs','crates/native-runtime/src/managed_java.rs','crates/native-runtime/src/network.rs']
for(const f of required)if(!fs.existsSync(f))throw new Error(`missing ${f}`)
const agent=fs.readFileSync('packages/agent/src/index.ts','utf8'), mc=fs.readFileSync('packages/minecraft/src/index.ts','utf8'), model=fs.readFileSync('packages/model/src/store.ts','utf8'), cap=fs.readFileSync('crates/native-runtime/src/capability.rs','utf8'), skill=fs.readFileSync('skills/minecraft-server/SKILL.md','utf8')
for(const m of ['toolCalls','model.generate','ToolRegistry'])if(!agent.includes(m))throw new Error(`Agent Loop regression: ${m}`)
for(const m of ['minecraft.version.resolve','minecraft.paper.resolve','minecraft.fabric.resolve','minecraft.modrinth.search','minecraft.modrinth.resolve','minecraft.server.verify','minecraft.instance.create'])if(!mc.includes(m))throw new Error(`Minecraft Agent tool missing: ${m}`)
for(const m of ['defaultProvider','verifiedAt','agmp_probe'])if(!model.includes(m))throw new Error(`Model brain verification regression: ${m}`)
for(const m of ['remaining','fingerprint','capability run mismatch','capability tool mismatch'])if(!cap.includes(m))throw new Error(`Rust capability regression: ${m}`)
if(!skill.includes('vendor model selected by the user')||!skill.includes('remains the brain'))throw new Error('Skill must preserve vendor-model intelligence rule')
if(mc.includes('deployMinecraft(')||agent.includes('deployMinecraft('))throw new Error('Hard-coded deployMinecraft pipeline is forbidden; decisions belong to configured model')
console.log('AGMP Architecture Gate PASS (TS Agent Loop · vendor model brain · Rust Native · Agent-first Minecraft)')
