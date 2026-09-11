// Package platformfiles 提供 AGMP 跨业务共享的本地文件基础能力。
//
// 这里只放路径根解析、原子替换、打开目录等 OS/文件系统原语；上传、备份、
// 在线编辑和日志导出属于上层业务域，不得反向塞回平台层。
package platformfiles
