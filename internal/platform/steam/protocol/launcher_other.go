//go:build !windows

package protocol

import (
	"context"
	"errors"
)

func openURI(context.Context, string) error {
	return errors.New("Steam client protocol validation is currently supported on Windows only")
}
