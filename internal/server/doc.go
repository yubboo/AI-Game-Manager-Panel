// Package server 归拢所有“游戏服务器实例控制”相关公共业务。
//
// 当前真实代码集中在 workspace；实例生命周期、通用终端和 RCON 只有产生真实实现时
// 才增加对应文件/子包，禁止仅为路线图预建空目录。游戏专属规则仍放 internal/games。
package server
