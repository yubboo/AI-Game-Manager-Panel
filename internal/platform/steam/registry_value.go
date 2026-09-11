package steam

import (
	"encoding/binary"
	"unicode/utf16"
)

// decodeRegistryUTF16LE decodes a Windows REG_SZ/REG_EXPAND_SZ byte buffer.
// Registry strings are UTF-16LE and usually include a trailing NUL.
func decodeRegistryUTF16LE(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	values := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		value := binary.LittleEndian.Uint16(data[i : i+2])
		if value == 0 {
			break
		}
		values = append(values, value)
	}
	return string(utf16.Decode(values))
}
