use anyhow::{Context, Result, bail};
use std::ffi::{OsStr, c_void};
use std::fs::File;
use std::io;
use std::mem::{size_of, zeroed};
use std::os::windows::ffi::OsStrExt;
use std::os::windows::io::{AsRawHandle, FromRawHandle, RawHandle};
use std::path::Path;
use std::ptr::{null, null_mut};
use windows_sys::Win32::Foundation::{CloseHandle, HANDLE};
use windows_sys::Win32::System::Console::{
    COORD, ClosePseudoConsole, CreatePseudoConsole, HPCON, ResizePseudoConsole,
};
use windows_sys::Win32::System::Pipes::CreatePipe;
use windows_sys::Win32::System::Threading::{
    CREATE_UNICODE_ENVIRONMENT, CreateProcessW, DeleteProcThreadAttributeList,
    EXTENDED_STARTUPINFO_PRESENT, GetExitCodeProcess, InitializeProcThreadAttributeList,
    LPPROC_THREAD_ATTRIBUTE_LIST, PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, PROCESS_INFORMATION,
    STARTUPINFOEXW, TerminateProcess, UpdateProcThreadAttribute,
};

const STILL_ACTIVE: u32 = 259;

struct PseudoConsole(HPCON);

unsafe impl Send for PseudoConsole {}

impl Drop for PseudoConsole {
    fn drop(&mut self) {
        unsafe { ClosePseudoConsole(self.0) };
    }
}

struct ProcessHandle(HANDLE);

unsafe impl Send for ProcessHandle {}

impl Drop for ProcessHandle {
    fn drop(&mut self) {
        unsafe {
            CloseHandle(self.0);
        }
    }
}

struct AttributeList {
    _storage: Vec<usize>,
    ptr: LPPROC_THREAD_ATTRIBUTE_LIST,
}

impl AttributeList {
    fn new(pseudo_console: HPCON) -> Result<Self> {
        let mut bytes = 0_usize;
        unsafe {
            InitializeProcThreadAttributeList(null_mut(), 1, 0, &mut bytes);
        }
        if bytes == 0 {
            return Err(io::Error::last_os_error())
                .context("cannot size ConPTY process attribute list");
        }

        let words = bytes.div_ceil(size_of::<usize>());
        let mut storage = vec![0_usize; words];
        let ptr: LPPROC_THREAD_ATTRIBUTE_LIST = storage.as_mut_ptr().cast();
        if unsafe { InitializeProcThreadAttributeList(ptr, 1, 0, &mut bytes) } == 0 {
            return Err(io::Error::last_os_error())
                .context("cannot initialize ConPTY process attribute list");
        }
        if unsafe {
            UpdateProcThreadAttribute(
                ptr,
                0,
                PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE as usize,
                pseudo_console as *const c_void,
                size_of::<HPCON>(),
                null_mut(),
                null(),
            )
        } == 0
        {
            unsafe { DeleteProcThreadAttributeList(ptr) };
            return Err(io::Error::last_os_error())
                .context("cannot attach ConPTY to process attribute list");
        }

        Ok(Self {
            _storage: storage,
            ptr,
        })
    }
}

impl Drop for AttributeList {
    fn drop(&mut self) {
        unsafe { DeleteProcThreadAttributeList(self.ptr) };
    }
}

pub struct WindowsPtyProcess {
    process: ProcessHandle,
    pseudo_console: PseudoConsole,
    pid: u32,
}

pub struct WindowsPtySpawn {
    pub process: WindowsPtyProcess,
    pub reader: File,
    pub writer: File,
}

impl WindowsPtyProcess {
    pub fn pid(&self) -> u32 {
        self.pid
    }

    pub fn try_wait(&mut self) -> io::Result<Option<i32>> {
        let mut code = 0_u32;
        if unsafe { GetExitCodeProcess(self.process.0, &mut code) } == 0 {
            return Err(io::Error::last_os_error());
        }
        if code == STILL_ACTIVE {
            Ok(None)
        } else {
            Ok(Some(code as i32))
        }
    }

    pub fn kill(&mut self) -> io::Result<()> {
        if unsafe { TerminateProcess(self.process.0, 1) } == 0 {
            let error = io::Error::last_os_error();
            if self.try_wait()?.is_none() {
                return Err(error);
            }
        }
        Ok(())
    }

    pub fn resize(&self, rows: u16, cols: u16) -> io::Result<()> {
        let size = coord(rows, cols)?;
        let result = unsafe { ResizePseudoConsole(self.pseudo_console.0, size) };
        if result < 0 {
            return Err(io::Error::other(format!(
                "ResizePseudoConsole failed with HRESULT 0x{:08X}",
                result as u32
            )));
        }
        Ok(())
    }
}

pub fn spawn(
    executable: &str,
    arguments: &[String],
    cwd: &Path,
    rows: u16,
    cols: u16,
) -> Result<WindowsPtySpawn> {
    let size = coord(rows, cols)?;
    let (pty_input_read, input_write) = create_pipe().context("cannot create ConPTY input pipe")?;
    let (output_read, pty_output_write) =
        create_pipe().context("cannot create ConPTY output pipe")?;

    let mut pseudo_console: HPCON = 0;
    let result = unsafe {
        CreatePseudoConsole(
            size,
            pty_input_read.as_raw_handle() as HANDLE,
            pty_output_write.as_raw_handle() as HANDLE,
            0,
            &mut pseudo_console,
        )
    };
    if result < 0 {
        bail!(
            "CreatePseudoConsole failed with HRESULT 0x{:08X}",
            result as u32
        );
    }
    let pseudo_console = PseudoConsole(pseudo_console);
    let attributes = AttributeList::new(pseudo_console.0)?;

    let mut startup: STARTUPINFOEXW = unsafe { zeroed() };
    startup.StartupInfo.cb = size_of::<STARTUPINFOEXW>() as u32;
    startup.lpAttributeList = attributes.ptr;

    let mut process_info: PROCESS_INFORMATION = unsafe { zeroed() };
    let mut command_line = command_line(executable, arguments);
    let current_directory = wide_null(cwd.as_os_str());
    let created = unsafe {
        CreateProcessW(
            null(),
            command_line.as_mut_ptr(),
            null(),
            null(),
            0,
            EXTENDED_STARTUPINFO_PRESENT | CREATE_UNICODE_ENVIRONMENT,
            null(),
            current_directory.as_ptr(),
            &startup.StartupInfo,
            &mut process_info,
        )
    };
    drop(attributes);
    drop(pty_input_read);
    drop(pty_output_write);
    if created == 0 {
        return Err(io::Error::last_os_error())
            .with_context(|| format!("cannot start ConPTY executable: {executable}"));
    }

    unsafe {
        CloseHandle(process_info.hThread);
    }

    Ok(WindowsPtySpawn {
        process: WindowsPtyProcess {
            process: ProcessHandle(process_info.hProcess),
            pseudo_console,
            pid: process_info.dwProcessId,
        },
        reader: output_read,
        writer: input_write,
    })
}

fn create_pipe() -> io::Result<(File, File)> {
    let mut read: HANDLE = null_mut();
    let mut write: HANDLE = null_mut();
    if unsafe { CreatePipe(&mut read, &mut write, null(), 0) } == 0 {
        return Err(io::Error::last_os_error());
    }
    Ok((
        unsafe { File::from_raw_handle(read as RawHandle) },
        unsafe { File::from_raw_handle(write as RawHandle) },
    ))
}

fn coord(rows: u16, cols: u16) -> io::Result<COORD> {
    if rows > i16::MAX as u16 || cols > i16::MAX as u16 {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "ConPTY rows and cols must fit in signed 16-bit coordinates",
        ));
    }
    Ok(COORD {
        X: cols as i16,
        Y: rows as i16,
    })
}

fn command_line(executable: &str, arguments: &[String]) -> Vec<u16> {
    let mut command = quote_windows_argument(executable);
    for argument in arguments {
        command.push(' ');
        command.push_str(&quote_windows_argument(argument));
    }
    wide_null(OsStr::new(&command))
}

fn wide_null(value: &OsStr) -> Vec<u16> {
    value.encode_wide().chain(std::iter::once(0)).collect()
}

fn quote_windows_argument(value: &str) -> String {
    if !value.is_empty()
        && !value
            .chars()
            .any(|character| character.is_whitespace() || character == '"')
    {
        return value.to_string();
    }

    let mut quoted = String::with_capacity(value.len() + 2);
    quoted.push('"');
    let mut backslashes = 0_usize;
    for character in value.chars() {
        match character {
            '\\' => backslashes += 1,
            '"' => {
                quoted.push_str(&"\\".repeat(backslashes * 2 + 1));
                quoted.push('"');
                backslashes = 0;
            }
            _ => {
                quoted.push_str(&"\\".repeat(backslashes));
                backslashes = 0;
                quoted.push(character);
            }
        }
    }
    quoted.push_str(&"\\".repeat(backslashes * 2));
    quoted.push('"');
    quoted
}

#[cfg(test)]
mod tests {
    use super::quote_windows_argument;

    #[test]
    fn windows_argument_quoting_preserves_spaces_quotes_and_trailing_slashes() {
        assert_eq!(quote_windows_argument("plain"), "plain");
        assert_eq!(quote_windows_argument("two words"), "\"two words\"");
        assert_eq!(quote_windows_argument(""), "\"\"");
        assert_eq!(quote_windows_argument("a\\\"b"), "\"a\\\\\\\"b\"");
        assert_eq!(
            quote_windows_argument("C:\\path with space\\"),
            "\"C:\\path with space\\\\\""
        );
    }
}
