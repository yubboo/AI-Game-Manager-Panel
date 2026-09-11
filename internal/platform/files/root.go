package platformfiles

import (
	"os"
	"path/filepath"
)

// ResolveRoot 返回 AI Game Manager Panel 可写运行根目录。
//
// 优先级：
//  1. AGMP_ROOT：安装器/启动器显式指定的可写目录；
//  2. 开发源码根目录（go.mod + frontend）；
//  3. 便携版可执行文件所在目录；
//  4. 当前工作目录。
//
// Linux .deb 安装后程序本体位于只读系统目录，因此启动脚本会通过
// AGMP_ROOT 指向当前用户的 XDG 数据目录；Windows 便携版/安装版仍可
// 直接使用应用目录作为 Root；configs/ 保持稳定，0.1.65 新默认可写数据收口到 runtime/。
func ResolveRoot() string {
	if value := os.Getenv("AGMP_ROOT"); value != "" {
		return filepath.Clean(value)
	}
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	return resolveRoot(cwd, exe)
}

func resolveRoot(cwd, executable string) string {
	if isProjectRoot(cwd) {
		return filepath.Clean(cwd)
	}

	dir := ""
	if executable != "" {
		dir = filepath.Dir(executable)
		probe := dir
		for i := 0; i < 6; i++ {
			if isProjectRoot(probe) {
				return filepath.Clean(probe)
			}
			parent := filepath.Dir(probe)
			if parent == probe {
				break
			}
			probe = parent
		}
	}

	if dir != "" {
		return filepath.Clean(dir)
	}
	if cwd != "" {
		return filepath.Clean(cwd)
	}
	return "."
}

func isProjectRoot(dir string) bool {
	if dir == "" {
		return false
	}
	if info, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil || info.IsDir() {
		return false
	}
	if info, err := os.Stat(filepath.Join(dir, "frontend")); err != nil || !info.IsDir() {
		return false
	}
	return true
}
