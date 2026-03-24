package upload

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

func TestServicePresignPutCompleteAndOpen(t *testing.T) {
	dir := t.TempDir()
	service := NewService(dir)

	presigned, err := service.Presign(contracts.UploadPresignRequest{
		Filename:    "photo.png",
		ContentType: "image/png",
		SizeBytes:   4,
	})
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if filepath.Ext(presigned.ObjectKey) != ".png" {
		t.Fatalf("expected png extension, got %q", presigned.ObjectKey)
	}

	if err := service.PutObject(presigned.ObjectKey, bytes.NewBufferString("data")); err != nil {
		t.Fatalf("put object: %v", err)
	}

	completed, err := service.Complete(contracts.UploadCompleteRequest{
		ObjectKey:   presigned.ObjectKey,
		Filename:    "photo.png",
		ContentType: "image/png",
		SizeBytes:   4,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Attachment.URL != "/uploads/"+presigned.ObjectKey {
		t.Fatalf("unexpected url: %q", completed.Attachment.URL)
	}

	file, contentType, err := service.Open(presigned.ObjectKey)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "data" {
		t.Fatalf("unexpected body: %q", string(body))
	}
	if contentType != "image/png" {
		t.Fatalf("unexpected content type: %q", contentType)
	}
}

func TestServiceRejectsInvalidObjectKey(t *testing.T) {
	service := NewService(t.TempDir())

	if err := service.PutObject("../escape", bytes.NewBufferString("data")); err == nil {
		t.Fatal("expected invalid object key error")
	}
}
