// Package xiaoyu is the Go-side integration boundary for 小鱼 (XiaoYu), AGMP's embedded intelligence core.
//
// 小鱼真正的 Brain / Planner / Workflow / Context / Audit execution lives in the Rust
// xiaoyu-core. Go only owns stable domain Tool adapters, approval/permission bridges,
// provider-facing contracts and the runtime bridge. Business logic must stay in its domain.
package xiaoyu
