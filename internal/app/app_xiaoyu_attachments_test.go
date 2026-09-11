package app

import (
	"encoding/base64"
	"testing"
)

func TestNormalizeXiaoYuAttachments(t *testing.T) {
	items, err := normalizeXiaoYuAttachments([]XiaoYuAttachmentRequest{{Name: "shot.png", MediaType: "image/png", DataBase64: base64.StdEncoding.EncodeToString([]byte{1, 2, 3})}})
	if err != nil || len(items) != 1 || items[0].ID == "" || len(items[0].Data) != 3 {
		t.Fatalf("unexpected attachment: %+v err=%v", items, err)
	}
	if _, err := normalizeXiaoYuAttachments([]XiaoYuAttachmentRequest{{Name: "x.txt", MediaType: "text/plain", DataBase64: base64.StdEncoding.EncodeToString([]byte("x"))}}); err == nil {
		t.Fatal("unsupported attachment should fail")
	}
}
