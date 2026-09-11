# AGMP 0.1.92 First-Screen / Refresh Validation

## 回归目标

1. XiaoYu 未配置 Brain 时，黄色/中性配置提示只按内容高度显示，不得撑满中央工作区。
2. 已登录 Web 页面按 F5/Ctrl+R 后，不得短暂显示登录卡或 Bootstrap 页面。
3. Dark/Light 模式刷新时，首帧背景必须与刷新前主题一致，不得出现白闪/黑闪。
4. Session 过期时必须仍然进入登录页，不能因为“无感刷新”绕过认证。

## 自动检查

- XiaoYu Harness Gate 新增工作台布局防回归标记检查。
- Web Session Security Gate 新增 Silent Session Restore、Shell-first hydration、Prepaint Theme Restore 检查。
