use serde::{Deserialize, Serialize}; use serde_json::Value;
pub const PROTOCOL_VERSION:&str="agmp.native.v1";
#[derive(Debug,Serialize,Deserialize)]pub struct JsonRpcRequest{pub jsonrpc:String,pub id:u64,pub method:String,#[serde(default)]pub params:Value}
#[derive(Debug,Serialize,Deserialize)]pub struct JsonRpcError{pub code:i64,pub message:String}
#[derive(Debug,Serialize,Deserialize)]pub struct JsonRpcResponse{pub jsonrpc:String,pub id:u64,#[serde(skip_serializing_if="Option::is_none")]pub result:Option<Value>,#[serde(skip_serializing_if="Option::is_none")]pub error:Option<JsonRpcError>}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct RuntimeStatus{pub name:String,pub version:String,pub protocol:String,pub ready:bool,pub capabilities:Vec<String>}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct CapabilityIssueRequest{pub run_id:String,pub tool:String,pub approval_hash:String,pub scope:String,pub payload:Value,#[serde(default)]pub ttl_ms:u64}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct CapabilityLease{pub id:String,pub scope:String,pub run_id:String,pub tool:String,pub expires_at:u64,pub max_uses:u8}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct FileWriteTextRequest{pub path:String,pub content:String,pub capability_lease_id:String,pub capability_scope:String,pub run_id:String,pub tool:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct FileCreateDirRequest{pub path:String,pub capability_lease_id:String,pub capability_scope:String,pub run_id:String,pub tool:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct DownloadRequest{pub url:String,pub path:String,pub hash:String,pub hash_type:String,#[serde(default)]pub max_bytes:u64,pub capability_lease_id:String,pub capability_scope:String,pub run_id:String,pub tool:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct JobStartRequest{pub session_id:Option<String>,pub executable:String,#[serde(default)]pub arguments:Vec<String>,pub cwd:Option<String>,#[serde(default)]pub max_output_bytes:usize,pub capability_lease_id:String,pub capability_scope:String,pub run_id:String,pub tool:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct JobSnapshot{pub id:String,pub state:String,pub pid:u32,pub exit_code:Option<i32>,pub cwd:String,pub created_at:u64}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct OutputChunk{pub sequence:u64,pub stream:String,pub text:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct JavaRuntimeInfo{pub available:bool,pub executable:String,pub major:u32,pub version:String,pub source:String,pub error:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct ManagedJavaRequest{pub major:u32,pub capability_lease_id:String,pub capability_scope:String,pub run_id:String,pub tool:String}
#[derive(Debug,Clone,Serialize,Deserialize)]#[serde(rename_all="camelCase")]pub struct MinecraftPingResult{pub reachable:bool,pub latency_ms:u128,pub version:String,pub protocol:i64,pub players_online:i64,pub players_max:i64,pub description:String}
