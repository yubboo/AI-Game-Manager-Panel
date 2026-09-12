use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use serde_json::{Value, json};
use std::io::{self, BufRead, Write};
use std::path::PathBuf;
use xiaoyu_core::{RUNTIME_VERSION, Runtime};
use xiaoyu_protocol::{ApprovalMode, JsonRpcRequest, JsonRpcResponse, RiskLevel};

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
    /// Compatibility/introspection command. Domain Tools live in AGMP Go
    /// services; generic native Tools migrate into XiaoYu Runtime incrementally.
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
        "tools/search" => {
            let search: xiaoyu_protocol::ToolSearchRequest =
                serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.search_tools(search))?)
        }
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
        "session/get" => {
            let id = request
                .params
                .get("id")
                .and_then(Value::as_str)
                .unwrap_or_default();
            Ok(serde_json::to_value(runtime.get_session(id)?)?)
        }
        "session/list" => Ok(serde_json::to_value(runtime.list_sessions()?)?),
        "session/close" => {
            let id = request
                .params
                .get("id")
                .and_then(Value::as_str)
                .unwrap_or_default();
            Ok(json!({"closed": runtime.close_session(id)?}))
        }
        "jobs/start" => {
            let job: xiaoyu_protocol::JobStartRequest =
                serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.start_job(job)?)?)
        }
        "jobs/get" => {
            let id = request
                .params
                .get("id")
                .and_then(Value::as_str)
                .unwrap_or_default();
            Ok(serde_json::to_value(runtime.get_job(id)?)?)
        }
        "jobs/list" => Ok(serde_json::to_value(runtime.list_jobs()?)?),
        "jobs/output" => {
            let output: xiaoyu_protocol::JobOutputRequest =
                serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.job_output(output)?)?)
        }
        "jobs/cancel" => {
            let id = request
                .params
                .get("id")
                .and_then(Value::as_str)
                .unwrap_or_default();
            Ok(serde_json::to_value(runtime.cancel_job(id)?)?)
        }
        "terminal/start" => {
            let terminal: xiaoyu_protocol::TerminalStartRequest =
                serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.start_terminal(terminal)?)?)
        }
        "terminal/get" => {
            let id = request
                .params
                .get("id")
                .and_then(Value::as_str)
                .unwrap_or_default();
            Ok(serde_json::to_value(runtime.get_terminal(id)?)?)
        }
        "terminal/list" => Ok(serde_json::to_value(runtime.list_terminals()?)?),
        "terminal/write" => {
            let write: xiaoyu_protocol::TerminalWriteRequest =
                serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.write_terminal(write)?)?)
        }
        "terminal/output" => {
            let output: xiaoyu_protocol::TerminalOutputRequest =
                serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.terminal_output(output)?)?)
        }
        "terminal/close" => {
            let id = request
                .params
                .get("id")
                .and_then(Value::as_str)
                .unwrap_or_default();
            Ok(serde_json::to_value(runtime.close_terminal(id)?)?)
        }
        "brain/prepare" => Ok(serde_json::to_value(
            runtime.prepare_brain(request.params.clone())?,
        )?),
        "brain/resolve" => {
            let turn: xiaoyu_protocol::ModelTurn = serde_json::from_value(request.params.clone())?;
            Ok(serde_json::to_value(runtime.resolve_brain(turn)?)?)
        }
        "policy/preview" => {
            let risk: RiskLevel = serde_json::from_value(
                request
                    .params
                    .get("risk")
                    .cloned()
                    .unwrap_or(json!("system")),
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
