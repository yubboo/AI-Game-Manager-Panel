use crate::session::resolve_cwd;
use anyhow::{Result, bail};
#[cfg(not(target_os = "linux"))]
use anyhow::Context;
use std::collections::{HashMap, VecDeque};
use std::io::{self, Read, Write};
use std::path::PathBuf;
use std::process::Child;
#[cfg(not(target_os = "linux"))]
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use uuid::Uuid;
use xiaoyu_protocol::{
    JobOutputChunk, SessionInfo, TerminalOutputRequest, TerminalOutputResponse,
    TerminalResizeRequest, TerminalSnapshot, TerminalStartRequest, TerminalState,
    TerminalWriteRequest,
};

const DEFAULT_OUTPUT_BYTES: usize = 512 * 1024;
const MAX_OUTPUT_BYTES: usize = 4 * 1024 * 1024;
const MAX_TERMINALS: usize = 32;
const DEFAULT_OUTPUT_CHUNKS: usize = 200;
const MAX_OUTPUT_CHUNKS: usize = 1000;
const MAX_WRITE_BYTES: usize = 64 * 1024;
const DEFAULT_ROWS: u16 = 24;
const DEFAULT_COLS: u16 = 80;
const PIPE_BACKEND: &str = "stdio-pipe-v1";
#[cfg(target_os = "linux")]
const NATIVE_PTY_BACKEND: &str = "linux-pty-v1";

#[derive(Clone)]
pub struct TerminalManager {
    root: PathBuf,
    terminals: Arc<Mutex<HashMap<String, TerminalEntry>>>,
}

#[derive(Clone)]
struct TerminalEntry {
    state: Arc<Mutex<TerminalStateData>>,
    process: Arc<Mutex<Option<TerminalProcess>>>,
    input: Arc<Mutex<Option<Box<dyn Write + Send>>>>,
    output: Arc<Mutex<OutputBuffer>>,
}

#[derive(Debug, Clone)]
struct TerminalStateData {
    id: String,
    session_id: Option<String>,
    executable: String,
    arguments: Vec<String>,
    cwd: String,
    state: TerminalState,
    pid: Option<u32>,
    exit_code: Option<i32>,
    created_at: u64,
    finished_at: Option<u64>,
    close_requested: bool,
    backend: String,
    rows: u16,
    cols: u16,
}

enum TerminalProcess {
    Pipe(Child),
    #[cfg(target_os = "linux")]
    LinuxPty(crate::pty_linux::LinuxPtyProcess),
}

impl TerminalProcess {
    fn pid(&self) -> u32 {
        match self {
            Self::Pipe(child) => child.id(),
            #[cfg(target_os = "linux")]
            Self::LinuxPty(process) => process.pid(),
        }
    }

    fn try_wait(&mut self) -> io::Result<Option<i32>> {
        match self {
            Self::Pipe(child) => child
                .try_wait()
                .map(|status| status.map(|status| status.code().unwrap_or_default())),
            #[cfg(target_os = "linux")]
            Self::LinuxPty(process) => process.try_wait(),
        }
    }

    fn kill(&mut self) -> io::Result<()> {
        match self {
            Self::Pipe(child) => child.kill(),
            #[cfg(target_os = "linux")]
            Self::LinuxPty(process) => process.kill(),
        }
    }

    fn resize(&mut self, rows: u16, cols: u16) -> io::Result<()> {
        match self {
            Self::Pipe(_) => Err(io::Error::new(
                io::ErrorKind::Unsupported,
                "terminal backend does not support resize",
            )),
            #[cfg(target_os = "linux")]
            Self::LinuxPty(process) => process.resize(rows, cols),
        }
    }
}

struct SpawnedTerminal {
    process: TerminalProcess,
    input: Box<dyn Write + Send>,
    readers: Vec<(&'static str, Box<dyn Read + Send>)>,
    backend: &'static str,
}

fn spawn_terminal(
    executable: &str,
    arguments: &[String],
    cwd: &std::path::Path,
    rows: u16,
    cols: u16,
) -> Result<SpawnedTerminal> {
    #[cfg(target_os = "linux")]
    {
        let spawned = crate::pty_linux::spawn(executable, arguments, cwd, rows, cols)?;
        return Ok(SpawnedTerminal {
            process: TerminalProcess::LinuxPty(spawned.process),
            input: Box::new(spawned.writer),
            readers: vec![("pty", Box::new(spawned.reader))],
            backend: NATIVE_PTY_BACKEND,
        });
    }

    #[cfg(not(target_os = "linux"))]
    {
        let mut command = Command::new(executable);
        command
            .args(arguments)
            .current_dir(cwd)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let mut child = command
            .spawn()
            .with_context(|| format!("cannot start terminal executable: {executable}"))?;
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| anyhow::anyhow!("terminal stdin pipe unavailable"))?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| anyhow::anyhow!("terminal stdout pipe unavailable"))?;
        let stderr = child
            .stderr
            .take()
            .ok_or_else(|| anyhow::anyhow!("terminal stderr pipe unavailable"))?;
        Ok(SpawnedTerminal {
            process: TerminalProcess::Pipe(child),
            input: Box::new(stdin),
            readers: vec![("stdout", Box::new(stdout)), ("stderr", Box::new(stderr))],
            backend: PIPE_BACKEND,
        })
    }
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
        self.chunks.push_back(JobOutputChunk {
            sequence: self.next_sequence,
            stream: stream.to_string(),
            text,
        });
        self.next_sequence = self.next_sequence.saturating_add(1);
        self.total_bytes = self.total_bytes.saturating_add(bytes);
        while self.total_bytes > self.max_bytes {
            let Some(front) = self.chunks.pop_front() else {
                break;
            };
            self.total_bytes = self.total_bytes.saturating_sub(front.text.len());
            self.truncated = true;
        }
    }

    fn read(&self, id: String, after: u64, limit: usize) -> TerminalOutputResponse {
        let limit = if limit == 0 {
            DEFAULT_OUTPUT_CHUNKS
        } else {
            limit.min(MAX_OUTPUT_CHUNKS)
        };
        let chunks = self
            .chunks
            .iter()
            .filter(|chunk| chunk.sequence > after)
            .take(limit)
            .cloned()
            .collect::<Vec<_>>();
        let next_cursor = chunks.last().map(|chunk| chunk.sequence).unwrap_or(after);
        TerminalOutputResponse {
            id,
            chunks,
            next_cursor,
            truncated: self.truncated,
        }
    }
}

impl TerminalManager {
    pub fn new(root: PathBuf) -> Self {
        Self {
            root,
            terminals: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn start(
        &self,
        request: TerminalStartRequest,
        session: Option<&SessionInfo>,
    ) -> Result<TerminalSnapshot> {
        if !request.host_authorized {
            bail!("terminal start requires Host authorization");
        }
        let executable = request.executable.trim();
        if executable.is_empty() {
            bail!("terminal executable is required");
        }
        let cwd = if let Some(session) = session {
            resolve_cwd(
                &self.root,
                request.cwd.as_deref().or(Some(session.cwd.as_str())),
            )?
        } else {
            resolve_cwd(&self.root, request.cwd.as_deref())?
        };
        let max_output_bytes = if request.max_output_bytes == 0 {
            DEFAULT_OUTPUT_BYTES
        } else {
            request.max_output_bytes.min(MAX_OUTPUT_BYTES)
        };
        {
            let terminals = self
                .terminals
                .lock()
                .map_err(|_| anyhow::anyhow!("terminal registry lock poisoned"))?;
            if terminals.len() >= MAX_TERMINALS {
                bail!("too many terminal sessions; close an existing terminal first");
            }
        }

        let rows = if request.rows == 0 {
            DEFAULT_ROWS
        } else {
            request.rows
        };
        let cols = if request.cols == 0 {
            DEFAULT_COLS
        } else {
            request.cols
        };
        let spawned = spawn_terminal(executable, &request.arguments, &cwd, rows, cols)?;
        let pid = spawned.process.pid();

        let id = format!("XYT-{}", Uuid::new_v4());
        let created_at = unix_millis();
        let state = Arc::new(Mutex::new(TerminalStateData {
            id: id.clone(),
            session_id: request.session_id.clone(),
            executable: executable.to_string(),
            arguments: request.arguments.clone(),
            cwd: cwd.to_string_lossy().into_owned(),
            state: TerminalState::Running,
            pid: Some(pid),
            exit_code: None,
            created_at,
            finished_at: None,
            close_requested: false,
            backend: spawned.backend.to_string(),
            rows,
            cols,
        }));
        let process = Arc::new(Mutex::new(Some(spawned.process)));
        let input = Arc::new(Mutex::new(Some(spawned.input)));
        let output = Arc::new(Mutex::new(OutputBuffer::new(max_output_bytes)));
        let readers = Arc::new(AtomicUsize::new(0));
        for (stream, reader) in spawned.readers {
            spawn_reader(reader, stream, output.clone(), readers.clone());
        }
        spawn_monitor(state.clone(), process.clone(), input.clone(), readers);

        let entry = TerminalEntry {
            state,
            process,
            input,
            output,
        };
        let snapshot = snapshot(&entry)?;
        self.terminals
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal registry lock poisoned"))?
            .insert(id, entry);
        Ok(snapshot)
    }

    pub fn get(&self, id: &str) -> Result<TerminalSnapshot> {
        let entry = self.entry(id)?;
        snapshot(&entry)
    }

    pub fn list(&self) -> Result<Vec<TerminalSnapshot>> {
        let entries = self
            .terminals
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal registry lock poisoned"))?
            .values()
            .cloned()
            .collect::<Vec<_>>();
        let mut snapshots = Vec::with_capacity(entries.len());
        for entry in entries {
            snapshots.push(snapshot(&entry)?);
        }
        snapshots.sort_by(|a, b| a.id.cmp(&b.id));
        Ok(snapshots)
    }

    pub fn write(&self, request: TerminalWriteRequest) -> Result<TerminalSnapshot> {
        if !request.host_authorized {
            bail!("terminal input requires Host authorization");
        }
        if request.data.len() > MAX_WRITE_BYTES {
            bail!("terminal input exceeds {MAX_WRITE_BYTES} bytes");
        }
        let entry = self.entry(&request.id)?;
        {
            let state = entry
                .state
                .lock()
                .map_err(|_| anyhow::anyhow!("terminal state lock poisoned"))?;
            if state.state != TerminalState::Running {
                bail!("terminal is not running: {}", request.id);
            }
        }
        let mut input = entry
            .input
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal input lock poisoned"))?;
        let writer = input
            .as_mut()
            .ok_or_else(|| anyhow::anyhow!("terminal input is already closed"))?;
        writer.write_all(request.data.as_bytes())?;
        if request.append_newline {
            writer.write_all(b"\n")?;
        }
        writer.flush()?;
        drop(input);
        snapshot(&entry)
    }

    pub fn output(&self, request: TerminalOutputRequest) -> Result<TerminalOutputResponse> {
        let entry = self.entry(&request.id)?;
        let buffer = entry
            .output
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal output lock poisoned"))?;
        Ok(buffer.read(request.id, request.after, request.limit))
    }

    pub fn resize(&self, request: TerminalResizeRequest) -> Result<TerminalSnapshot> {
        if !request.host_authorized {
            bail!("terminal resize requires Host authorization");
        }
        if request.rows == 0 || request.cols == 0 {
            bail!("terminal resize requires non-zero rows and cols");
        }
        let entry = self.entry(&request.id)?;
        {
            let state = entry
                .state
                .lock()
                .map_err(|_| anyhow::anyhow!("terminal state lock poisoned"))?;
            if state.state != TerminalState::Running {
                bail!("terminal is not running: {}", request.id);
            }
        }
        let mut process = entry
            .process
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal process lock poisoned"))?;
        let process = process
            .as_mut()
            .ok_or_else(|| anyhow::anyhow!("terminal process is unavailable"))?;
        process.resize(request.rows, request.cols)?;
        drop(process);
        {
            let mut state = entry
                .state
                .lock()
                .map_err(|_| anyhow::anyhow!("terminal state lock poisoned"))?;
            state.rows = request.rows;
            state.cols = request.cols;
        }
        snapshot(&entry)
    }

    pub fn close(&self, id: &str) -> Result<TerminalSnapshot> {
        let entry = self.entry(id)?;
        {
            let mut state = entry
                .state
                .lock()
                .map_err(|_| anyhow::anyhow!("terminal state lock poisoned"))?;
            if state.state != TerminalState::Running {
                drop(state);
                return snapshot(&entry);
            }
            state.close_requested = true;
            state.state = TerminalState::Closed;
        }
        if let Ok(mut input) = entry.input.lock() {
            *input = None;
        }
        let mut process = entry
            .process
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal process lock poisoned"))?;
        if let Some(process) = process.as_mut() {
            let _ = process.kill();
        }
        drop(process);
        snapshot(&entry)
    }

    fn entry(&self, id: &str) -> Result<TerminalEntry> {
        let id = id.trim();
        if id.is_empty() {
            bail!("terminal id is required");
        }
        self.terminals
            .lock()
            .map_err(|_| anyhow::anyhow!("terminal registry lock poisoned"))?
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("unknown terminal: {id}"))
    }
}

fn snapshot(entry: &TerminalEntry) -> Result<TerminalSnapshot> {
    let state = entry
        .state
        .lock()
        .map_err(|_| anyhow::anyhow!("terminal state lock poisoned"))?
        .clone();
    let output_truncated = entry
        .output
        .lock()
        .map_err(|_| anyhow::anyhow!("terminal output lock poisoned"))?
        .truncated;
    Ok(TerminalSnapshot {
        id: state.id,
        session_id: state.session_id,
        executable: state.executable,
        arguments: state.arguments,
        cwd: state.cwd,
        state: state.state,
        pid: state.pid,
        exit_code: state.exit_code,
        created_at: state.created_at,
        finished_at: state.finished_at,
        output_truncated,
        backend: state.backend,
        rows: state.rows,
        cols: state.cols,
    })
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
                    #[cfg(target_os = "linux")]
                    if stream == "pty" && error.raw_os_error() == Some(5) {
                        break;
                    }
                    if let Ok(mut buffer) = output.lock() {
                        buffer.append("runtime", format!("terminal output read failed: {error}\n"));
                    }
                    break;
                }
            }
        }
        readers.fetch_sub(1, Ordering::SeqCst);
    });
}

fn spawn_monitor(
    state: Arc<Mutex<TerminalStateData>>,
    process: Arc<Mutex<Option<TerminalProcess>>>,
    input: Arc<Mutex<Option<Box<dyn Write + Send>>>>,
    readers: Arc<AtomicUsize>,
) {
    thread::spawn(move || {
        let exit_status = loop {
            let result = {
                let mut process = match process.lock() {
                    Ok(value) => value,
                    Err(_) => return,
                };
                match process.as_mut() {
                    Some(process) => process.try_wait(),
                    None => return,
                }
            };
            match result {
                Ok(Some(code)) => break Ok(code),
                Ok(None) => thread::sleep(Duration::from_millis(50)),
                Err(error) => break Err(error),
            }
        };
        if let Ok(mut input) = input.lock() {
            *input = None;
        }
        while readers.load(Ordering::SeqCst) > 0 {
            thread::sleep(Duration::from_millis(10));
        }
        let mut state = match state.lock() {
            Ok(value) => value,
            Err(_) => return,
        };
        state.finished_at = Some(unix_millis());
        match exit_status {
            Ok(code) => {
                state.exit_code = Some(code);
                state.state = if state.close_requested {
                    TerminalState::Closed
                } else {
                    TerminalState::Exited
                };
            }
            Err(_) => {
                state.exit_code = None;
                state.state = if state.close_requested {
                    TerminalState::Closed
                } else {
                    TerminalState::Exited
                };
            }
        }
        drop(state);
        if let Ok(mut process) = process.lock() {
            *process = None;
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

    fn shell_request(root: &std::path::Path) -> TerminalStartRequest {
        #[cfg(windows)]
        let (executable, arguments) = ("cmd".to_string(), vec!["/Q".to_string()]);
        #[cfg(not(windows))]
        let (executable, arguments) = ("sh".to_string(), Vec::new());
        TerminalStartRequest {
            session_id: None,
            executable,
            arguments,
            cwd: Some(root.to_string_lossy().into_owned()),
            max_output_bytes: 32 * 1024,
            rows: DEFAULT_ROWS,
            cols: DEFAULT_COLS,
            host_authorized: true,
        }
    }

    #[test]
    fn terminal_start_requires_host_authorization() {
        let root = std::env::current_dir().unwrap();
        let manager = TerminalManager::new(root.clone());
        let mut request = shell_request(&root);
        request.host_authorized = false;
        assert!(manager.start(request, None).is_err());
    }

    #[test]
    fn terminal_write_requires_host_authorization() {
        let root = std::env::current_dir().unwrap();
        let manager = TerminalManager::new(root.clone());
        let terminal = manager.start(shell_request(&root), None).unwrap();
        let result = manager.write(TerminalWriteRequest {
            id: terminal.id.clone(),
            data: "echo blocked".to_string(),
            append_newline: true,
            host_authorized: false,
        });
        assert!(result.is_err());
        let _ = manager.close(&terminal.id);
    }

    #[test]
    fn interactive_terminal_accepts_input_and_exposes_output() {
        let root = std::env::current_dir().unwrap();
        let manager = TerminalManager::new(root.clone());
        let terminal = manager.start(shell_request(&root), None).unwrap();
        #[cfg(target_os = "linux")]
        assert_eq!(terminal.backend, NATIVE_PTY_BACKEND);
        #[cfg(not(target_os = "linux"))]
        assert_eq!(terminal.backend, PIPE_BACKEND);
        manager
            .write(TerminalWriteRequest {
                id: terminal.id.clone(),
                data: "echo xiaoyu-terminal".to_string(),
                append_newline: true,
                host_authorized: true,
            })
            .unwrap();
        manager
            .write(TerminalWriteRequest {
                id: terminal.id.clone(),
                data: "exit".to_string(),
                append_newline: true,
                host_authorized: true,
            })
            .unwrap();
        for _ in 0..150 {
            let status = manager.get(&terminal.id).unwrap();
            let output = manager
                .output(TerminalOutputRequest {
                    id: terminal.id.clone(),
                    after: 0,
                    limit: 100,
                })
                .unwrap();
            if output
                .chunks
                .iter()
                .any(|chunk| chunk.text.contains("xiaoyu-terminal"))
            {
                assert!(matches!(
                    status.state,
                    TerminalState::Running | TerminalState::Exited
                ));
                return;
            }
            thread::sleep(Duration::from_millis(20));
        }
        panic!("interactive terminal output did not arrive in time");
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn linux_terminal_is_backed_by_a_real_tty() {
        let root = std::env::current_dir().unwrap();
        let manager = TerminalManager::new(root.clone());
        let terminal = manager.start(shell_request(&root), None).unwrap();
        assert_eq!(terminal.backend, NATIVE_PTY_BACKEND);
        manager
            .write(TerminalWriteRequest {
                id: terminal.id.clone(),
                data: "printf 'TTY:%s\\n' \"$(test -t 0 && echo yes || echo no)\"".to_string(),
                append_newline: true,
                host_authorized: true,
            })
            .unwrap();
        manager
            .write(TerminalWriteRequest {
                id: terminal.id.clone(),
                data: "exit".to_string(),
                append_newline: true,
                host_authorized: true,
            })
            .unwrap();
        for _ in 0..150 {
            let output = manager
                .output(TerminalOutputRequest {
                    id: terminal.id.clone(),
                    after: 0,
                    limit: 100,
                })
                .unwrap();
            if output
                .chunks
                .iter()
                .any(|chunk| chunk.text.contains("TTY:yes"))
            {
                return;
            }
            thread::sleep(Duration::from_millis(20));
        }
        panic!("Linux PTY did not report a TTY on stdin");
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn linux_terminal_resize_updates_kernel_winsize() {
        let root = std::env::current_dir().unwrap();
        let manager = TerminalManager::new(root.clone());
        let terminal = manager.start(shell_request(&root), None).unwrap();
        let resized = manager
            .resize(TerminalResizeRequest {
                id: terminal.id.clone(),
                rows: 40,
                cols: 120,
                host_authorized: true,
            })
            .unwrap();
        assert_eq!(resized.rows, 40);
        assert_eq!(resized.cols, 120);
        manager
            .write(TerminalWriteRequest {
                id: terminal.id.clone(),
                data: "stty size".to_string(),
                append_newline: true,
                host_authorized: true,
            })
            .unwrap();
        manager
            .write(TerminalWriteRequest {
                id: terminal.id.clone(),
                data: "exit".to_string(),
                append_newline: true,
                host_authorized: true,
            })
            .unwrap();
        for _ in 0..150 {
            let output = manager
                .output(TerminalOutputRequest {
                    id: terminal.id.clone(),
                    after: 0,
                    limit: 100,
                })
                .unwrap();
            if output
                .chunks
                .iter()
                .any(|chunk| chunk.text.contains("40 120"))
            {
                return;
            }
            thread::sleep(Duration::from_millis(20));
        }
        panic!("Linux PTY resize was not visible through stty size");
    }
}
