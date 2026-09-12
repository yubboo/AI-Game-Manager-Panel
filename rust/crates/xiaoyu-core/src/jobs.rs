use crate::session::resolve_cwd;
use anyhow::{Context, Result, bail};
use std::collections::{HashMap, VecDeque};
use std::io::Read;
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use uuid::Uuid;
use xiaoyu_protocol::{
    JobOutputChunk, JobOutputRequest, JobOutputResponse, JobSnapshot, JobStartRequest, JobState,
    SessionInfo,
};

const DEFAULT_OUTPUT_BYTES: usize = 512 * 1024;
const MAX_OUTPUT_BYTES: usize = 4 * 1024 * 1024;
const MAX_JOBS: usize = 64;
const DEFAULT_OUTPUT_CHUNKS: usize = 200;
const MAX_OUTPUT_CHUNKS: usize = 1000;

#[derive(Clone)]
pub struct JobManager {
    root: PathBuf,
    jobs: Arc<Mutex<HashMap<String, JobEntry>>>,
}

#[derive(Clone)]
struct JobEntry {
    state: Arc<Mutex<JobStateData>>,
    child: Arc<Mutex<Option<Child>>>,
    output: Arc<Mutex<OutputBuffer>>,
}

#[derive(Debug, Clone)]
struct JobStateData {
    id: String,
    session_id: Option<String>,
    executable: String,
    arguments: Vec<String>,
    cwd: String,
    state: JobState,
    pid: Option<u32>,
    exit_code: Option<i32>,
    created_at: u64,
    started_at: Option<u64>,
    finished_at: Option<u64>,
    cancel_requested: bool,
}

#[derive(Debug)]
struct OutputBuffer {
    chunks: VecDeque<JobOutputChunk>,
    total_bytes: usize,
    max_bytes: usize,
    next_sequence: u64,
    truncated: bool,
}

impl OutputBuffer {
    fn new(max_bytes: usize) -> Self {
        Self {
            chunks: VecDeque::new(),
            total_bytes: 0,
            max_bytes,
            next_sequence: 1,
            truncated: false,
        }
    }

    fn append(&mut self, stream: &str, mut text: String) {
        if text.is_empty() {
            return;
        }
        if text.len() > self.max_bytes {
            let mut start = text.len() - self.max_bytes;
            while start < text.len() && !text.is_char_boundary(start) {
                start += 1;
            }
            text = text[start..].to_string();
            self.truncated = true;
        }
        let bytes = text.len();
        let chunk = JobOutputChunk {
            sequence: self.next_sequence,
            stream: stream.to_string(),
            text,
        };
        self.next_sequence += 1;
        self.total_bytes += bytes;
        self.chunks.push_back(chunk);
        while self.total_bytes > self.max_bytes {
            if let Some(removed) = self.chunks.pop_front() {
                self.total_bytes = self.total_bytes.saturating_sub(removed.text.len());
                self.truncated = true;
            } else {
                break;
            }
        }
    }
}

impl JobManager {
    pub fn new(root: PathBuf) -> Self {
        Self {
            root,
            jobs: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn start(
        &self,
        request: JobStartRequest,
        session: Option<&SessionInfo>,
    ) -> Result<JobSnapshot> {
        if !request.host_authorized {
            bail!("job start requires Host authorization");
        }
        let executable = request.executable.trim();
        if executable.is_empty() {
            bail!("job executable is required");
        }
        self.prune_completed()?;
        if self
            .jobs
            .lock()
            .map_err(|_| anyhow::anyhow!("job registry lock poisoned"))?
            .len()
            >= MAX_JOBS
        {
            bail!("job registry is full");
        }

        let requested_cwd = request
            .cwd
            .as_deref()
            .or_else(|| session.map(|value| value.cwd.as_str()));
        let cwd = resolve_cwd(&self.root, requested_cwd)?;
        let max_output_bytes = match request.max_output_bytes {
            0 => DEFAULT_OUTPUT_BYTES,
            value => value.clamp(1024, MAX_OUTPUT_BYTES),
        };

        let mut command = Command::new(executable);
        command
            .args(&request.arguments)
            .current_dir(&cwd)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let mut child = command
            .spawn()
            .with_context(|| format!("failed to start job executable: {executable}"))?;

        let id = format!("JOB-{}", Uuid::new_v4());
        let now = unix_millis();
        let state = Arc::new(Mutex::new(JobStateData {
            id: id.clone(),
            session_id: request.session_id.clone(),
            executable: executable.to_string(),
            arguments: request.arguments.clone(),
            cwd: cwd.to_string_lossy().into_owned(),
            state: JobState::Running,
            pid: Some(child.id()),
            exit_code: None,
            created_at: now,
            started_at: Some(now),
            finished_at: None,
            cancel_requested: false,
        }));
        let output = Arc::new(Mutex::new(OutputBuffer::new(max_output_bytes)));
        let readers = Arc::new(AtomicUsize::new(0));

        if let Some(stdout) = child.stdout.take() {
            spawn_reader(stdout, "stdout", output.clone(), readers.clone());
        }
        if let Some(stderr) = child.stderr.take() {
            spawn_reader(stderr, "stderr", output.clone(), readers.clone());
        }

        let child = Arc::new(Mutex::new(Some(child)));
        let entry = JobEntry {
            state: state.clone(),
            child: child.clone(),
            output: output.clone(),
        };
        self.jobs
            .lock()
            .map_err(|_| anyhow::anyhow!("job registry lock poisoned"))?
            .insert(id, entry.clone());
        spawn_monitor(state, child, readers);
        self.snapshot_from_parts(&entry)
    }

    pub fn get(&self, id: &str) -> Result<JobSnapshot> {
        let entry = self.entry(id)?;
        self.snapshot_from_parts(&entry)
    }

    pub fn list(&self) -> Result<Vec<JobSnapshot>> {
        let entries = self
            .jobs
            .lock()
            .map_err(|_| anyhow::anyhow!("job registry lock poisoned"))?
            .values()
            .cloned()
            .collect::<Vec<_>>();
        let mut snapshots = entries
            .iter()
            .map(|entry| self.snapshot_from_parts(entry))
            .collect::<Result<Vec<_>>>()?;
        snapshots.sort_by(|a, b| b.created_at.cmp(&a.created_at));
        Ok(snapshots)
    }

    pub fn output(&self, request: JobOutputRequest) -> Result<JobOutputResponse> {
        let entry = self.entry(&request.id)?;
        let output = entry
            .output
            .lock()
            .map_err(|_| anyhow::anyhow!("job output lock poisoned"))?;
        let limit = match request.limit {
            0 => DEFAULT_OUTPUT_CHUNKS,
            value => value.clamp(1, MAX_OUTPUT_CHUNKS),
        };
        let chunks = output
            .chunks
            .iter()
            .filter(|chunk| chunk.sequence > request.after)
            .take(limit)
            .cloned()
            .collect::<Vec<_>>();
        let next_cursor = chunks
            .last()
            .map(|chunk| chunk.sequence)
            .unwrap_or(request.after);
        Ok(JobOutputResponse {
            id: request.id,
            chunks,
            next_cursor,
            truncated: output.truncated,
        })
    }

    pub fn cancel(&self, id: &str) -> Result<JobSnapshot> {
        let entry = self.entry(id)?;
        {
            let mut state = entry
                .state
                .lock()
                .map_err(|_| anyhow::anyhow!("job state lock poisoned"))?;
            if state.state.is_terminal() {
                drop(state);
                return self.snapshot_from_parts(&entry);
            }
            state.cancel_requested = true;
        }
        let mut child = entry
            .child
            .lock()
            .map_err(|_| anyhow::anyhow!("job child lock poisoned"))?;
        if let Some(process) = child.as_mut() {
            if process.try_wait()?.is_none() {
                process.kill().context("failed to cancel job")?;
            }
        }
        drop(child);
        self.snapshot_from_parts(&entry)
    }

    fn entry(&self, id: &str) -> Result<JobEntry> {
        let id = id.trim();
        if id.is_empty() {
            bail!("job id is required");
        }
        self.jobs
            .lock()
            .map_err(|_| anyhow::anyhow!("job registry lock poisoned"))?
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("unknown job: {id}"))
    }

    fn snapshot_from_parts(&self, entry: &JobEntry) -> Result<JobSnapshot> {
        let state = entry
            .state
            .lock()
            .map_err(|_| anyhow::anyhow!("job state lock poisoned"))?
            .clone();
        let output = entry
            .output
            .lock()
            .map_err(|_| anyhow::anyhow!("job output lock poisoned"))?;
        Ok(JobSnapshot {
            id: state.id,
            session_id: state.session_id,
            executable: state.executable,
            arguments: state.arguments,
            cwd: state.cwd,
            state: state.state,
            pid: state.pid,
            exit_code: state.exit_code,
            created_at: state.created_at,
            started_at: state.started_at,
            finished_at: state.finished_at,
            output_truncated: output.truncated,
        })
    }

    fn prune_completed(&self) -> Result<()> {
        let mut jobs = self
            .jobs
            .lock()
            .map_err(|_| anyhow::anyhow!("job registry lock poisoned"))?;
        if jobs.len() < MAX_JOBS {
            return Ok(());
        }
        let mut completed = jobs
            .iter()
            .filter_map(|(id, entry)| {
                let state = entry.state.lock().ok()?;
                state
                    .state
                    .is_terminal()
                    .then_some((id.clone(), state.finished_at.unwrap_or(state.created_at)))
            })
            .collect::<Vec<_>>();
        completed.sort_by_key(|(_, finished_at)| *finished_at);
        while jobs.len() >= MAX_JOBS {
            let Some((id, _)) = completed.first().cloned() else {
                break;
            };
            completed.remove(0);
            jobs.remove(&id);
        }
        Ok(())
    }
}

fn spawn_reader<R>(
    reader: R,
    stream: &'static str,
    output: Arc<Mutex<OutputBuffer>>,
    readers: Arc<AtomicUsize>,
) where
    R: Read + Send + 'static,
{
    readers.fetch_add(1, Ordering::SeqCst);
    thread::spawn(move || {
        let mut reader = reader;
        let mut chunk = [0_u8; 8192];
        loop {
            match reader.read(&mut chunk) {
                Ok(0) => break,
                Ok(size) => {
                    if let Ok(mut buffer) = output.lock() {
                        buffer.append(stream, String::from_utf8_lossy(&chunk[..size]).into_owned());
                    }
                }
                Err(error) => {
                    if let Ok(mut buffer) = output.lock() {
                        buffer.append("runtime", format!("output read failed: {error}\n"));
                    }
                    break;
                }
            }
        }
        readers.fetch_sub(1, Ordering::SeqCst);
    });
}

fn spawn_monitor(
    state: Arc<Mutex<JobStateData>>,
    child: Arc<Mutex<Option<Child>>>,
    readers: Arc<AtomicUsize>,
) {
    thread::spawn(move || {
        let exit_status = loop {
            let result = {
                let mut child = match child.lock() {
                    Ok(value) => value,
                    Err(_) => return,
                };
                match child.as_mut() {
                    Some(process) => process.try_wait(),
                    None => return,
                }
            };
            match result {
                Ok(Some(status)) => break Ok(status),
                Ok(None) => thread::sleep(Duration::from_millis(50)),
                Err(error) => break Err(error),
            }
        };
        while readers.load(Ordering::SeqCst) > 0 {
            thread::sleep(Duration::from_millis(10));
        }
        let mut state = match state.lock() {
            Ok(value) => value,
            Err(_) => return,
        };
        state.finished_at = Some(unix_millis());
        match exit_status {
            Ok(status) => {
                state.exit_code = status.code();
                state.state = if state.cancel_requested {
                    JobState::Cancelled
                } else if status.success() {
                    JobState::Succeeded
                } else {
                    JobState::Failed
                };
            }
            Err(_) => {
                state.exit_code = None;
                state.state = if state.cancel_requested {
                    JobState::Cancelled
                } else {
                    JobState::Failed
                };
            }
        }
        if let Ok(mut child) = child.lock() {
            *child = None;
        }
    });
}

fn unix_millis() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis()
        .min(u128::from(u64::MAX)) as u64
}

#[cfg(test)]
mod tests {
    use super::*;

    fn echo_request(root: &std::path::Path) -> JobStartRequest {
        #[cfg(windows)]
        let (executable, arguments) = (
            "cmd".to_string(),
            vec!["/C".to_string(), "echo xiaoyu-job".to_string()],
        );
        #[cfg(not(windows))]
        let (executable, arguments) = (
            "sh".to_string(),
            vec!["-c".to_string(), "printf 'xiaoyu-job\\n'".to_string()],
        );
        JobStartRequest {
            session_id: None,
            executable,
            arguments,
            cwd: Some(root.to_string_lossy().into_owned()),
            max_output_bytes: 16 * 1024,
            host_authorized: true,
        }
    }

    fn long_request(root: &std::path::Path) -> JobStartRequest {
        #[cfg(windows)]
        let (executable, arguments) = (
            "ping".to_string(),
            vec!["-n".to_string(), "30".to_string(), "127.0.0.1".to_string()],
        );
        #[cfg(not(windows))]
        let (executable, arguments) = ("sleep".to_string(), vec!["5".to_string()]);
        JobStartRequest {
            session_id: None,
            executable,
            arguments,
            cwd: Some(root.to_string_lossy().into_owned()),
            max_output_bytes: 16 * 1024,
            host_authorized: true,
        }
    }

    #[test]
    fn job_start_requires_host_authorization() {
        let root = std::env::current_dir().unwrap();
        let manager = JobManager::new(root.clone());
        let mut request = echo_request(&root);
        request.host_authorized = false;
        assert!(manager.start(request, None).is_err());
    }

    #[test]
    fn job_runs_and_exposes_bounded_output() {
        let root = std::env::current_dir().unwrap();
        let manager = JobManager::new(root.clone());
        let job = manager.start(echo_request(&root), None).unwrap();
        for _ in 0..100 {
            let status = manager.get(&job.id).unwrap();
            if status.state.is_terminal() {
                assert_eq!(status.state, JobState::Succeeded);
                let output = manager
                    .output(JobOutputRequest {
                        id: job.id.clone(),
                        after: 0,
                        limit: 20,
                    })
                    .unwrap();
                assert!(
                    output
                        .chunks
                        .iter()
                        .any(|chunk| chunk.text.contains("xiaoyu-job"))
                );
                return;
            }
            thread::sleep(Duration::from_millis(20));
        }
        panic!("job did not finish in time");
    }

    #[test]
    fn running_job_can_be_cancelled() {
        let root = std::env::current_dir().unwrap();
        let manager = JobManager::new(root.clone());
        let job = manager.start(long_request(&root), None).unwrap();
        let cancelled = manager.cancel(&job.id).unwrap();
        assert!(cancelled.state == JobState::Running || cancelled.state == JobState::Cancelled);
        for _ in 0..100 {
            let status = manager.get(&job.id).unwrap();
            if status.state.is_terminal() {
                assert_eq!(status.state, JobState::Cancelled);
                return;
            }
            thread::sleep(Duration::from_millis(20));
        }
        panic!("cancelled job did not become terminal in time");
    }
}
