# AGMP 0.1.91 Runtime Restore 验证记录

0.1.90 交付包不是网络代理故障，而是源码完整性故障：源码包缺少本 module 内三个 Runtime package，`go mod tidy` 才会错误尝试从 GitHub/Go Proxy 查找它们。

本版本恢复：

- `internal/platform/runtime`：统一 Process / stdio / Session / Run / Shell 边界；
- `internal/xiaoyu/runtime`：Go Host 到 XiaoYu Rust Runtime 的桥接；
- `internal/games/dst/runtime`：DST 运行态与进程管理；
- `runtime/README.md`：运行数据目录契约。

已执行：

```text
node scripts/common/check-*.mjs

go test ./internal/platform/runtime \
  ./internal/games/dst/runtime \
  ./internal/xiaoyu/runtime \
  ./internal/app \
  ./internal/bridge/httpapi \
  ./internal/deploy/environment \
  ./internal/xiaoyu/host
```

结果：PASS。
