package platformhttp

import (
	"errors"
	"net/url"
	"strings"
)

func OpenExternalURL(target string) error {
	target = strings.TrimSpace(target)
	parsed, err := url.Parse(target)
	if err != nil || parsed.Host == "" {
		return errors.New("外部链接无效")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("只允许打开 http/https 外部链接")
	}
	return openSystemBrowser(target)
}
