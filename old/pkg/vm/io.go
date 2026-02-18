package vm

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/mwantia/vega/old/pkg/value"
	"github.com/mwantia/vfs/data"
)

func newBuiltinPrintFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.String()
	}
	fmt.Fprint(vm.stdout, strings.Join(parts, " "))
	return value.Nil, nil
}

func newBuiltinPrintlnFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.String()
	}
	fmt.Fprintln(vm.stdout, strings.Join(parts, " "))
	return value.Nil, nil
}

func newBuiltinInputFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	// Print prompt if provided
	if len(args) > 0 {
		fmt.Fprint(vm.stdout, args[0].String())
	}

	reader := bufio.NewReader(vm.stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return value.Nil, nil
	}
	return value.NewString(strings.TrimSuffix(line, "\n")), nil
}

func newBuiltinEtagFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if vm.vfs == nil {
		return nil, fmt.Errorf("no VFS attached")
	}

	ctx := vm.Context()

	if len(args) < 1 {
		return nil, fmt.Errorf("etag() expects at least 1 argument, got %d", len(args))
	}

	path, ok := args[0].(*value.String)
	if !ok {
		return nil, fmt.Errorf("read expects string path, got %s", args[0].Type())
	}

	// First, try to get metadata with ETag
	stat, err := vm.vfs.StatMetadata(ctx, path.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve metadata: %v", err)
	}

	// Check if it's a directory
	if stat.Mode.IsDir() {
		return nil, fmt.Errorf("%s: is a directory", path.Value)
	}

	size := int64(8 * 1024 * 1024) // Default 8MB
	if len(args) > 2 {
		if sv, ok := args[1].(*value.String); ok {
			if size, err = sv.ParseToSize(); err != nil {
				return nil, fmt.Errorf("failed to parse size: %v", err)
			}
		}
	}

	// If ETag exists in metadata and we're not skipping, use it
	if len(args) > 3 {
		if bv, ok := args[2].(*value.Boolean); ok && bv.Value {
			return value.NewString(stat.ETag), nil
		}
	}

	// Calculate ETag from file content
	stream, err := vm.vfs.OpenFile(ctx, path.Value, data.AccessModeRead)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer stream.Close()

	var md5Hashes [][]byte
	buffer := make([]byte, size)

	for {
		n, err := io.ReadFull(stream, buffer)
		if n > 0 {
			// Calculate MD5 for this chunk
			hasher := md5.New()
			hasher.Write(buffer[:n])
			md5Hashes = append(md5Hashes, hasher.Sum(nil))
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to calculate chunk: %v", err)
		}
	}

	// Empty file
	if len(md5Hashes) < 1 {
		hash := hex.EncodeToString(md5.New().Sum(nil))
		return value.NewString(hash), nil
	}

	// Single chunk - return its MD5
	if len(md5Hashes) == 1 {
		hash := hex.EncodeToString(md5Hashes[0])
		return value.NewString(hash), nil
	}

	// Multiple chunks - concatenate digests, MD5 that, append part count
	var digests []byte
	for _, hash := range md5Hashes {
		digests = append(digests, hash...)
	}

	finalHasher := md5.New()
	finalHasher.Write(digests)

	return value.NewString(fmt.Sprintf("%s-%d\n", hex.EncodeToString(finalHasher.Sum(nil)), len(md5Hashes))), nil
}
