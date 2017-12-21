package reflection

import (
	"errors"
	"reflect"
	"time"
)

// errors ...
var (
	ErrNoFieldWithTagFound = errors.New("field with tag name not found in struct")
)

// MapAdapter defines a function type which takes a Field returning a appropriate
// representation value or an error.
type MapAdapter func(Field) (interface{}, error)

// TimeMapper returns a MapAdapter which always formats time into provided layout
// and returns the string version of the giving time.
func TimeMapper(layout string) MapAdapter {
	return func(f Field) (interface{}, error) {
		if timeObj, ok := f.Value.Interface().(time.Time); ok {
			return timeObj.Format(layout), nil
		}
		if timeObj, ok := f.Value.Interface().(*time.Time); ok {
			return timeObj.Format(layout), nil
		}
		return nil, errors.New("not time value")
	}
}

// InverseMapAdapter defines a function type which takes a Field and concrete value
// returning appropriate go value or an error. It does the inverse of a MapAdapter.
type InverseMapAdapter func(Field, interface{}) (interface{}, error)

// TimeInverseMapper returns a InverseMapAdapter for time.Time values which
// turns incoming string values of time into Time.Time object.
func TimeInverseMapper(layout string) InverseMapAdapter {
	return func(f Field, val interface{}) (interface{}, error) {
		if _, ok := val.(time.Time); ok {
			return val, nil
		}
		if dtime, ok := val.(*time.Time); ok {
			return *dtime, nil
		}
		if formatted, ok := val.(string); ok {
			return time.Parse(layout, formatted)
		}
		return nil, errors.New("non supported time type")
	}
}

// Mapper defines an interface which exposes methods to
// map a struct from giving tags to a map and vise-versa.
type Mapper interface {
	MapTo(string, interface{}, map[string]interface{}) error
	MapFrom(string, interface{}) (map[string]interface{}, error)
}

// StructMapper implements a struct mapping utility which allows mapping struct fields
// to a map and vise-versa.
// It uses custom adapters which if available for a giving type will handle the necessary
// conversion else use the default value's of those fields in the map. This means, no nil
// struct pointer instance should be passed for either conversion or mapping back.
// WARNING: StructMapper is not goroutine safe.
type StructMapper struct {
	adapters  map[reflect.Type]MapAdapter
	iadapters map[reflect.Type]InverseMapAdapter
}

// NewStructMapper returns a new instance of StructMapper.
func NewStructMapper() *StructMapper {
	return &StructMapper{
		adapters:  make(map[reflect.Type]MapAdapter),
		iadapters: make(map[reflect.Type]InverseMapAdapter),
	}
}

// MapTo takes giving struct(target) and map of values which it attempts to map
// back into struct field types using tag. It returns error if operation fails.
// Ensure provided type is a pointer of giving struct type and is non-nil.
func (sm *StructMapper) MapTo(tag string, target interface{}, data map[string]interface{}) error {
	fields, err := GetTagFields(target, tag, true)
	if err != nil {
		return err
	}

	// If no fields get pulled, just stop here.
	if len(fields) == 0 {
		return ErrNoFieldWithTagFound
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	for _, field := range fields {
		// We do a 3 step checks, first with tag name, if non, then name as is, if not
		// then use lowercase of name.
		value, ok := data[field.Tag]
		if !ok {
			value, ok = data[field.Name]
			if !ok {
				value, ok = data[field.NameLC]
				if !ok {
					continue
				}
			}
		}

		fieldTarget := targetValue.Field(field.Index)

		if !fieldTarget.CanSet() {
			continue
		}

		if iadapter, ok := sm.iadapters[field.Type]; ok {
			converted, err := iadapter(field, value)
			if err != nil {
				return err
			}

			fieldTarget.Set(reflect.ValueOf(converted))
			continue
		}

		// If it's a map and the type is a struct, attempt to
		// map that struct fields with map.
		if innerMap, ok := value.(map[string]interface{}); ok {
			if field.Type.Kind() == reflect.Struct {
				if err := sm.MapTo(tag, fieldTarget.Addr().Interface(), innerMap); err != nil {
					return err
				}
				continue
			}
		}

		fieldTarget.Set(reflect.ValueOf(value))
	}

	return nil
}

// MapFrom returns a map which contains all values of provided struct returned as a map
// using giving tag name.
// Ensure provided type is non-nil.
func (sm *StructMapper) MapFrom(tag string, target interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	fields, err := GetTagFields(target, tag, true)
	if err != nil {
		return data, err
	}

	// If it has no fields, or non was extractable, then return empty map.
	if len(fields) == 0 {
		return data, nil
	}

	for _, field := range fields {
		if !field.Value.CanInterface() {
			continue
		}

		if adapter, ok := sm.adapters[field.Type]; ok {
			res, err := adapter(field)
			if err != nil {
				return data, err
			}

			if field.Tag == "" {
				data[field.Name] = res
			} else {
				data[field.Tag] = res
			}
			continue
		}

		if field.Type.Kind() == reflect.Struct {
			mapped, err := sm.MapFrom(tag, field.Value.Interface())
			if err != nil {
				return data, err
			}

			if field.Tag == "" {
				data[field.Name] = mapped
			} else {
				data[field.Tag] = mapped
			}
			continue
		}

		if field.Tag == "" {
			data[field.Name] = field.Value.Interface()
		} else {
			data[field.Tag] = field.Value.Interface()
		}
	}

	return data, nil
}

// HasInverseAdapter returns true/false if giving type has inverse adapter registered.
func (sm *StructMapper) HasInverseAdapter(ty reflect.Type) bool {
	if sm.iadapters == nil {
		sm.iadapters = make(map[reflect.Type]InverseMapAdapter)
		return false
	}
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	_, exists := sm.adapters[ty]
	return exists
}

// AddInverseAdapter adds giving inverse adapter to be responsible for generating go type
// for giving reflect type.
// It replaces any previous inverse adapter with new inverse adapter for type.
// WARNING: Ensure to use StructMapper.HasAdapter to validate if adapter
// exists for type.
func (sm *StructMapper) AddInverseAdapter(ty reflect.Type, adapter InverseMapAdapter) {
	if sm.iadapters == nil {
		sm.iadapters = make(map[reflect.Type]InverseMapAdapter)
	}
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	sm.iadapters[ty] = adapter
}

// HasAdapter returns true/false if giving type has adapter registered.
func (sm *StructMapper) HasAdapter(ty reflect.Type) bool {
	if sm.adapters == nil {
		sm.adapters = make(map[reflect.Type]MapAdapter)
		return false
	}
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	_, exists := sm.adapters[ty]
	return exists
}

// AddAdapter adds giving adapter to be responsible for giving type.
// It replaces any previous adapter with new adapter for type.
// WARNING: Ensure to use StructMapper.HasAdapter to validate if adapter
// exists for type.
func (sm *StructMapper) AddAdapter(ty reflect.Type, adapter MapAdapter) {
	if sm.adapters == nil {
		sm.adapters = make(map[reflect.Type]MapAdapter)
	}
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	sm.adapters[ty] = adapter
}
