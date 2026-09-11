package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
)

type builtinPatchChunk struct {
	context string
	old     []string
	new     []string
}

type builtinPatchOperation struct {
	kind    string
	path    string
	content string
	chunks  []builtinPatchChunk
}

func parseBuiltinPatch(patchText string) ([]builtinPatchOperation, error) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(patchText, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(strings.TrimSpace(normalized), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "*** Begin Patch" || strings.TrimSpace(lines[len(lines)-1]) != "*** End Patch" {
		return nil, errors.New("patch must start with *** Begin Patch and end with *** End Patch")
	}
	operations := make([]builtinPatchOperation, 0)
	for index := 1; index < len(lines)-1; {
		line := lines[index]
		operation := builtinPatchOperation{}
		switch {
		case strings.HasPrefix(line, "*** Add File:"):
			operation.kind = "add"
			operation.path = strings.TrimSpace(strings.TrimPrefix(line, "*** Add File:"))
			index++
			content := make([]string, 0)
			for index < len(lines)-1 && !strings.HasPrefix(lines[index], "*** ") {
				if !strings.HasPrefix(lines[index], "+") {
					return nil, fmt.Errorf("add file %q contains a line without + prefix", operation.path)
				}
				content = append(content, strings.TrimPrefix(lines[index], "+"))
				index++
			}
			operation.content = strings.Join(content, "\n")
			if len(content) > 0 {
				operation.content += "\n"
			}
		case strings.HasPrefix(line, "*** Delete File:"):
			operation.kind = "delete"
			operation.path = strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File:"))
			index++
		case strings.HasPrefix(line, "*** Update File:"):
			operation.kind = "update"
			operation.path = strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))
			index++
			for index < len(lines)-1 && !strings.HasPrefix(lines[index], "*** ") {
				if !strings.HasPrefix(lines[index], "@@") {
					return nil, fmt.Errorf("update file %q must contain @@ chunks", operation.path)
				}
				chunk := builtinPatchChunk{context: strings.TrimSpace(strings.TrimPrefix(lines[index], "@@"))}
				index++
				for index < len(lines)-1 && !strings.HasPrefix(lines[index], "@@") && !strings.HasPrefix(lines[index], "*** ") {
					change := lines[index]
					if change == "" {
						return nil, fmt.Errorf("update file %q contains an unprefixed blank line", operation.path)
					}
					switch change[0] {
					case ' ':
						chunk.old = append(chunk.old, change[1:])
						chunk.new = append(chunk.new, change[1:])
					case '-':
						chunk.old = append(chunk.old, change[1:])
					case '+':
						chunk.new = append(chunk.new, change[1:])
					default:
						return nil, fmt.Errorf("update file %q contains an invalid change line", operation.path)
					}
					index++
				}
				if len(chunk.old) == 0 && len(chunk.new) == 0 {
					return nil, fmt.Errorf("update file %q contains an empty chunk", operation.path)
				}
				operation.chunks = append(operation.chunks, chunk)
			}
			if len(operation.chunks) == 0 {
				return nil, fmt.Errorf("update file %q has no chunks", operation.path)
			}
		default:
			return nil, fmt.Errorf("unexpected patch line %q", line)
		}
		if operation.path == "" {
			return nil, errors.New("patch file path must not be empty")
		}
		operations = append(operations, operation)
	}
	if len(operations) == 0 {
		return nil, errors.New("patch contains no file operations")
	}
	return operations, nil
}

func findPatchSequence(lines, pattern []string, start int) int {
	if len(pattern) == 0 {
		return len(lines)
	}
	for index := start; index+len(pattern) <= len(lines); index++ {
		matched := true
		for offset := range pattern {
			if lines[index+offset] != pattern[offset] {
				matched = false
				break
			}
		}
		if matched {
			return index
		}
	}
	return -1
}

func applyBuiltinPatchChunks(path string, content []byte, chunks []builtinPatchChunk) ([]byte, error) {
	text := strings.ReplaceAll(strings.ReplaceAll(string(content), "\r\n", "\n"), "\r", "\n")
	hadTrailingNewline := strings.HasSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\n")
	lines := []string{}
	if text != "" {
		lines = strings.Split(text, "\n")
	}
	cursor := 0
	for _, chunk := range chunks {
		if chunk.context != "" {
			contextIndex := findPatchSequence(lines, []string{chunk.context}, cursor)
			if contextIndex < 0 {
				return nil, fmt.Errorf("failed to find context %q in %s", chunk.context, path)
			}
			cursor = contextIndex + 1
		}
		index := findPatchSequence(lines, chunk.old, cursor)
		if index < 0 {
			return nil, fmt.Errorf("failed to find expected lines in %s:\n%s", path, strings.Join(chunk.old, "\n"))
		}
		next := make([]string, 0, len(lines)-len(chunk.old)+len(chunk.new))
		next = append(next, lines[:index]...)
		next = append(next, chunk.new...)
		next = append(next, lines[index+len(chunk.old):]...)
		lines = next
		cursor = index + len(chunk.new)
	}
	result := strings.Join(lines, "\n")
	if hadTrailingNewline || result != "" {
		result += "\n"
	}
	return []byte(result), nil
}

func callBuiltinApplyPatch(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	patchText, err := builtinString(arguments, "patchText", true)
	if err != nil {
		return builtinToolError(err)
	}
	operations, err := parseBuiltinPatch(patchText)
	if err != nil {
		return builtinToolError(fmt.Errorf("apply_patch verification failed: %w", err))
	}
	type preparedOperation struct {
		kind     string
		target   string
		relative string
		content  []byte
	}
	prepared := make([]preparedOperation, 0, len(operations))
	maxBytes := config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes
	for _, operation := range operations {
		target, relative, resolveErr := resolveWorkspacePath(ctx, operation.path, false, operation.kind == "add")
		if resolveErr != nil {
			return builtinToolError(fmt.Errorf("apply_patch verification failed for %s: %w", operation.path, resolveErr))
		}
		change := preparedOperation{kind: operation.kind, target: target, relative: relative}
		switch operation.kind {
		case "add":
			if _, statErr := os.Stat(target); statErr == nil {
				return builtinToolError(fmt.Errorf("apply_patch verification failed: file already exists: %s", relative))
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return builtinToolError(statErr)
			}
			change.content = []byte(operation.content)
		case "update":
			current, readErr := os.ReadFile(target)
			if readErr != nil {
				return builtinToolError(fmt.Errorf("apply_patch verification failed: %w", readErr))
			}
			change.content, readErr = applyBuiltinPatchChunks(relative, current, operation.chunks)
			if readErr != nil {
				return builtinToolError(fmt.Errorf("apply_patch verification failed: %w", readErr))
			}
		case "delete":
			info, statErr := os.Stat(target)
			if statErr != nil || !info.Mode().IsRegular() {
				return builtinToolError(fmt.Errorf("apply_patch verification failed: file is unavailable: %s", relative))
			}
		}
		if int64(len(change.content)) > maxBytes {
			return builtinToolError(fmt.Errorf("apply_patch result exceeds the configured size limit: %s", relative))
		}
		prepared = append(prepared, change)
	}

	applied := make([]map[string]string, 0, len(prepared))
	for _, operation := range prepared {
		switch operation.kind {
		case "add", "update":
			if err = writeBuiltinFile(operation.target, operation.content); err != nil {
				return builtinToolError(fmt.Errorf("patch partially applied before %s: %w", operation.relative, err))
			}
		case "delete":
			if err = os.Remove(operation.target); err != nil {
				return builtinToolError(fmt.Errorf("patch partially applied before %s: %w", operation.relative, err))
			}
		}
		applied = append(applied, map[string]string{"operation": operation.kind, "path": operation.relative})
	}
	return builtinToolResult(map[string]any{"applied": applied})
}
