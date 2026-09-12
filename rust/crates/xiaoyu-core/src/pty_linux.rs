use anyhow::{Context, Result, bail};
use std::ffi::CStr;
use std::fs::{File, OpenOptions};
use std::io;
use std::os::fd::{AsRawFd, FromRawFd};
use std::os::raw::{c_char, c_int, c_ulong};
use std::os::unix::ffi::OsStringExt;
use std::os::unix::process::CommandExt;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};

const O_RDWR: c_int = 0x0002;
const O_NOCTTY: c_int = 0x0100;
const F_SETFD: c_int = 2;
const FD_CLOEXEC: c_int = 1;
const SIGKILL: c_int = 9;
const TIOCSCTTY: c_ulong = 0x540E;
const TIOCSWINSZ: c_ulong = 0x5414;

#[repr(C)]
struct Winsize {
    ws_row: u16,
    ws_col: u16,
    ws_xpixel: u16,
    ws_ypixel: u16,
}

unsafe extern "C" {
    fn posix_openpt(flags: c_int) -> c_int;
    fn grantpt(fd: c_int) -> c_int;
    fn unlockpt(fd: c_int) -> c_int;
    fn ptsname_r(fd: c_int, buf: *mut c_char, buflen: usize) -> c_int;
    fn setsid() -> c_int;
    fn ioctl(fd: c_int, request: c_ulong, ...) -> c_int;
    fn fcntl(fd: c_int, command: c_int, ...) -> c_int;
    #[link_name = "kill"]
    fn c_kill(pid: c_int, signal: c_int) -> c_int;
}

pub struct LinuxPtyProcess {
    child: Child,
    control: File,
}

pub struct LinuxPtySpawn {
    pub process: LinuxPtyProcess,
    pub reader: File,
    pub writer: File,
}

impl LinuxPtyProcess {
    pub fn pid(&self) -> u32 {
        self.child.id()
    }

    pub fn try_wait(&mut self) -> io::Result<Option<i32>> {
        self.child
            .try_wait()
            .map(|status| status.map(|status| status.code().unwrap_or_default()))
    }

    pub fn kill(&mut self) -> io::Result<()> {
        let pid = self.child.id();
        if pid <= c_int::MAX as u32 {
            let result = unsafe { c_kill(-(pid as c_int), SIGKILL) };
            if result == 0 {
                return Ok(());
            }
            let error = io::Error::last_os_error();
            if error.raw_os_error() != Some(3) {
                return Err(error);
            }
        }
        self.child.kill()
    }

    pub fn resize(&self, rows: u16, cols: u16) -> io::Result<()> {
        set_winsize(self.control.as_raw_fd(), rows, cols)
    }
}

pub fn spawn(
    executable: &str,
    arguments: &[String],
    cwd: &Path,
    rows: u16,
    cols: u16,
) -> Result<LinuxPtySpawn> {
    let master_fd = unsafe { posix_openpt(O_RDWR | O_NOCTTY) };
    if master_fd < 0 {
        return Err(io::Error::last_os_error()).context("posix_openpt failed");
    }
    let master = unsafe { File::from_raw_fd(master_fd) };
    if unsafe { fcntl(master.as_raw_fd(), F_SETFD, FD_CLOEXEC) } < 0 {
        return Err(io::Error::last_os_error()).context("cannot set PTY master close-on-exec");
    }

    if unsafe { grantpt(master.as_raw_fd()) } != 0 {
        return Err(io::Error::last_os_error()).context("grantpt failed");
    }
    if unsafe { unlockpt(master.as_raw_fd()) } != 0 {
        return Err(io::Error::last_os_error()).context("unlockpt failed");
    }

    let slave_path = slave_path(master.as_raw_fd())?;
    let slave = OpenOptions::new()
        .read(true)
        .write(true)
        .open(&slave_path)
        .with_context(|| format!("cannot open PTY slave: {}", slave_path.display()))?;

    set_winsize(master.as_raw_fd(), rows, cols).context("cannot set initial PTY size")?;

    let stdin = slave.try_clone()?;
    let stdout = slave.try_clone()?;
    let stderr = slave.try_clone()?;
    let mut command = Command::new(executable);
    command
        .args(arguments)
        .current_dir(cwd)
        .env("TERM", "xterm-256color")
        .stdin(Stdio::from(stdin))
        .stdout(Stdio::from(stdout))
        .stderr(Stdio::from(stderr));

    unsafe {
        command.pre_exec(|| {
            if setsid() < 0 {
                return Err(io::Error::last_os_error());
            }
            if ioctl(0, TIOCSCTTY, 0) < 0 {
                return Err(io::Error::last_os_error());
            }
            Ok(())
        });
    }

    let child = command
        .spawn()
        .with_context(|| format!("cannot start PTY executable: {executable}"))?;
    drop(slave);

    Ok(LinuxPtySpawn {
        process: LinuxPtyProcess {
            child,
            control: master.try_clone()?,
        },
        reader: master.try_clone()?,
        writer: master,
    })
}

fn slave_path(master_fd: c_int) -> Result<PathBuf> {
    let mut buffer = vec![0_i8; 4096];
    let result = unsafe { ptsname_r(master_fd, buffer.as_mut_ptr(), buffer.len()) };
    if result != 0 {
        bail!("ptsname_r failed with errno {result}");
    }
    let name = unsafe { CStr::from_ptr(buffer.as_ptr()) };
    Ok(PathBuf::from(std::ffi::OsString::from_vec(
        name.to_bytes().to_vec(),
    )))
}

fn set_winsize(fd: c_int, rows: u16, cols: u16) -> io::Result<()> {
    let size = Winsize {
        ws_row: rows,
        ws_col: cols,
        ws_xpixel: 0,
        ws_ypixel: 0,
    };
    if unsafe { ioctl(fd, TIOCSWINSZ, &size) } < 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}
