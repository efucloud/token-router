package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
)

const builtinChatServerName = "builtin"

type chatBuiltinTool struct {
	capability dtos.ChatMCPToolCapability
	call       func(context.Context, map[string]any) dtos.ChatMCPToolResult
}

type builtinCommandInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// The image PATH is immutable for the lifetime of the service in normal
// deployments. Discover it once while the package is initialized so every
// model request sees the same bounded tool catalog.
var builtinCommands = discoverBuiltinCommands()

func discoverBuiltinCommands() []builtinCommandInfo {
	seen := map[string]string{}
	for _, directory := range filepath.SplitList(os.Getenv("PATH")) {
		directory = strings.TrimSpace(directory)
		if directory == "" {
			continue
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || len(seen) >= 4096 {
				continue
			}
			name := entry.Name()
			if _, exists := seen[name]; exists {
				continue
			}
			path := filepath.Join(directory, name)
			info, err := os.Stat(path)
			if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
				seen[name] = path
			}
		}
	}
	result := make([]builtinCommandInfo, 0, len(seen))
	for name, path := range seen {
		result = append(result, builtinCommandInfo{Name: name, Path: path})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func builtinCommandSummary() string {
	available := make(map[string]struct{}, len(builtinCommands))
	for _, command := range builtinCommands {
		available[command.Name] = struct{}{}
	}
	priority := []string{"kubectl", "helm", "kustomize", "docker", "podman", "git", "rg", "jq", "yq", "curl", "wget", "python3", "node", "npm", "go"}
	found := make([]string, 0, len(priority))
	for _, name := range priority {
		if _, ok := available[name]; ok {
			found = append(found, name)
		}
	}
	if len(found) == 0 {
		return fmt.Sprintf("%d executable commands detected; call discover_commands before choosing one", len(builtinCommands))
	}
	return fmt.Sprintf("notable commands detected: %s; call discover_commands to inspect the full catalog", strings.Join(found, ", "))
}

func builtinObjectSchema(required []string, properties map[string]any) map[string]any {
	result := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func builtinChatTools(_ context.Context) []chatBuiltinTool {
	tools := []chatBuiltinTool{
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "read_file", Description: "Read a UTF-8 text file from the current signed-in user's personal workspace. Paths are relative to that workspace and cannot escape it.",
				InputSchema: builtinObjectSchema([]string{"path"}, map[string]any{
					"path":   map[string]any{"type": "string", "description": "Workspace-relative file path"},
					"offset": map[string]any{"type": "integer", "minimum": 1, "description": "First line to return (1-based, default 1)"},
					"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 2000, "description": "Maximum lines to return (default 200)"},
				}),
			}, call: callBuiltinReadFile,
		},
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "list_files", Description: "List files and directories in the current signed-in user's personal workspace. Hidden entries are included; results are bounded.",
				InputSchema: builtinObjectSchema(nil, map[string]any{
					"path":  map[string]any{"type": "string", "description": "Workspace-relative directory path, default workspace root"},
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 500, "description": "Maximum entries (default 200)"},
				}),
			}, call: callBuiltinListFiles,
		},
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "search_files", Description: "Search the current signed-in user's UTF-8 personal workspace files for literal text and return bounded line matches. Use this before editing when the target location is unknown.",
				InputSchema: builtinObjectSchema([]string{"query"}, map[string]any{
					"query": map[string]any{"type": "string", "description": "Literal text to find"},
					"path":  map[string]any{"type": "string", "description": "Workspace-relative file or directory, default workspace root"},
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 500, "description": "Maximum matches (default 100)"},
				}),
			}, call: callBuiltinSearchFiles,
		},
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "write_file", Description: "Create or replace one UTF-8 file inside the current signed-in user's personal workspace. Parent directories are created when needed.",
				InputSchema: builtinObjectSchema([]string{"path", "content"}, map[string]any{
					"path":    map[string]any{"type": "string", "description": "Workspace-relative file path"},
					"content": map[string]any{"type": "string", "description": "Complete new file content"},
				}),
			}, call: callBuiltinWriteFile,
		},
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "edit_file", Description: "Replace exact text in one UTF-8 workspace file. The old text must occur exactly once unless replaceAll is true.",
				InputSchema: builtinObjectSchema([]string{"path", "oldText", "newText"}, map[string]any{
					"path":       map[string]any{"type": "string", "description": "Workspace-relative file path"},
					"oldText":    map[string]any{"type": "string", "minLength": 1, "description": "Exact text to replace"},
					"newText":    map[string]any{"type": "string", "description": "Replacement text"},
					"replaceAll": map[string]any{"type": "boolean", "description": "Replace every exact occurrence (default false)"},
				}),
			}, call: callBuiltinEditFile,
		},
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "apply_patch", Description: "Apply one workspace patch with add, update, and delete file operations. Use the *** Begin Patch / *** Add File, Update File, or Delete File / *** End Patch format; update chunks use @@ and space, -, or + line prefixes.",
				InputSchema: builtinObjectSchema([]string{"patchText"}, map[string]any{
					"patchText": map[string]any{"type": "string", "minLength": 1, "description": "Complete patch text in the documented *** Begin Patch format"},
				}),
			}, call: callBuiltinApplyPatch,
		},
		{
			capability: dtos.ChatMCPToolCapability{
				Name: "discover_commands", Description: "Discover executable programs installed on the Token Router service PATH. Use it to check whether kubectl, helm, git, language runtimes, or other image tools are available before calling command.",
				InputSchema: builtinObjectSchema(nil, map[string]any{
					"query": map[string]any{"type": "string", "description": "Optional case-insensitive name filter"},
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 500, "description": "Maximum results (default 100)"},
				}),
			}, call: callBuiltinDiscoverCommands,
		},
	}
	if config.ApplicationConfig.Chat.BuiltinTools.CommandEnabled {
		tools = append(tools, chatBuiltinTool{
			capability: dtos.ChatMCPToolCapability{
				Name:        "command",
				Description: "Execute a shell command inside the Token Router service container with the service process identity and network access; " + builtinCommandSummary() + ". Use workspace-relative workdir values. Commands may change external systems (for example kubectl), so verify the target and use the least destructive command that satisfies the request.",
				InputSchema: builtinObjectSchema([]string{"command"}, map[string]any{
					"command":        map[string]any{"type": "string", "minLength": 1, "description": "Shell command string to execute with /bin/sh -lc"},
					"workdir":        map[string]any{"type": "string", "description": "Workspace-relative working directory, default workspace root"},
					"timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "description": "Timeout, capped by the service configuration"},
				}),
			}, call: callBuiltinCommand,
		})
	}
	return tools
}

func builtinToolResult(value any) dtos.ChatMCPToolResult {
	data, err := json.Marshal(value)
	if err != nil {
		return builtinToolError(err)
	}
	return dtos.ChatMCPToolResult{Content: string(data)}
}

func builtinToolError(err error) dtos.ChatMCPToolResult {
	data, _ := json.Marshal(map[string]string{"error": err.Error()})
	return dtos.ChatMCPToolResult{Content: string(data), IsError: true}
}

func builtinString(arguments map[string]any, name string, required bool) (string, error) {
	value, exists := arguments[name]
	if !exists || value == nil {
		if required {
			return "", fmt.Errorf("%s is required", name)
		}
		return "", nil
	}
	result, ok := value.(string)
	if !ok || required && strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return result, nil
}

func builtinInt(arguments map[string]any, name string, fallback, maximum int) (int, error) {
	value, exists := arguments[name]
	if !exists || value == nil {
		return fallback, nil
	}
	var result int
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
		result = int(typed)
	case int:
		result = typed
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
		result = parsed
	default:
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if result <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	if maximum > 0 && result > maximum {
		result = maximum
	}
	return result, nil
}

func builtinBool(arguments map[string]any, name string) (bool, error) {
	value, exists := arguments[name]
	if !exists || value == nil {
		return false, nil
	}
	result, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	return result, nil
}

func callBuiltinReadFile(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	path, err := builtinString(arguments, "path", true)
	if err != nil {
		return builtinToolError(err)
	}
	offset, err := builtinInt(arguments, "offset", 1, 0)
	if err != nil {
		return builtinToolError(err)
	}
	limit, err := builtinInt(arguments, "limit", 200, 2000)
	if err != nil {
		return builtinToolError(err)
	}
	target, relative, err := resolveWorkspacePath(ctx, path, false, false)
	if err != nil {
		return builtinToolError(err)
	}
	file, err := os.Open(target)
	if err != nil {
		return builtinToolError(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	maxBytes := config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes
	scanner.Buffer(make([]byte, 64*1024), int(maxBytes))
	lines := make([]string, 0, limit)
	lineNumber := 0
	capturedBytes := int64(0)
	truncated := false
	for scanner.Scan() {
		lineNumber++
		if lineNumber < offset {
			continue
		}
		if len(lines) >= limit {
			truncated = true
			break
		}
		text := scanner.Text()
		if capturedBytes+int64(len(text)) > maxBytes {
			truncated = true
			break
		}
		lines = append(lines, text)
		capturedBytes += int64(len(text))
	}
	if err = scanner.Err(); err != nil {
		return builtinToolError(err)
	}
	return builtinToolResult(map[string]any{"path": relative, "offset": offset, "lines": lines, "truncated": truncated})
}

func callBuiltinListFiles(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	path, err := builtinString(arguments, "path", false)
	if err != nil {
		return builtinToolError(err)
	}
	limit, err := builtinInt(arguments, "limit", 200, 500)
	if err != nil {
		return builtinToolError(err)
	}
	target, relative, err := resolveWorkspacePath(ctx, path, true, false)
	if err != nil {
		return builtinToolError(err)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return builtinToolError(err)
	}
	type item struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Size int64  `json:"size,omitempty"`
	}
	result := make([]item, 0, min(limit, len(entries)))
	for _, entry := range entries {
		if len(result) >= limit {
			break
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		kind := "file"
		if entry.IsDir() {
			kind = "directory"
		}
		result = append(result, item{Name: entry.Name(), Type: kind, Size: info.Size()})
	}
	return builtinToolResult(map[string]any{"path": relative, "entries": result, "truncated": len(entries) > len(result)})
}

func callBuiltinSearchFiles(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	query, err := builtinString(arguments, "query", true)
	if err != nil {
		return builtinToolError(err)
	}
	path, err := builtinString(arguments, "path", false)
	if err != nil {
		return builtinToolError(err)
	}
	limit, err := builtinInt(arguments, "limit", 100, 500)
	if err != nil {
		return builtinToolError(err)
	}
	target, _, err := resolveWorkspacePath(ctx, path, true, false)
	if err != nil {
		return builtinToolError(err)
	}
	root, err := personalWorkspaceRoot(ctx)
	if err != nil {
		return builtinToolError(err)
	}
	type match struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Text string `json:"text"`
	}
	matches := make([]match, 0, limit)
	maxBytes := config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes
	capturedBytes := int64(0)
	truncated := false
	err = filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || len(matches) >= limit {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() {
			if path != target && (entry.Name() == ".git" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || info.Size() > maxBytes {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), int(maxBytes))
		line := 0
		for scanner.Scan() && len(matches) < limit {
			line++
			text := scanner.Text()
			if strings.Contains(text, query) {
				if capturedBytes+int64(len(text)) > maxBytes {
					truncated = true
					break
				}
				relative, _ := filepath.Rel(root, path)
				matches = append(matches, match{Path: filepath.ToSlash(relative), Line: line, Text: text})
				capturedBytes += int64(len(text))
			}
		}
		return nil
	})
	if err != nil {
		return builtinToolError(err)
	}
	return builtinToolResult(map[string]any{"query": query, "matches": matches, "truncated": truncated || len(matches) == limit})
}

func writeBuiltinFile(target string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".token-router-write-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err = temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Chmod(0o640); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, target)
}

func callBuiltinWriteFile(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	path, err := builtinString(arguments, "path", true)
	if err != nil {
		return builtinToolError(err)
	}
	content, err := builtinString(arguments, "content", false)
	if err != nil {
		return builtinToolError(err)
	}
	if int64(len(content)) > config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes {
		return builtinToolError(errors.New("content exceeds the configured size limit"))
	}
	target, relative, err := resolveWorkspacePath(ctx, path, false, true)
	if err != nil {
		return builtinToolError(err)
	}
	_, statErr := os.Stat(target)
	created := errors.Is(statErr, os.ErrNotExist)
	if err = writeBuiltinFile(target, []byte(content)); err != nil {
		return builtinToolError(err)
	}
	return builtinToolResult(map[string]any{"path": relative, "bytes": len(content), "created": created})
}

func callBuiltinEditFile(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	path, err := builtinString(arguments, "path", true)
	if err != nil {
		return builtinToolError(err)
	}
	oldText, err := builtinString(arguments, "oldText", true)
	if err != nil {
		return builtinToolError(err)
	}
	newText, err := builtinString(arguments, "newText", false)
	if err != nil {
		return builtinToolError(err)
	}
	replaceAll, err := builtinBool(arguments, "replaceAll")
	if err != nil {
		return builtinToolError(err)
	}
	target, relative, err := resolveWorkspacePath(ctx, path, false, false)
	if err != nil {
		return builtinToolError(err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return builtinToolError(err)
	}
	if int64(len(content)) > config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes {
		return builtinToolError(errors.New("file exceeds the configured size limit"))
	}
	count := bytes.Count(content, []byte(oldText))
	if count == 0 {
		return builtinToolError(errors.New("oldText was not found"))
	}
	if count > 1 && !replaceAll {
		return builtinToolError(fmt.Errorf("oldText occurs %d times; provide more context or set replaceAll", count))
	}
	replacements := 1
	if replaceAll {
		replacements = count
	}
	updated := bytes.Replace(content, []byte(oldText), []byte(newText), replacements)
	if int64(len(updated)) > config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes {
		return builtinToolError(errors.New("updated file exceeds the configured size limit"))
	}
	if err = writeBuiltinFile(target, updated); err != nil {
		return builtinToolError(err)
	}
	return builtinToolResult(map[string]any{"path": relative, "replacements": replacements})
}

func callBuiltinDiscoverCommands(_ context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	query, err := builtinString(arguments, "query", false)
	if err != nil {
		return builtinToolError(err)
	}
	limit, err := builtinInt(arguments, "limit", 100, 500)
	if err != nil {
		return builtinToolError(err)
	}
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]builtinCommandInfo, 0, min(limit, len(builtinCommands)))
	for _, command := range builtinCommands {
		if query != "" && !strings.Contains(strings.ToLower(command.Name), query) {
			continue
		}
		if len(result) >= limit {
			break
		}
		result = append(result, command)
	}
	return builtinToolResult(map[string]any{"commands": result, "returned": len(result), "detected": len(builtinCommands)})
}

type builtinLimitedBuffer struct {
	buffer    bytes.Buffer
	limit     int64
	truncated bool
}

func (b *builtinLimitedBuffer) Write(value []byte) (int, error) {
	written := len(value)
	remaining := b.limit - int64(b.buffer.Len())
	if remaining <= 0 {
		b.truncated = true
		return written, nil
	}
	if int64(len(value)) > remaining {
		value = value[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(value)
	return written, nil
}

func callBuiltinCommand(ctx context.Context, arguments map[string]any) dtos.ChatMCPToolResult {
	if !config.ApplicationConfig.Chat.BuiltinTools.CommandEnabled {
		return builtinToolError(errors.New("command execution is disabled"))
	}
	commandText, err := builtinString(arguments, "command", true)
	if err != nil {
		return builtinToolError(err)
	}
	workdir, err := builtinString(arguments, "workdir", false)
	if err != nil {
		return builtinToolError(err)
	}
	maximumTimeout := config.ApplicationConfig.Chat.BuiltinTools.CommandTimeoutSeconds
	timeout, err := builtinInt(arguments, "timeoutSeconds", maximumTimeout, maximumTimeout)
	if err != nil {
		return builtinToolError(err)
	}
	directory, relative, err := resolveWorkspacePath(ctx, workdir, true, false)
	if err != nil {
		return builtinToolError(err)
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return builtinToolError(errors.New("workdir is not a directory"))
	}
	commandContext, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	command := exec.CommandContext(commandContext, "/bin/sh", "-lc", commandText)
	command.Dir = directory
	command.Env = os.Environ()
	output := &builtinLimitedBuffer{limit: config.ApplicationConfig.Chat.BuiltinTools.MaxOutputBytes}
	command.Stdout = output
	command.Stderr = output
	err = command.Run()
	exitCode := 0
	if err != nil {
		if commandContext.Err() == context.DeadlineExceeded {
			return builtinToolResult(map[string]any{"command": commandText, "workdir": relative, "output": output.buffer.String(), "timedOut": true, "truncated": output.truncated})
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return builtinToolError(err)
		}
	}
	return builtinToolResult(map[string]any{"command": commandText, "workdir": relative, "output": output.buffer.String(), "exitCode": exitCode, "truncated": output.truncated})
}

func builtinChatCapability(ctx context.Context) dtos.ChatMCPServerCapability {
	result := dtos.ChatMCPServerCapability{
		Name: builtinChatServerName, Kind: "builtin", Status: "connected",
		DefaultEnabled: config.ApplicationConfig.Chat.BuiltinTools.DefaultEnabled,
		Tools:          make([]dtos.ChatMCPToolCapability, 0),
	}
	for _, tool := range builtinChatTools(ctx) {
		result.Tools = append(result.Tools, tool.capability)
	}
	return result
}

func callChatBuiltinTool(ctx context.Context, name string, arguments map[string]any) (dtos.ChatMCPToolResult, error) {
	if !config.ApplicationConfig.Chat.BuiltinTools.Enabled {
		return dtos.ChatMCPToolResult{}, errors.New("builtin tools are disabled")
	}
	for _, tool := range builtinChatTools(ctx) {
		if tool.capability.Name == name {
			return tool.call(ctx, arguments), nil
		}
	}
	return dtos.ChatMCPToolResult{}, errors.New("builtin tool is not published")
}

var _ io.Writer = (*builtinLimitedBuffer)(nil)
