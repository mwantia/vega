package vm

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/mwantia/vega/pkg/value"
	"github.com/mwantia/vfs/data"
)

// read(path) or read(path, offset, size) - reads file content
func newBuiltinReadFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 1 {
		return nil, fmt.Errorf("read expects at least 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("read expects string path, got %s", args[0].Type())
	}

	// Optional offset and size
	var offset, size int64 = 0, -1
	if len(args) >= 2 {
		if v, ok := args[1].(*value.Integer); ok {
			offset = int64(v.Value)
		}
	}
	if len(args) >= 3 {
		if v, ok := args[2].(*value.Integer); ok {
			size = int64(v.Value)
		}
	}

	content, err := vm.vfs.ReadFile(vm.Context(), path.Value, offset, size)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return value.NewString(string(content)), nil
}

// write(path, data) or write(path, data, offset) - writes data to file
func newBuiltinWriteFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("write expects at least 2 arguments (path, data), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("write expects string path, got %s", args[0].Type())
	}

	content := args[1].String()

	// Optional offset
	var offset int64 = 0
	if len(args) >= 3 {
		if v, ok := args[2].(*value.Integer); ok {
			offset = int64(v.Value)
		}
	}

	n, err := vm.vfs.WriteFile(vm.Context(), path.Value, offset, []byte(content))
	if err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	return value.NewLong(int64(n)), nil
}

// stat(path) - returns file metadata
func newBuiltinStatFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("stat expects 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("stat expects string path, got %s", args[0].Type())
	}

	meta, err := vm.vfs.StatMetadata(vm.Context(), path.Value)
	if err != nil {
		return nil, fmt.Errorf("stat failed: %w", err)
	}

	return value.NewMetadata(meta), nil
}

// lookup(path) - checks if path exists, returns bool
func newBuiltinLookupFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("lookup expects 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("lookup expects string path, got %s", args[0].Type())
	}

	exists, err := vm.vfs.LookupMetadata(vm.Context(), path.Value)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}

	return value.NewBoolean(exists), nil
}

// readdir(path) - returns array of metadata values for directory entries
func newBuiltinReaddirFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("readdir expects 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("readdir expects string path, got %s", args[0].Type())
	}

	entries, err := vm.vfs.ReadDirectory(vm.Context(), path.Value)
	if err != nil {
		return nil, fmt.Errorf("readdir failed: %w", err)
	}

	elements := make([]value.Value, len(entries))
	for i, entry := range entries {
		elements[i] = value.NewMetadata(entry)
	}

	return value.NewArray(elements), nil
}

// createdir(path) - creates a directory
func newBuiltinCreatedirFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("createdir expects 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("createdir expects string path, got %s", args[0].Type())
	}

	if err := vm.vfs.CreateDirectory(vm.Context(), path.Value); err != nil {
		return nil, fmt.Errorf("createdir failed: %w", err)
	}

	return value.Nil, nil
}

// remdir(path) or remdir(path, force) - removes a directory
func newBuiltinRemdirFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 1 {
		return nil, fmt.Errorf("remdir expects at least 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("remdir expects string path, got %s", args[0].Type())
	}

	force := false
	if len(args) >= 2 {
		if v, ok := args[1].(*value.Boolean); ok {
			force = v.Value
		}
	}

	if err := vm.vfs.RemoveDirectory(vm.Context(), path.Value, force); err != nil {
		return nil, fmt.Errorf("remdir failed: %w", err)
	}

	return value.Nil, nil
}

// unlink(path) - removes a file
func newBuiltinUnlinkFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("unlink expects 1 argument (path), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("unlink expects string path, got %s", args[0].Type())
	}

	if err := vm.vfs.UnlinkFile(vm.Context(), path.Value); err != nil {
		return nil, fmt.Errorf("unlink failed: %w", err)
	}

	return value.Nil, nil
}

// rename(oldpath, newpath) - renames/moves a file or directory
func newBuiltinRenameFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) != 2 {
		return nil, fmt.Errorf("rename expects 2 arguments (oldpath, newpath), got %d", len(args))
	}

	oldPath, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("rename expects string oldpath, got %s", args[0].Type())
	}

	newPath, ok := args[1].(*value.String)
	if !ok {
		return nil, fmt.Errorf("rename expects string newpath, got %s", args[1].Type())
	}

	if err := vm.vfs.Rename(vm.Context(), oldPath.Value, newPath.Value); err != nil {
		return nil, fmt.Errorf("rename failed: %w", err)
	}

	return value.Nil, nil
}

// parseAccessMode converts a mode string to VFS AccessMode flags.
// Supported modes: "r" (read), "w" (write+create+trunc), "a" (append+create),
// "rw"/"r+" (read+write), "wx" (write+create+excl)
func parseAccessMode(mode string) (data.AccessMode, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "r":
		return data.AccessModeRead, nil
	case "w":
		return data.AccessModeWrite | data.AccessModeCreate | data.AccessModeTrunc, nil
	case "a":
		return data.AccessModeWrite | data.AccessModeCreate | data.AccessModeAppend, nil
	case "rw", "r+", "wr":
		return data.AccessModeRead | data.AccessModeWrite | data.AccessModeCreate, nil
	case "wx":
		return data.AccessModeWrite | data.AccessModeCreate | data.AccessModeExcl, nil
	default:
		return 0, fmt.Errorf("unknown mode '%s': use r, w, a, rw, or wx", mode)
	}
}

// open(path, mode) - opens a file and returns a stream
// mode: "r" (read), "w" (write), "a" (append), "rw" (read+write), "wx" (exclusive write)
func newBuiltinOpenFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("open expects 1-2 arguments (path [, mode]), got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("open expects string path, got %s", args[0].Type())
	}

	// Default mode is read
	mode := "r"
	if len(args) >= 2 {
		modeVal, ok := args[1].(*value.String)
		if !ok {
			return nil, fmt.Errorf("open expects string mode, got %s", args[1].Type())
		}
		mode = modeVal.Value
	}

	flags, err := parseAccessMode(mode)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	streamer, err := vm.vfs.OpenFile(vm.Context(), path.Value, flags)
	if err != nil {
		return nil, fmt.Errorf("open failed: %w", err)
	}

	// Wrap VFS streamer in a StreamValue
	// The streamer implements io.Reader, io.Writer, and io.Closer
	var reader io.Reader
	var writer io.Writer
	if streamer.CanRead() {
		reader = streamer
	}
	if streamer.CanWrite() {
		writer = streamer
	}

	return value.NewStream(path.Value, reader, writer, streamer), nil
}

// exec(command...) - executes a VFS command and returns exit code
// Output goes to VM's stdout
func newBuiltinExecFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 1 {
		return nil, fmt.Errorf("exec expects at least 1 argument (command), got %d", len(args))
	}

	// Convert args to strings
	cmdArgs := make([]string, len(args))
	for i, arg := range args {
		cmdArgs[i] = arg.String()
	}

	exitCode, err := vm.vfs.Execute(vm.Context(), vm.stdout, cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("exec failed: %w", err)
	}

	return value.NewInteger(exitCode), nil
}

// sexec(command...) - executes a VFS command with VM's stdin/stdout/stderr
// Returns exit code
func newBuiltinSexecFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 1 {
		return nil, fmt.Errorf("sexec expects at least 1 argument (command), got %d", len(args))
	}

	// Convert args to strings
	cmdArgs := make([]string, len(args))
	for i, arg := range args {
		cmdArgs[i] = arg.String()
	}

	exitCode, err := vm.vfs.ExecuteWithStreams(vm.Context(), vm.stdin, vm.stdout, vm.stderr, cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("sexec failed: %w", err)
	}

	return value.NewInteger(exitCode), nil
}

// capture(command...) - executes a VFS command and returns output as string
func newBuiltinCaptureFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}
	if len(args) < 1 {
		return nil, fmt.Errorf("capture expects at least 1 argument (command), got %d", len(args))
	}

	// Convert args to strings
	cmdArgs := make([]string, len(args))
	for i, arg := range args {
		cmdArgs[i] = arg.String()
	}

	// Capture output to buffer
	var buf bytes.Buffer
	_, err := vm.vfs.Execute(vm.Context(), &buf, cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("capture failed: %w", err)
	}

	return value.NewString(buf.String()), nil
}
