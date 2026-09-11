# runtime

此目录只保留项目运行时目录说明文件。实际运行数据不会进入源码仓库。

运行后可能生成：

- `data/`：应用配置、账号与本地状态。
- `log/`：运行日志。
- `backups/`：备份。
- `instances/`：游戏服务器实例状态。
- `plugins/`、`cache/`、`temp/`、`exports/`：运行期扩展与缓存。

## 安全规则

除本 `README.md` 外，`runtime/` 下的内容均属于本机运行数据，默认被 `.gitignore` 和 GitHub Safety Gate 排除。不要把许可证激活文件、账号数据、服务器 Token、存档或私钥提交到 GitHub。

升级 AGMP 时应覆盖程序文件而不是删除此目录；卸载程序默认也应保留用户运行数据，除非用户明确选择删除。
