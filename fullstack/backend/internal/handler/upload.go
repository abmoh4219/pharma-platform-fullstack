package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/middleware"
)

var allowedUploadExt = map[string]struct{}{
	".pdf":  {},
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".doc":  {},
	".docx": {},
	".xls":  {},
	".xlsx": {},
	".csv":  {},
	".txt":  {},
}

type uploadInitRequest struct {
	ModuleName   string `json:"module_name" binding:"required,oneof=case_ledgers candidates qualifications restrictions positions"`
	RecordID     int64  `json:"record_id" binding:"required,gte=1"`
	OriginalName string `json:"original_name" binding:"required,min=1,max=255"`
	MimeType     string `json:"mime_type" binding:"omitempty,max=128"`
	TotalChunks  int    `json:"total_chunks" binding:"required,gte=1,lte=5000"`
	FileSize     int64  `json:"file_size" binding:"required,gte=1"`
}

func (a *API) InitiateUpload(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req uploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid upload init payload")
		return
	}

	req.ModuleName = strings.TrimSpace(req.ModuleName)
	req.OriginalName = strings.TrimSpace(req.OriginalName)
	req.MimeType = strings.TrimSpace(req.MimeType)
	if req.ModuleName == "" || req.RecordID <= 0 || req.OriginalName == "" || req.TotalChunks <= 0 || req.FileSize <= 0 {
		badRequest(c, "INVALID_UPLOAD_REQUEST", "module_name, record_id, original_name, total_chunks, and file_size are required")
		return
	}

	ext := strings.ToLower(filepath.Ext(req.OriginalName))
	if _, ok := allowedUploadExt[ext]; !ok {
		badRequest(c, "UNSUPPORTED_FILE_FORMAT", "file format is not allowed")
		return
	}
	maxUploadBytes := a.cfg.MaxUploadMB * 1024 * 1024
	if req.FileSize > maxUploadBytes {
		badRequest(c, "FILE_TOO_LARGE", fmt.Sprintf("file exceeds max size of %d MB", a.cfg.MaxUploadMB))
		return
	}
	if !a.canAccessModuleRecord(user, req.ModuleName, req.RecordID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "record is outside your data scope")
		return
	}

	uploadID, err := newUploadID()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_ID_ERROR", "failed to create upload id")
		return
	}
	tmpDir := filepath.Join(a.cfg.UploadTmpDir, uploadID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_INIT_ERROR", "failed to create upload temp folder")
		return
	}

	_, err = a.db.Exec(`
		INSERT INTO upload_sessions
		(id, module_name, record_id, original_name, mime_type, total_chunks, file_size, status, uploaded_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'in_progress', ?)
	`, uploadID, req.ModuleName, req.RecordID, req.OriginalName, req.MimeType, req.TotalChunks, req.FileSize, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create upload session")
		return
	}

	a.logAudit(c, user.ID, "files.upload.init", "upload_sessions", uploadID, req)
	writeSuccess(c, http.StatusCreated, gin.H{"upload_id": uploadID})
}

func (a *API) UploadChunk(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	uploadID := strings.TrimSpace(c.PostForm("upload_id"))
	chunkIndexStr := strings.TrimSpace(c.PostForm("chunk_index"))
	if uploadID == "" || chunkIndexStr == "" {
		badRequest(c, "INVALID_CHUNK_REQUEST", "upload_id and chunk_index are required")
		return
	}
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		badRequest(c, "INVALID_CHUNK_INDEX", "chunk_index must be >= 0")
		return
	}

	session, err := a.loadUploadSession(uploadID)
	if err != nil {
		writeError(c, http.StatusNotFound, "UPLOAD_SESSION_NOT_FOUND", "upload session not found")
		return
	}
	if session.Status != "in_progress" {
		writeError(c, http.StatusConflict, "UPLOAD_NOT_ACTIVE", "upload session is not active")
		return
	}
	if session.UploadedBy != user.ID && user.Role != "system_admin" {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "upload session is outside your ownership")
		return
	}
	if !a.canAccessModuleRecord(user, session.ModuleName, session.RecordID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "record is outside your data scope")
		return
	}
	if chunkIndex >= session.TotalChunks {
		badRequest(c, "CHUNK_OUT_OF_RANGE", "chunk_index exceeds total_chunks")
		return
	}

	fh, err := c.FormFile("chunk")
	if err != nil {
		badRequest(c, "CHUNK_FILE_REQUIRED", "chunk file is required")
		return
	}
	chunkFile, err := fh.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "CHUNK_OPEN_ERROR", "failed to open chunk file")
		return
	}
	defer chunkFile.Close()

	tmpDir := filepath.Join(a.cfg.UploadTmpDir, uploadID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to create upload temp folder")
		return
	}

	chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%06d.part", chunkIndex))
	if _, err := os.Stat(chunkPath); err == nil {
		received, _ := countChunkFiles(tmpDir)
		writeSuccess(c, http.StatusOK, gin.H{
			"upload_id":        uploadID,
			"chunk_index":      chunkIndex,
			"already_uploaded": true,
			"received_chunks":  received,
			"total_chunks":     session.TotalChunks,
		})
		return
	}

	dst, err := os.Create(chunkPath)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to create chunk file")
		return
	}
	written, err := io.Copy(dst, io.LimitReader(chunkFile, a.cfg.MaxUploadMB*1024*1024+1))
	_ = dst.Close()
	if err != nil {
		_ = os.Remove(chunkPath)
		writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to store chunk")
		return
	}
	if written > a.cfg.MaxUploadMB*1024*1024 {
		_ = os.Remove(chunkPath)
		badRequest(c, "CHUNK_TOO_LARGE", fmt.Sprintf("chunk exceeds max size of %d MB", a.cfg.MaxUploadMB))
		return
	}

	received, _ := countChunkFiles(tmpDir)
	writeSuccess(c, http.StatusOK, gin.H{
		"upload_id":       uploadID,
		"chunk_index":     chunkIndex,
		"received_chunks": received,
		"total_chunks":    session.TotalChunks,
	})
}

type uploadCompleteRequest struct {
	UploadID string `json:"upload_id" binding:"required,min=1,max=64"`
}

func (a *API) CompleteUpload(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req uploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid complete upload payload")
		return
	}
	req.UploadID = strings.TrimSpace(req.UploadID)
	if req.UploadID == "" {
		badRequest(c, "UPLOAD_ID_REQUIRED", "upload_id is required")
		return
	}

	session, err := a.loadUploadSession(req.UploadID)
	if err != nil {
		writeError(c, http.StatusNotFound, "UPLOAD_SESSION_NOT_FOUND", "upload session not found")
		return
	}
	if !a.canAccessModuleRecord(user, session.ModuleName, session.RecordID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "record is outside your data scope")
		return
	}
	if session.Status != "in_progress" {
		writeError(c, http.StatusConflict, "UPLOAD_NOT_ACTIVE", "upload session is not active")
		return
	}
	if session.UploadedBy != user.ID && user.Role != "system_admin" {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "upload session is outside your ownership")
		return
	}

	tmpDir := filepath.Join(a.cfg.UploadTmpDir, session.ID)
	chunkFiles, err := filepath.Glob(filepath.Join(tmpDir, "chunk_*.part"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to list upload chunks")
		return
	}
	if len(chunkFiles) != session.TotalChunks {
		writeError(c, http.StatusConflict, "INCOMPLETE_UPLOAD", "not all chunks have been uploaded")
		return
	}
	sort.Strings(chunkFiles)

	datePath := time.Now().UTC().Format("2006/01")
	finalDir := filepath.Join(a.cfg.UploadDir, datePath)
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to create final upload folder")
		return
	}

	ext := strings.ToLower(filepath.Ext(session.OriginalName))
	storedName := fmt.Sprintf("%d-%s%s", time.Now().UTC().UnixNano(), session.ID, ext)
	finalPath := filepath.Join(finalDir, storedName)

	out, err := os.Create(finalPath)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to create final file")
		return
	}
	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	for _, chunkPath := range chunkFiles {
		in, err := os.Open(chunkPath)
		if err != nil {
			_ = out.Close()
			_ = os.Remove(finalPath)
			writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to open chunk")
			return
		}
		if _, err := io.Copy(writer, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			_ = os.Remove(finalPath)
			writeError(c, http.StatusInternalServerError, "UPLOAD_IO_ERROR", "failed to merge chunk")
			return
		}
		_ = in.Close()
	}
	_ = out.Close()

	hashHex := hex.EncodeToString(hasher.Sum(nil))

	var existingID int64
	var existingPath string
	if err := a.db.QueryRow(`SELECT id, file_path FROM attachments WHERE sha256 = ? LIMIT 1`, hashHex).Scan(&existingID, &existingPath); err == nil {
		_ = os.Remove(finalPath)
		_, _ = a.db.Exec(`UPDATE upload_sessions SET status = 'completed' WHERE id = ?`, session.ID)
		_ = os.RemoveAll(tmpDir)
		writeSuccess(c, http.StatusOK, gin.H{
			"deduplicated":  true,
			"attachment_id": existingID,
			"existing_path": existingPath,
			"sha256":        hashHex,
		})
		return
	}

	relativePath, err := filepath.Rel(a.cfg.UploadDir, finalPath)
	if err != nil {
		relativePath = finalPath
	}

	res, err := a.db.Exec(`
		INSERT INTO attachments
		(module_name, record_id, original_name, stored_name, file_path, mime_type, file_size, sha256, uploaded_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ModuleName, session.RecordID, session.OriginalName, storedName, filepath.ToSlash(relativePath), session.MimeType, session.FileSize, hashHex, user.ID)
	if err != nil {
		_ = os.Remove(finalPath)
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to register attachment")
		return
	}
	attachmentID, _ := res.LastInsertId()

	_, _ = a.db.Exec(`UPDATE upload_sessions SET status = 'completed' WHERE id = ?`, session.ID)
	_ = os.RemoveAll(tmpDir)

	a.logAudit(c, user.ID, "files.upload.complete", "attachments", strconv.FormatInt(attachmentID, 10), gin.H{
		"module_name": session.ModuleName,
		"record_id":   session.RecordID,
		"sha256":      hashHex,
	})

	writeSuccess(c, http.StatusOK, gin.H{
		"deduplicated":  false,
		"attachment_id": attachmentID,
		"file_path":     filepath.ToSlash(relativePath),
		"sha256":        hashHex,
	})
}

func (a *API) GetUploadSession(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	uploadID := strings.TrimSpace(c.Param("id"))
	if uploadID == "" {
		badRequest(c, "UPLOAD_ID_REQUIRED", "upload id is required")
		return
	}
	session, err := a.loadUploadSession(uploadID)
	if err != nil {
		writeError(c, http.StatusNotFound, "UPLOAD_SESSION_NOT_FOUND", "upload session not found")
		return
	}
	if session.UploadedBy != user.ID && user.Role != "system_admin" {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "upload session is outside your ownership")
		return
	}
	if !a.canAccessModuleRecord(user, session.ModuleName, session.RecordID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "record is outside your data scope")
		return
	}
	tmpDir := filepath.Join(a.cfg.UploadTmpDir, uploadID)
	received, _ := countChunkFiles(tmpDir)
	writeSuccess(c, http.StatusOK, gin.H{
		"upload_id":       session.ID,
		"module_name":     session.ModuleName,
		"record_id":       session.RecordID,
		"original_name":   session.OriginalName,
		"total_chunks":    session.TotalChunks,
		"received_chunks": received,
		"status":          session.Status,
	})
}

func (a *API) DownloadAttachment(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid attachment id")
		return
	}

	var (
		moduleName   string
		recordID     int64
		originalName string
		storedPath   string
	)
	if err := a.db.QueryRow(`SELECT module_name, record_id, original_name, file_path FROM attachments WHERE id = ?`, id).Scan(&moduleName, &recordID, &originalName, &storedPath); err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "attachment not found")
		return
	}
	if !a.canAccessModuleRecord(user, moduleName, recordID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "attachment is outside your data scope")
		return
	}

	fullPath := filepath.Join(a.cfg.UploadDir, filepath.FromSlash(storedPath))
	if _, err := os.Stat(fullPath); err != nil {
		writeError(c, http.StatusNotFound, "FILE_NOT_FOUND", "stored file does not exist")
		return
	}
	c.FileAttachment(fullPath, originalName)
}

type uploadSession struct {
	ID           string
	ModuleName   string
	RecordID     int64
	OriginalName string
	MimeType     string
	TotalChunks  int
	FileSize     int64
	Status       string
	UploadedBy   int64
}

func (a *API) loadUploadSession(id string) (uploadSession, error) {
	var s uploadSession
	err := a.db.QueryRow(`
		SELECT id, module_name, record_id, original_name, mime_type, total_chunks, file_size, status, uploaded_by
		FROM upload_sessions
		WHERE id = ?
	`, id).Scan(&s.ID, &s.ModuleName, &s.RecordID, &s.OriginalName, &s.MimeType, &s.TotalChunks, &s.FileSize, &s.Status, &s.UploadedBy)
	if err != nil {
		return uploadSession{}, err
	}
	return s, nil
}

func countChunkFiles(tmpDir string) (int, error) {
	files, err := filepath.Glob(filepath.Join(tmpDir, "chunk_*.part"))
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

func newUploadID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("upl_%d_%s", time.Now().UTC().UnixNano(), hex.EncodeToString(buf)), nil
}

func (a *API) canAccessModuleRecord(user middleware.AuthUser, module string, recordID int64) bool {
	table := ""
	switch module {
	case "case_ledgers":
		table = "case_ledgers"
	case "candidates":
		table = "candidates"
	case "qualifications":
		table = "qualifications"
	case "restrictions":
		table = "restrictions"
	case "positions":
		table = "positions"
	default:
		return false
	}

	query := "SELECT COUNT(1) FROM " + table + " WHERE id = ?"
	args := []any{recordID}
	if user.Role != "system_admin" {
		query += " AND institution = ? AND department = ? AND team = ?"
		args = append(args, user.Institution, user.Department, user.Team)
	}

	var count int
	if err := a.db.QueryRow(query, args...).Scan(&count); err != nil {
		return false
	}
	return count > 0
}
