package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
)

const workspaceHashNamespace = "token-router-workspace-v1\x00"

type workspaceError struct {
	status  int
	message string
}

func (e *workspaceError) Error() string { return e.message }

func workspaceErrorWithStatus(status int, message string) error {
	return &workspaceError{status: status, message: message}
}

func WorkspaceHTTPStatus(err error) int {
	var target *workspaceError
	if errors.As(err, &target) {
		return target.status
	}
	return 500
}

func workspaceDirectoryKey(accountID string) string {
	digest := sha256.Sum256([]byte(workspaceHashNamespace + accountID))
	return hex.EncodeToString(digest[:])
}

func workspaceBaseRoot() (string, error) {
	configured := strings.TrimSpace(config.ApplicationConfig.Chat.BuiltinTools.WorkspaceDirectory)
	if configured == "" {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	root, err := filepath.Abs(configured)
	if err != nil {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	if err = os.MkdirAll(root, 0o750); err != nil {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	return root, nil
}

func personalWorkspaceRoot(ctx context.Context) (string, error) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return "", workspaceErrorWithStatus(401, "authenticated account is missing")
	}
	base, err := workspaceBaseRoot()
	if err != nil {
		return "", err
	}
	root := filepath.Join(base, workspaceDirectoryKey(accountID))
	if err = os.MkdirAll(root, 0o750); err != nil {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil || !pathWithin(base, root) {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	return root, nil
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func resolveWorkspacePath(ctx context.Context, value string, allowRoot, allowMissing bool) (string, string, error) {
	if strings.ContainsRune(value, '\x00') || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
	}
	cleaned := filepath.Clean(filepath.FromSlash(value))
	if cleaned == "." && !allowRoot {
		return "", "", workspaceErrorWithStatus(400, "path must identify a file")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
	}
	root, err := personalWorkspaceRoot(ctx)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(root, cleaned)
	if !pathWithin(root, target) {
		return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
	}
	if evaluated, evalErr := filepath.EvalSymlinks(target); evalErr == nil {
		if !pathWithin(root, evaluated) {
			return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
		}
	} else if !allowMissing {
		if errors.Is(evalErr, os.ErrNotExist) {
			return "", "", workspaceErrorWithStatus(404, "workspace entry not found")
		}
		return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
	} else {
		ancestor := filepath.Dir(target)
		for {
			evaluated, ancestorErr := filepath.EvalSymlinks(ancestor)
			if ancestorErr == nil {
				if !pathWithin(root, evaluated) {
					return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
				}
				break
			}
			parent := filepath.Dir(ancestor)
			if parent == ancestor || !pathWithin(root, parent) {
				return "", "", workspaceErrorWithStatus(400, "invalid workspace path")
			}
			ancestor = parent
		}
	}
	relative, _ := filepath.Rel(root, target)
	if relative == "." {
		relative = ""
	}
	return target, filepath.ToSlash(relative), nil
}

func workspaceEntry(relative string, info os.FileInfo) dtos.ChatWorkspaceEntry {
	kind := "file"
	if info.Mode()&os.ModeSymlink != 0 {
		kind = "symlink"
	} else if info.IsDir() {
		kind = "directory"
	}
	return dtos.ChatWorkspaceEntry{
		Name: filepath.Base(relative), Path: filepath.ToSlash(relative), Type: kind,
		Size: info.Size(), UpdatedAt: info.ModTime(),
	}
}

func (ChatService) ListWorkspace(ctx context.Context, path string) (dtos.ChatWorkspaceListing, error) {
	settings := config.ApplicationConfig.Chat.BuiltinTools
	if !settings.Enabled {
		return dtos.ChatWorkspaceListing{}, workspaceErrorWithStatus(503, "workspace is disabled")
	}
	target, relative, err := resolveWorkspacePath(ctx, path, true, false)
	if err != nil {
		return dtos.ChatWorkspaceListing{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return dtos.ChatWorkspaceListing{}, workspaceErrorWithStatus(404, "workspace entry not found")
	}
	if !info.IsDir() {
		return dtos.ChatWorkspaceListing{}, workspaceErrorWithStatus(409, "workspace entry is not a directory")
	}
	items, err := os.ReadDir(target)
	if err != nil {
		return dtos.ChatWorkspaceListing{}, workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	entries := make([]dtos.ChatWorkspaceEntry, 0, len(items))
	for _, item := range items {
		itemInfo, infoErr := item.Info()
		if infoErr != nil {
			continue
		}
		entryRelative := filepath.Join(relative, item.Name())
		entries = append(entries, workspaceEntry(entryRelative, itemInfo))
	}
	sort.Slice(entries, func(i, j int) bool {
		leftDirectory := entries[i].Type == "directory"
		rightDirectory := entries[j].Type == "directory"
		if leftDirectory != rightDirectory {
			return leftDirectory
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return dtos.ChatWorkspaceListing{Path: relative, Entries: entries, MaxUploadBytes: settings.MaxUploadBytes}, nil
}

func validateWorkspaceFilename(filename string) error {
	if filename == "" || filename == "." || filename == ".." || strings.ContainsAny(filename, "/\\\x00") || filepath.Base(filename) != filename {
		return workspaceErrorWithStatus(400, "invalid upload filename")
	}
	for _, character := range filename {
		if character < 0x20 || character == 0x7f {
			return workspaceErrorWithStatus(400, "invalid upload filename")
		}
	}
	return nil
}

func (ChatService) UploadWorkspaceFile(ctx context.Context, directory, filename string, source io.Reader) (dtos.ChatWorkspaceEntry, error) {
	settings := config.ApplicationConfig.Chat.BuiltinTools
	if !settings.Enabled {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "workspace is disabled")
	}
	if err := validateWorkspaceFilename(filename); err != nil {
		return dtos.ChatWorkspaceEntry{}, err
	}
	directoryTarget, directoryRelative, err := resolveWorkspacePath(ctx, directory, true, false)
	if err != nil {
		return dtos.ChatWorkspaceEntry{}, err
	}
	directoryInfo, err := os.Stat(directoryTarget)
	if err != nil || !directoryInfo.IsDir() {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(409, "upload target is not a directory")
	}
	target, relative, err := resolveWorkspacePath(ctx, filepath.Join(directoryRelative, filename), false, true)
	if err != nil {
		return dtos.ChatWorkspaceEntry{}, err
	}
	if existing, statErr := os.Lstat(target); statErr == nil && !existing.Mode().IsRegular() {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(409, "upload target is not a regular file")
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	temporary, err := os.CreateTemp(directoryTarget, ".token-router-upload-*")
	if err != nil {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "workspace is unavailable")
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	limit := settings.MaxUploadBytes
	written, copyErr := io.Copy(temporary, io.LimitReader(source, limit+1))
	if copyErr != nil {
		_ = temporary.Close()
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "upload failed")
	}
	if written > limit {
		_ = temporary.Close()
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(413, "file exceeds the upload size limit")
	}
	if err = temporary.Chmod(0o640); err != nil {
		_ = temporary.Close()
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "upload failed")
	}
	if err = temporary.Close(); err != nil {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "upload failed")
	}
	if err = os.Rename(temporaryName, target); err != nil {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "upload failed")
	}
	info, err := os.Stat(target)
	if err != nil {
		return dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "upload failed")
	}
	return workspaceEntry(relative, info), nil
}

func (ChatService) OpenWorkspaceFile(ctx context.Context, path string) (*os.File, dtos.ChatWorkspaceEntry, error) {
	if !config.ApplicationConfig.Chat.BuiltinTools.Enabled {
		return nil, dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(503, "workspace is disabled")
	}
	target, relative, err := resolveWorkspacePath(ctx, path, false, false)
	if err != nil {
		return nil, dtos.ChatWorkspaceEntry{}, err
	}
	linkInfo, err := os.Lstat(target)
	if err != nil {
		return nil, dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(404, "workspace entry not found")
	}
	if !linkInfo.Mode().IsRegular() {
		return nil, dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(409, "workspace entry is not a regular file")
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(404, "workspace entry not found")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, dtos.ChatWorkspaceEntry{}, workspaceErrorWithStatus(409, "workspace entry is not a regular file")
	}
	return file, workspaceEntry(relative, info), nil
}
