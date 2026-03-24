package contracts

type AttachmentDTO struct {
	ObjectKey   string `json:"object_key"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type MessageReferenceDTO struct {
	ID          uint64         `json:"id"`
	SenderID    uint64         `json:"sender_id"`
	MessageType string         `json:"message_type"`
	Content     string         `json:"content"`
	Attachment  *AttachmentDTO `json:"attachment,omitempty"`
	RecalledAt  string         `json:"recalled_at"`
}

type UploadPresignRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type UploadPresignResponse struct {
	ObjectKey  string            `json:"object_key"`
	UploadPath string            `json:"upload_path"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	PublicURL  string            `json:"public_url"`
}

type UploadCompleteRequest struct {
	ObjectKey   string `json:"object_key"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type UploadCompleteResponse struct {
	Attachment AttachmentDTO `json:"attachment"`
}
