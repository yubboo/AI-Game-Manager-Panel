// Package environment 提供 AI Game Manager Panel 首次启动所需的运行环境发现与初始化服务。
//
// 0.1.45 首先覆盖 SteamCMD、DST Dedicated Server 默认安装位置与 AI Game Manager Panel
// 运行目录初始化。0.1.50 允许用户持久化跳过首次环境初始化，并在主界面的
// “运行环境”页面稍后补齐。Steam OpenID 身份绑定与 SteamCMD/游戏登录凭据
// 保持分离，本包不保存 Steam 密码或把 Web 登录会话复用为游戏平台凭据。
package environment
