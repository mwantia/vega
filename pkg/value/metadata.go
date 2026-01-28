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
var _ Memberable = (*Metadata)(nil)
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

func (v *Metadata) GetMember(name string) (Value, error) {
	switch name {
	case "id":
		return NewString(v.Meta.ID), nil
	case "key":
		return NewString(v.Meta.Key), nil
	case "mode":
		i := int(v.Meta.Mode)
		return NewInteger(i), nil
	case "size":
		return NewLong(v.Meta.Size), nil
	case "accesstime":
		s := v.Meta.AccessTime.Format(time.RFC3339)
		return NewString(s), nil
	case "modifytime":
		s := v.Meta.ModifyTime.Format(time.RFC3339)
		return NewString(s), nil
	case "createtime":
		s := v.Meta.CreateTime.Format(time.RFC3339)
		return NewString(s), nil
	case "uid":
		return NewLong(v.Meta.UID), nil
	case "gid":
		return NewLong(v.Meta.GID), nil
	case "contentType":
		s := string(v.Meta.ContentType)
		return NewString(s), nil
	case "etag":
		return NewString(v.Meta.ETag), nil
	}

	return nil, fmt.Errorf("unknown membername defined")
}

func (v *Metadata) SetMember(name string, val Value) (bool, error) {
	switch name {
	case "id":
		if s, ok := val.(*String); ok {
			v.Meta.ID = s.Value
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'string', got '%s'", val.Type())
	case "key":
		if s, ok := val.(*String); ok {
			v.Meta.Key = s.Value
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'string', got '%s'", val.Type())
	case "mode":
		if i, ok := val.(*Integer); ok {
			v.Meta.Mode = data.FileMode(i.Value)
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'integer', got '%s'", val.Type())
	case "size":
		if l, ok := val.(*Long); ok {
			v.Meta.Size = l.Value
			return true, nil
		}
		if i, ok := val.(*Integer); ok {
			v.Meta.Size = int64(i.Value)
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'integer' or 'long', got '%s'", val.Type())
	case "accesstime":
		return false, fmt.Errorf("not yet implemented")
	case "modifytime":
		return false, fmt.Errorf("not yet implemented")
	case "createtime":
		return false, fmt.Errorf("not yet implemented")
	case "uid":
		if l, ok := val.(*Long); ok {
			v.Meta.UID = l.Value
			return true, nil
		}
		if i, ok := val.(*Integer); ok {
			v.Meta.UID = int64(i.Value)
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'integer' or 'long', got '%s'", val.Type())
	case "gid":
		if l, ok := val.(*Long); ok {
			v.Meta.GID = l.Value
			return true, nil
		}
		if i, ok := val.(*Integer); ok {
			v.Meta.GID = int64(i.Value)
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'integer' or 'long', got '%s'", val.Type())
	case "contentType":
		if s, ok := val.(*String); ok {
			v.Meta.ContentType = data.ContentType(s.Value)
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'string', got '%s'", val.Type())
	case "etag":
		if s, ok := val.(*String); ok {
			v.Meta.ETag = s.Value
			return true, nil
		}
		return false, fmt.Errorf("member argument must be 'string', got '%s'", val.Type())
	}
	return false, fmt.Errorf("unknown membername defined")
}

func (v *Metadata) Method(name string, args []Value) (Value, error) {
	return nil, fmt.Errorf("unknown method call defined")
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
		return NewInteger(int(m.Meta.Mode)), nil
	case "size":
		return NewLong(m.Meta.Size), nil
	case "accesstime":
		return NewString(m.Meta.AccessTime.Format(time.RFC3339)), nil
	case "modifytime":
		return NewString(m.Meta.ModifyTime.Format(time.RFC3339)), nil
	case "createtime":
		return NewString(m.Meta.CreateTime.Format(time.RFC3339)), nil
	case "uid":
		return NewLong(m.Meta.UID), nil
	case "gid":
		return NewLong(m.Meta.GID), nil
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
