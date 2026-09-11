// Package platformruntime provides AGMP's single shared process/stdio runtime boundary.
//
// It owns executable launch, stdin/stdout/stderr streaming, PID/state/exit
// snapshots, bounded command execution, output subscriptions/history, process
// termination and platform-specific process behavior. Game and system domains
// keep their own command syntax, readiness parsing and business state, but must
// not copy OS process/stdio mechanics. PTY/ConPTY may extend this package later;
// it must never introduce a second Process Core.
package platformruntime
