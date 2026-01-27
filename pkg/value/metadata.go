package value

import (
	"fmt"
	"strings"
	"time"

	"github.com/mwantia/vfs/data"
)

// Metadata wraps VFS metadata as a first-class Vega value.
type Metadata struct {
	Meta data.Metadata
}

var _ Value = (*Metadata)(nil)
var _ Indexable = (*Metadata)(nil)
var _ Methodable = (*Metadata)(nil)

// NewMetadata creates a new Metadata from VFS metadata.
func NewMetadata(meta data.Metadata) *Metadata {
	return &Metadata{Meta: meta}
}

func (m *Metadata) Type() string {
	return TypeMetadata
}

func (m *Metadata) String() string {
	return fmt.Sprintf("metadata{key: %q, type: %s, size: %d}",
		m.Meta.Key, m.Meta.GetType(), m.Meta.Size)
}

func (m *Metadata) Boolean() bool {
	return m.Meta.Key != ""
}

func (m *Metadata) Equal(other Value) bool {
	o, ok := other.(*Metadata)
	if !ok {
		return false
	}

	return m.Meta.ID == o.Meta.ID && m.Meta.Key == o.Meta.Key
}

func (v *Metadata) Method(name string, args []Value) (Value, error) {
	return nil, fmt.Errorf("unknown method call")
}

// Index allows accessing metadata fields via meta["field"].
func (m *Metadata) Index(key Value) (Value, error) {
	k, ok := key.(*String)
	if !ok {
		return nil, fmt.Errorf("metadata key must be string, got %s", key.Type())
	}
	return m.GetField(k.Value)
}

// SetIndex is not supported for metadata (read-only).
func (m *Metadata) SetIndex(key Value, val Value) error {
	return fmt.Errorf("metadata values are read-only")
}

// GetField returns a metadata field by name.
// Supports both camelCase and snake_case field names.
func (m *Metadata) GetField(name string) (Value, error) {
	switch strings.ToLower(name) {
	case "id":
		return NewString(m.Meta.ID), nil
	case "key":
		return NewString(m.Meta.Key), nil
	case "mode":
		return NewInteger(int64(m.Meta.Mode)), nil
	case "size":
		return NewInteger(m.Meta.Size), nil
	case "accesstime":
		return NewString(m.Meta.AccessTime.Format(time.RFC3339)), nil
	case "modifytime":
		return NewString(m.Meta.ModifyTime.Format(time.RFC3339)), nil
	case "createtime":
		return NewString(m.Meta.CreateTime.Format(time.RFC3339)), nil
	case "uid":
		return NewInteger(m.Meta.UID), nil
	case "gid":
		return NewInteger(m.Meta.GID), nil
	case "contentType":
		return NewString(string(m.Meta.ContentType)), nil
	case "etag":
		return NewString(m.Meta.ETag), nil
	case "filetype":
		return NewString(string(m.Meta.GetType())), nil
	case "isdir":
		return NewBoolean(m.Meta.Mode.IsDir()), nil
	case "isfile":
		return NewBoolean(m.Meta.Mode.IsRegular()), nil
	case "ismount":
		return NewBoolean(m.Meta.Mode.IsMount()), nil
	case "issymlink":
		return NewBoolean(m.Meta.Mode.IsSymlink()), nil
	case "attributes":
		return m.attributesToMap(), nil
	default:
		// Check if it's an extended attribute
		if m.Meta.Attributes != nil {
			if v, ok := m.Meta.Attributes[name]; ok {
				return NewString(v), nil
			}
		}
		return Nil, nil
	}
}

// attributesToMap converts the extended attributes to a Map.
func (m *Metadata) attributesToMap() *Map {
	mv := NewMap()
	if m.Meta.Attributes != nil {
		for k, v := range m.Meta.Attributes {
			mv.Set(k, NewString(v))
		}
	}
	return mv
}

// Fields returns all available field names.
func (m *Metadata) Fields() []string {
	return []string{
		"id",
		"key",
		"mode",
		"size",
		"accessTime",
		"modifyTime",
		"createTime",
		"uid",
		"gid",
		"contentType",
		"etag",
		"filetype",
		"isdir",
		"isfile",
		"ismount",
		"issymlink",
		"attributes",
	}
}
