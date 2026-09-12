use agmp_native_protocol::{CapabilityIssueRequest, ManagedJavaRequest};
use agmp_native_runtime::NativeRuntime;
use serde_json::json;
use std::fs;
use uuid::Uuid;

#[test]
#[ignore = "downloads a real Temurin JRE; GitHub Runner smoke only"]
fn managed_java_real_smoke() {
    let root = std::env::temp_dir().join(format!("agmp-java-smoke-{}", Uuid::new_v4()));
    let workspace = root.join("workspace");
    let managed = root.join("native");
    fs::create_dir_all(&workspace).unwrap();
    fs::create_dir_all(&managed).unwrap();
    let runtime = NativeRuntime::new(workspace, managed).unwrap();
    let lease = runtime.issue(CapabilityIssueRequest {
        run_id: "run-java-smoke".into(),
        tool: "system.java.ensure".into(),
        approval_hash: "github-runner-approved".into(),
        scope: "runtime.java.ensure".into(),
        payload: json!({"major": 21}),
        ttl_ms: 120_000,
    }).unwrap();
    let info = runtime.java_ensure(ManagedJavaRequest {
        major: 21,
        capability_lease_id: lease.id,
        capability_scope: lease.scope,
        run_id: "run-java-smoke".into(),
        tool: "system.java.ensure".into(),
    }).unwrap();
    assert!(info.available, "managed Java should be available: {:?}", info.error);
    assert_eq!(info.major, 21);
    assert!(!info.executable.is_empty());
    let _ = fs::remove_dir_all(root);
}
