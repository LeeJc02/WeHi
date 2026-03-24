package upload

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeJc02/WeHi/backend/internal/platform/apperr"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type Service struct {
	dir string
}

func NewService(dir string) *Service {
	return &Service{dir: dir}
}

func (s *Service) EnsureReady() error {
	return os.MkdirAll(s.dir, 0o755)
}

func (s *Service) Presign(req contracts.UploadPresignRequest) (*contracts.UploadPresignResponse, error) {
	filename := strings.TrimSpace(req.Filename)
	if filename == "" || req.SizeBytes <= 0 {
		return nil, apperr.BadRequest("INVALID_UPLOAD_REQUEST", "filename and size_bytes are required")
	}
	key := randomKey() + sanitizeExt(filename)
	return &contracts.UploadPresignResponse{
		ObjectKey:  key,
		UploadPath: "/api/v1/uploads/object/" + key,
		Method:     "PUT",
		Headers:    map[string]string{},
		PublicURL:  "/uploads/" + key,
	}, nil
}

func (s *Service) PutObject(objectKey string, reader io.Reader) error {
	if err := s.EnsureReady(); err != nil {
		return err
	}
	path, err := s.objectPath(objectKey)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func (s *Service) Complete(req contracts.UploadCompleteRequest) (*contracts.UploadCompleteResponse, error) {
	path, err := s.objectPath(req.ObjectKey)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.NotFound("UPLOAD_NOT_FOUND", "uploaded object not found")
		}
		return nil, err
	}
	if req.SizeBytes > 0 && info.Size() != req.SizeBytes {
		return nil, apperr.BadRequest("UPLOAD_SIZE_MISMATCH", "uploaded object size mismatch")
	}
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(req.Filename))
	}
	return &contracts.UploadCompleteResponse{
		Attachment: contracts.AttachmentDTO{
			ObjectKey:   req.ObjectKey,
			URL:         "/uploads/" + req.ObjectKey,
			Filename:    strings.TrimSpace(req.Filename),
			ContentType: contentType,
			SizeBytes:   info.Size(),
		},
	}, nil
}

func (s *Service) Open(objectKey string) (*os.File, string, error) {
	path, err := s.objectPath(objectKey)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", apperr.NotFound("UPLOAD_NOT_FOUND", "uploaded object not found")
		}
		return nil, "", err
	}
	return file, mime.TypeByExtension(filepath.Ext(objectKey)), nil
}

func (s *Service) objectPath(objectKey string) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" || strings.Contains(objectKey, "/") || strings.Contains(objectKey, "\\") {
		return "", apperr.BadRequest("INVALID_UPLOAD_OBJECT_KEY", "invalid upload object key")
	}
	return filepath.Join(s.dir, objectKey), nil
}

func randomKey() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "upload"
	}
	return hex.EncodeToString(buf)
}

func sanitizeExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" || len(ext) > 10 || strings.Contains(ext, "..") {
		return ""
	}
	return ext
}
