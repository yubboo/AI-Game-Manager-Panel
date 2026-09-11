use xiaoyu_protocol::{ApprovalMode, JsonRpcRequest, JsonRpcResponse, RiskLevel};
use xiaoyu_core::{Runtime, RUNTIME_VERSION};
use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use serde_json::{json, Value};
use std::io::{self, BufRead, Write};
use std::path::PathBuf;

#[derive(Debug, Parser)]
#[command(name = "xiaoyu", version = RUNTIME_VERSION, about = "小鱼 · AGMP Intelligence Core / CLI")]
struct Cli {
    #[arg(long, global = true)]
    root: Option<PathBuf>,
    #[command(subcommand)]
    command: Command,
}

#[derive(Debug, Subcommand)]
enum Command {
    Doctor {
        #[arg(long)]
        json: bool,
    },
    /// Compatibility/introspection command. Executable Tools live in AGMP Go
    /// domains, therefore XiaoYu Core intentionally reports no local Tools.
    Tools {
        #[arg(long)]
        json: bool,
    },
    Rpc,
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    let root = cli
        .root
        .unwrap_or(std::env::current_dir().context("无法读取当前工作目录")?);
    let runtime = Runtime::new(root);

    match cli.command {
        Command::Doctor { json } => print_value(json, &runtime.status()),
        Command::Tools { json } => print_value(json, &runtime.tools()),
        Command::Rpc => serve_rpc(runtime),
    }
}

fn print_value<T: serde::Serialize + std::fmt::Debug>(as_json: bool, value: &T) -> Result<()> {
    if as_json {
        println!("{}", serde_json::to_string(value)?);
    } else {
        println!("{value:#?}");
    }
    Ok(())
}

fn serve_rpc(runtime: Runtime) -> Result<()> {
    let stdin = io::stdin();
    let mut stdout = io::stdout().lock();
    for line in stdin.lock().lines() {
        let line = line?;
        if line.trim().is_empty() {
            continue;
        }
        let response = match serde_json::from_str::<JsonRpcRequest>(&line) {
            Ok(request) => dispatch(&runtime, request),
            Err(error) => {
                JsonRpcResponse::err(Value::Null, -32700, format!("JSON 解析失败：{error}"))
            }
        };
        serde_json::to_writer(&mut stdout, &response)?;
        stdout.write_all(b"\n")?;
        stdout.flush()?;
    }
    Ok(())
}

fn dispatch(runtime: &Runtime, request: JsonRpcRequest) -> JsonRpcResponse {
    if request.jsonrpc != "2.0" {
        return JsonRpcResponse::err(request.id, -32600, "仅支持 JSON-RPC 2.0");
    }
    let id = request.id.clone();
    let result: Result<Value> = (|| match request.method.as_str() {
        "initialize" | "runtime/status" => Ok(serde_json::to_value(runtime.status())?),
        "tools/list" => Ok(serde_json::to_value(runtime.tools())?),
        "session/create" => {
            let cwd = request.params.get("cwd").and_then(Value::as_str);
            let mode: ApprovalMode = serde_json::from_value(
                request
                    .params
                    .get("approvalMode")
                    .cloned()
                    .unwrap_or(json!("ask")),
            )?;
            Ok(serde_json::to_value(runtime.create_session(cwd, mode)?)?)
        }
        "brain/prepare" => Ok(serde_json::to_value(runtime.prepare_brain(request.params.clone())?)?),
        "brain/resolve" => {
            let turn: xiaoyu_protocol::ModelTurn = serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.resolve_brain(turn)?)?)
        }
        "policy/preview" => {
            let risk: RiskLevel = serde_json::from_value(
                request.params.get("risk").cloned().unwrap_or(json!("system")),
            )?;
            let mode: ApprovalMode = serde_json::from_value(
                request
                    .params
                    .get("approvalMode")
                    .cloned()
                    .unwrap_or(json!("ask")),
            )?;
            Ok(json!({"decision": runtime.approval_hint(risk, mode)}))
        }
        _ => Err(anyhow::anyhow!("未知 RPC 方法：{}", request.method)),
    })();

    match result {
        Ok(value) => JsonRpcResponse::ok(id, value),
        Err(error) => JsonRpcResponse::err(id, -32000, error.to_string()),
    }
}
