# AI Game Manager Panel 0.1.37 正式构建热修复

## 问题

0.1.36 新增 `network` 工作区后，`WorkspaceSection` 已包含 `network`，但 `DstWorkspace.vue` 中的 `placeholderCopy` 仍使用“排除若干真实页面后，其余页面必须全部提供占位文案”的 `Record<Exclude<...>>` 类型。

因此正式构建执行：

```text
vue-tsc --noEmit && vite build
```

会在 `DstWorkspace.vue` 报 TS2741：`network` 缺失，导致双端构建在前端类型检查阶段终止。

## 修复

`placeholderCopy` 改为：

```ts
Partial<Record<WorkspaceSection, { title: string; description: string }>>
```

它只描述真正尚未实现的占位页面。`network`、`tokens`、`saves`、`install` 等真实页面不再需要进入占位映射，也不再依赖手工维护排除列表。

`activePlaceholder` 同时改为安全可空查找：

```ts
const activePlaceholder = computed(() => placeholderCopy[section.value] ?? null)
```

## 影响

- 修复 0.1.36 的 TS2741 正式构建失败。
- 不改变 0.1.36 的 DST 网络与 Shard 配置业务行为。
- 后续新增真实 Workspace 页面时，不会因为遗漏占位映射再次触发同类构建错误。
