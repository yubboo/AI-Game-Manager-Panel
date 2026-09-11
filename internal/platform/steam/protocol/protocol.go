package protocol

import (
	"context"
	"fmt"
)

type Launcher interface {
	Open(context.Context, string) error
}

type SystemLauncher struct{}

func NewSystemLauncher() *SystemLauncher { return &SystemLauncher{} }

func BuildValidateURI(appID uint32) string {
	return fmt.Sprintf("steam://validate/%d", appID)
}

func BuildInstallURI(appID uint32) string {
	return fmt.Sprintf("steam://install/%d", appID)
}

func (s *SystemLauncher) Open(ctx context.Context, uri string) error {
	return openURI(ctx, uri)
}
