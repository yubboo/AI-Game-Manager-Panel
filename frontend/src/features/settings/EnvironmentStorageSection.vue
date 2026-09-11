<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppIcon from '../../shared/components/AppIcon.vue'
import { backend } from '../../shared/api/backend'
import type { EnvironmentGameRuntimeProfile, EnvironmentRuntimeCatalog, EnvironmentRuntimeKind, EnvironmentSetupStatus } from '../../shared/types/backend'

const setup = ref<EnvironmentSetupStatus | null>(null)
const catalog = ref<EnvironmentRuntimeCatalog | null>(null)
const profiles = ref<EnvironmentGameRuntimeProfile[]>([])
const busy = ref(false)
const message = ref('')
const migrationRemoveSource = ref(false)
const javaTargetRoots = reactive<Record<number,string>>({ 8:'',17:'',21:'',25:'' })
const external = reactive({ kind: 'java' as EnvironmentRuntimeKind, executable: '', major: 21, setDefault: false })

const form = reactive({ steamcmdRoot:'', steamcmdPath:'', gameLibraryRoot:'', instanceConfigRoot:'', gameSaveRoot:'', cacheRoot:'' })
const desktopBrowseAvailable = computed(() => backend.mode() === 'desktop')
const serverPlatformLabel = computed(() => `${catalog.value?.platform || setup.value?.platform || '…'} / ${catalog.value?.architecture || setup.value?.architecture || '…'}`)

function fill(status: EnvironmentSetupStatus) {
  setup.value = status
  form.steamcmdRoot = status.paths.steamcmdRoot || status.steamcmd.root || ''
  form.steamcmdPath = status.paths.steamcmdPath || status.steamcmd.path || ''
  form.gameLibraryRoot = status.paths.gameLibraryRoot || status.defaultInstallRoot || ''
  form.instanceConfigRoot = status.paths.instanceConfigRoot || ''
  form.gameSaveRoot = status.paths.gameSaveRoot || ''
  form.cacheRoot = status.paths.cacheRoot || ''
}
function errorText(error: unknown) { return error instanceof Error ? error.message : String(error) }
function javaRuntime(major:number) { return catalog.value?.runtimes?.find(item => item.kind==='java' && item.major===major) }

async function refresh() {
  busy.value = true; message.value = ''
  try {
    const [status, runtimeCatalog, gameProfiles] = await Promise.all([backend.environmentSetupStatus(), backend.environmentRuntimeCatalog(), backend.environmentGameRuntimeProfiles()])
    fill(status)
    catalog.value = {
      ...runtimeCatalog,
      javaMajors: runtimeCatalog.javaMajors?.length ? runtimeCatalog.javaMajors : [8, 17, 21, 25],
      runtimes: runtimeCatalog.runtimes ?? [],
      warnings: runtimeCatalog.warnings ?? [],
    }
    profiles.value = gameProfiles ?? []
    message.value = '运行环境状态已重新检测。'
  } catch (error) { message.value = errorText(error) } finally { busy.value = false }
}
async function savePaths() {
  busy.value=true; message.value=''
  try { fill(await backend.updateEnvironmentPaths({ ...form })); message.value='运行环境与存储路径已保存。' } catch(error){message.value=errorText(error)} finally{busy.value=false}
}
async function browse(key: keyof typeof form, title: string) {
  if (!desktopBrowseAvailable.value) { message.value='Web 浏览器不能读取服务器本地目录，请输入 AGMP 服务端路径。'; return }
  const selected=await backend.selectDirectory(title,form[key]); if(selected) form[key]=selected
}
async function useSelectedSteamCMDRoot() {
  await browse('steamcmdRoot','选择已有 SteamCMD 根目录'); if(!form.steamcmdRoot)return
  const windows=(catalog.value?.platform || setup.value?.platform)==='windows'
  form.steamcmdPath=form.steamcmdRoot.replace(/[\\/]+$/,'')+(windows?'\\steamcmd.exe':'/steamcmd.sh')
  message.value='已按 AGMP 服务端平台生成 SteamCMD 路径，请保存或自动检测。'
}
async function installSteamCMD() {
  busy.value=true;message.value=''
  try { fill(await backend.installSteamCMD({targetRoot:form.steamcmdRoot})); await refresh(); message.value='SteamCMD 已安装并登记到 Runtime Registry。' } catch(error){message.value=errorText(error)} finally{busy.value=false}
}
async function installJava(major:number) {
  busy.value=true;message.value=''
  try { const value=await backend.installJavaRuntime({major,targetRoot:javaTargetRoots[major]||undefined,setDefault:major===21}); await refresh(); message.value=`${value.name} 已安装并通过 java -version 验证。` } catch(error){message.value=errorText(error)} finally{busy.value=false}
}
async function registerExternal() {
  busy.value=true;message.value=''
  try { const value=await backend.registerEnvironmentRuntime({kind:external.kind,executable:external.executable,major:external.kind==='java'?external.major:undefined,setDefault:external.setDefault}); external.executable=''; await refresh(); message.value=`已登记外部 Runtime：${value.name}` } catch(error){message.value=errorText(error)} finally{busy.value=false}
}
async function setDefault(id:string) { busy.value=true; try{await backend.setDefaultEnvironmentRuntime({id});await refresh()}catch(error){message.value=errorText(error)}finally{busy.value=false} }
async function removeRuntime(id:string) { if(!window.confirm('确认移除此 Runtime？AGMP 管理的 Runtime 会删除自身目录；外部 Runtime 只移除登记，不删除文件。'))return; busy.value=true;try{await backend.removeEnvironmentRuntime(id);await refresh()}catch(error){message.value=errorText(error)}finally{busy.value=false} }
async function installPrerequisite(id:string){busy.value=true;try{await backend.installEnvironmentSystemPrerequisite({id});await refresh();message.value='Linux 系统前置依赖安装完成并重新检测。'}catch(error){message.value=errorText(error)}finally{busy.value=false}}
async function openLocalDirectory(target:string,label:string){if(!target)return;if(backend.mode()!=='desktop'){message.value=`Web 模式请在 AGMP 服务端打开${label}：${target}`;return}const result=await backend.openLocalPath(target);message.value=result?`${label}打开结果：${result}`:`已请求打开${label}。`}
async function migrateSteamCMD(){const target=await backend.selectDirectory('选择新的 SteamCMD 目录',form.steamcmdRoot);if(!target)return;if(!window.confirm(`确认迁移 SteamCMD 到：\n${target}`))return;busy.value=true;try{const result=await backend.migrateSteamCMD({targetRoot:target,removeSource:migrationRemoveSource.value});await refresh();message.value=`${result.message}；复制 ${result.copiedFiles} 个文件。`}catch(error){message.value=errorText(error)}finally{busy.value=false}}
async function migrateGameLibrary(){const target=await backend.selectDirectory('选择新的游戏服务器库目录',form.gameLibraryRoot);if(!target)return;if(!window.confirm(`确认迁移游戏服务器库到：\n${target}`))return;busy.value=true;try{const result=await backend.migrateGameLibrary({targetRoot:target,removeSource:migrationRemoveSource.value});await refresh();message.value=`${result.message}；复制 ${result.copiedFiles} 个文件。`}catch(error){message.value=errorText(error)}finally{busy.value=false}}

onMounted(()=>{void refresh()})
</script>

<template>
<section class="settings-section settings-section--environment">
  <div class="settings-section__intro"><div class="settings-section__icon"><AppIcon name="environment" /></div><div><strong>运行环境与存储</strong><span>Java、SteamCMD 与游戏运行环境统一由 AGMP 服务端 Runtime Manager 管理。浏览器所在电脑的操作系统不会影响检测结果。</span></div></div>

  <div class="environment-health-grid">
    <article class="feature-card"><span class="feature-title">AGMP 服务端</span><p class="feature-description">Runtime 判断只读取真正运行 AGMP Core 的系统。</p><strong>{{ serverPlatformLabel }}</strong></article>
    <article class="feature-card"><span class="feature-title">Runtime Root</span><p class="feature-description">AGMP 自己安装的 Java/SteamCMD 默认放在这里。</p><code>{{ catalog?.runtimeRoot || setup?.paths.runtimeRoot || '读取中…' }}</code></article>
    <article class="feature-card"><span class="feature-title">游戏服务器库</span><p class="feature-description">游戏程序与 AGMP 程序文件分离。</p><code>{{ setup?.paths.gameLibraryRoot || '读取中…' }}</code></article>
  </div>

  <div class="settings-subsection">
    <div class="settings-subsection__heading"><strong>Java 多版本 Runtime</strong><span>Java 8 / 17 / 21 / 25 可并存。Minecraft Adapter 以后按服务端版本解析需要的 Java，不修改系统 PATH。</span></div>
    <div class="environment-health-grid">
      <article v-for="major in (catalog?.javaMajors || [8,17,21,25])" :key="major" class="feature-card">
        <span class="feature-title">Java {{ major }}</span>
        <p class="feature-description">{{ javaRuntime(major) ? `${javaRuntime(major)?.source} · ${javaRuntime(major)?.version}` : '未登记' }}</p>
        <code v-if="javaRuntime(major)">{{ javaRuntime(major)?.executable }}</code>
        <input v-model.trim="javaTargetRoots[major]" :placeholder="`自定义安装目录（可选）`" />
        <div class="inline-actions">
          <button class="btn btn--primary" type="button" :disabled="busy" @click="installJava(major)">{{ javaRuntime(major) ? '重新安装/修复' : '安装' }}</button>
          <button v-if="javaRuntime(major) && !javaRuntime(major)?.default" class="btn btn--secondary" type="button" :disabled="busy" @click="setDefault(javaRuntime(major)!.id)">设为默认</button>
          <button v-if="javaRuntime(major)" class="btn btn--secondary" type="button" :disabled="busy" @click="removeRuntime(javaRuntime(major)!.id)">移除</button>
        </div>
      </article>
    </div>
  </div>

  <div class="settings-subsection">
    <div class="settings-subsection__heading"><strong>登记已有 Runtime</strong><span>可复用系统或其他磁盘已经存在的 Java/SteamCMD。外部 Runtime 被移除时 AGMP 只删除登记，不删除你的文件。</span></div>
    <div class="path-grid">
      <label class="field-block"><span class="feature-title">类型</span><select v-model="external.kind"><option value="java">Java</option><option value="steamcmd">SteamCMD</option></select></label>
      <label v-if="external.kind==='java'" class="field-block"><span class="feature-title">期望 Java 主版本</span><select v-model.number="external.major"><option :value="8">8</option><option :value="17">17</option><option :value="21">21</option><option :value="25">25</option></select></label>
      <label class="field-block"><span class="feature-title">可执行文件</span><input v-model.trim="external.executable" :placeholder="external.kind==='java' ? '.../bin/java 或 java.exe' : '.../steamcmd.sh 或 steamcmd.exe'" /></label>
      <label class="checkbox-inline"><input v-model="external.setDefault" type="checkbox" />登记后设为默认</label>
    </div>
    <button class="btn btn--secondary" type="button" :disabled="busy || !external.executable" @click="registerExternal">验证并登记</button>
  </div>

  <div class="settings-subsection">
    <div class="settings-subsection__heading"><strong>SteamCMD</strong><span>DST 等 Steam 游戏按需使用；Minecraft 不要求安装 SteamCMD。Windows/Linux 均支持官方安装。</span></div>
    <div class="path-grid">
      <label class="field-block"><span class="feature-title">SteamCMD 根目录</span><div class="path-input-row"><input v-model.trim="form.steamcmdRoot" /><button class="btn btn--secondary" type="button" @click="browse('steamcmdRoot','选择 SteamCMD 根目录')">浏览</button></div></label>
      <label class="field-block"><span class="feature-title">SteamCMD 可执行文件</span><input v-model.trim="form.steamcmdPath" /></label>
    </div>
    <div class="inline-actions"><button class="btn btn--secondary" type="button" :disabled="busy" @click="refresh">自动检测</button><button class="btn btn--secondary" type="button" :disabled="busy || !desktopBrowseAvailable" @click="useSelectedSteamCMDRoot">选择已有目录</button><button class="btn btn--primary" type="button" :disabled="busy" @click="installSteamCMD">安装/修复 SteamCMD</button><button class="btn btn--secondary" type="button" :disabled="busy || !desktopBrowseAvailable" @click="migrateSteamCMD">迁移</button></div>
  </div>

  <div class="settings-subsection">
    <div class="settings-subsection__heading"><strong>游戏运行环境检查</strong><span>每个游戏只声明自己的依赖，不再存在“全局必须安装 SteamCMD”。</span></div>
    <article v-for="profile in profiles" :key="profile.gameId" class="feature-card">
      <span class="feature-title">{{ profile.name }}</span><strong :class="profile.ready ? 'status-success' : 'status-warning'">{{ profile.ready ? '运行环境已就绪' : '缺少运行环境' }}</strong>
      <p v-for="req in profile.requirements" :key="req.id" class="feature-description">{{ req.satisfied ? '✓' : '!' }} {{ req.message }}</p>
      <div v-for="item in profile.systemPrerequisites" :key="item.id" class="settings-actions-row"><span>{{ item.detected ? '✓' : '!' }} {{ item.name }} · {{ item.message }}</span><button v-if="!item.detected && item.installable" class="btn btn--secondary" type="button" :disabled="busy" @click="installPrerequisite(item.id)">安装系统依赖</button></div>
    </article>
  </div>

  <div class="settings-subsection">
    <div class="settings-subsection__heading"><strong>存储路径</strong><span>实例配置、存档、游戏服务器库和缓存与 Runtime Registry 分离。</span></div>
    <div class="path-grid">
      <label class="field-block"><span class="feature-title">游戏服务器库</span><input v-model.trim="form.gameLibraryRoot" /></label>
      <label class="field-block"><span class="feature-title">实例配置目录</span><input v-model.trim="form.instanceConfigRoot" /></label>
      <label class="field-block"><span class="feature-title">游戏存档根目录（可选）</span><input v-model.trim="form.gameSaveRoot" /></label>
      <label class="field-block"><span class="feature-title">缓存目录</span><input v-model.trim="form.cacheRoot" /></label>
    </div>
    <div class="settings-actions-row"><label class="checkbox-inline"><input v-model="migrationRemoveSource" type="checkbox" />迁移成功后删除旧目录</label><div class="inline-actions"><button class="btn btn--secondary" type="button" :disabled="busy || !desktopBrowseAvailable" @click="migrateGameLibrary">迁移游戏服务器库</button><button class="btn btn--secondary" type="button" :disabled="!setup?.paths.gameLibraryRoot || !desktopBrowseAvailable" @click="openLocalDirectory(setup?.paths.gameLibraryRoot || '', '游戏服务器库')">打开目录</button></div></div>
  </div>

  <div v-for="warning in [...(setup?.warnings || []), ...(catalog?.warnings || [])]" :key="warning" class="notice notice--warning">{{ warning }}</div>
  <div class="settings-footer settings-footer--sticky"><span class="save-message" :class="{active:message}">{{ message || '所有 Runtime 操作均作用于 AGMP 服务端，而不是当前浏览器。' }}</span><button class="btn btn--primary" type="button" :disabled="busy" @click="savePaths">{{ busy ? '处理中…' : '保存运行环境与存储' }}</button></div>
</section>
</template>
