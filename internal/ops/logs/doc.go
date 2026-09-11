// Package loghub 实现 AI Game Manager Panel 全平台唯一日志中心。
//
// 该包负责日志目录索引、真实行数统计、增量缓存、按需分页、正文筛选、导出与安全删除；
// 具体游戏只负责产生自己的日志，不允许再实现第二套历史日志管理页面。
package loghub
