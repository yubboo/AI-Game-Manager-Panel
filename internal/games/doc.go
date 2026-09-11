// Package games 是 AGMP 的游戏领域根。
//
// 只有已经开始实现真实适配逻辑的游戏才建立源码目录。planned 游戏统一保存在
// configs/games.json 和路线图中，禁止仅用 doc.go 预占目录；公共游戏契约归 common，
// DST 真实实现归 dst，Steam 平台关联的已实现适配归 steam。
package games
