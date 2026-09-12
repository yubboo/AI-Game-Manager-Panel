use anyhow::{Context, Result, bail};
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};
use uuid::Uuid;
use xiaoyu_protocol::{ApprovalMode, SessionInfo};

#[derive(Debug, Clone)]
pub struct SessionManager {
    root: PathBuf,
    sessions: Arc<Mutex<HashMap<String, SessionInfo>>>,
}

impl SessionManager {
    pub fn new(root: PathBuf) -> Self {
        Self {
            root,
            sessions: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn create(&self, cwd: Option<&str>, mode: ApprovalMode) -> Result<SessionInfo> {
        let cwd = resolve_cwd(&self.root, cwd)?;
        let session = SessionInfo {
            id: format!("XY-{}", Uuid::new_v4()),
            cwd: cwd.to_string_lossy().into_owned(),
            approval_mode: mode,
        };
        self.sessions
            .lock()
            .map_err(|_| anyhow::anyhow!("session registry lock poisoned"))?
            .insert(session.id.clone(), session.clone());
        Ok(session)
    }

    pub fn get(&self, id: &str) -> Result<SessionInfo> {
        let id = id.trim();
        if id.is_empty() {
            bail!("session id is required");
        }
        self.sessions
            .lock()
            .map_err(|_| anyhow::anyhow!("session registry lock poisoned"))?
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("unknown session: {id}"))
    }

    pub fn list(&self) -> Result<Vec<SessionInfo>> {
        let mut sessions = self
            .sessions
            .lock()
            .map_err(|_| anyhow::anyhow!("session registry lock poisoned"))?
            .values()
            .cloned()
            .collect::<Vec<_>>();
        sessions.sort_by(|a, b| a.id.cmp(&b.id));
        Ok(sessions)
    }

    pub fn close(&self, id: &str) -> Result<bool> {
        let id = id.trim();
        if id.is_empty() {
            bail!("session id is required");
        }
        Ok(self
            .sessions
            .lock()
            .map_err(|_| anyhow::anyhow!("session registry lock poisoned"))?
            .remove(id)
            .is_some())
    }
}

pub(crate) fn resolve_cwd(root: &Path, requested: Option<&str>) -> Result<PathBuf> {
    let root = root
        .canonicalize()
        .with_context(|| format!("cannot resolve runtime root: {}", root.display()))?;
    let candidate = match requested.map(str::trim).filter(|value| !value.is_empty()) {
        Some(value) => {
            let path = PathBuf::from(value);
            if path.is_absolute() {
                path
            } else {
                root.join(path)
            }
        }
        None => root.clone(),
    };
    let candidate = candidate
        .canonicalize()
        .with_context(|| format!("cannot resolve working directory: {}", candidate.display()))?;
    if !candidate.starts_with(&root) {
        bail!(
            "working directory escapes runtime root: {}",
            candidate.display()
        );
    }
    if !candidate.is_dir() {
        bail!("working directory is not a directory: {}", candidate.display());
    }
    Ok(candidate)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn session_registry_tracks_lifecycle() {
        let root = std::env::current_dir().unwrap();
        let manager = SessionManager::new(root.clone());
        let session = manager.create(None, ApprovalMode::Ask).unwrap();
        assert_eq!(
            manager.get(&session.id).unwrap().cwd,
            root.canonicalize().unwrap().to_string_lossy().into_owned()
        );
        assert_eq!(manager.list().unwrap().len(), 1);
        assert!(manager.close(&session.id).unwrap());
        assert!(manager.get(&session.id).is_err());
    }
}
