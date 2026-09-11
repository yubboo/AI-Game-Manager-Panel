//go:build !windows

package steam

// ClientReadBytes is currently available only on Windows, which is the desktop
// platform where AI Game Manager Panel launches the user's Steam client through steam://.
func ClientReadBytes() (uint64, bool) {
	return 0, false
}
