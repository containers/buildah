package config

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/containers/image/v5/manifest"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

// fillAllFields recursively fills in 1 or "1" for every field in the passed-in
// structure, and that slices and maps have at least one value in them.
func fillAllFields[pStruct any](t *testing.T, st pStruct) {
	v := reflect.ValueOf(st)
	if v.Kind() == reflect.Pointer {
		v = reflect.Indirect(v)
	}
	fillAllValueFields(t, v)
}

func fillAllValueFields(t *testing.T, v reflect.Value) {
	fields := reflect.VisibleFields(v.Type())
	for _, field := range fields {
		if field.Anonymous {
			// all right, fine, keep your secrets
			continue
		}
		f := v.FieldByName(field.Name)
		var keyType, elemType reflect.Type
		if field.Type.Kind() == reflect.Map {
			keyType = field.Type.Key()
		}
		switch field.Type.Kind() {
		case reflect.Array, reflect.Chan, reflect.Map, reflect.Pointer, reflect.Slice:
			elemType = field.Type.Elem()
		}
		fillValue(t, f, field.Name, field.Type.Kind(), keyType, elemType)
	}
}

func fillValue(t *testing.T, value reflect.Value, name string, kind reflect.Kind, keyType, elemType reflect.Type) {
	switch kind {
	case reflect.Invalid,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		require.NotEqualf(t, kind, kind, "unhandled %s field %s: tests require updating", kind, name)
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value.SetUint(1)
	case reflect.Map:
		if value.IsNil() {
			value.Set(reflect.MakeMap(value.Type()))
		}
		keyPtr := reflect.New(keyType)
		key := reflect.Indirect(keyPtr)
		fillValue(t, key, name, keyType.Kind(), nil, nil)
		elemPtr := reflect.New(elemType)
		elem := reflect.Indirect(elemPtr)
		fillValue(t, elem, name, elemType.Kind(), nil, nil)
		value.SetMapIndex(key, reflect.Indirect(elem))
	case reflect.Slice:
		vPtr := reflect.New(elemType)
		v := reflect.Indirect(vPtr)
		fillValue(t, v, name, elemType.Kind(), nil, nil)
		value.Set(reflect.Append(reflect.MakeSlice(value.Type(), 0, 1), v))
	case reflect.String:
		value.SetString("1")
	case reflect.Struct:
		fillAllValueFields(t, value)
	case reflect.Pointer:
		p := reflect.New(elemType)
		fillValue(t, reflect.Indirect(p), name, elemType.Kind(), nil, nil)
		value.Set(p)
	}
}

// checkAllFields recursively checks that every field not listed in allowZeroed
// is not set to its zero value, that every slice is not empty, and that every
// map has at least one entry.  It makes an additional exception for structs
// which have no defined fields.
func checkAllFields[pStruct any](t *testing.T, st pStruct, allowZeroed []string) {
	v := reflect.ValueOf(st)
	if v.Kind() == reflect.Pointer {
		v = reflect.Indirect(v)
	}
	checkAllValueFields(t, v, "", allowZeroed)
}

func checkAllValueFields(t *testing.T, v reflect.Value, name string, allowedToBeZero []string) {
	fields := reflect.VisibleFields(v.Type())
	for _, field := range fields {
		if field.Anonymous {
			// all right, fine, keep your secrets
			continue
		}
		fieldName := field.Name
		if name != "" {
			fieldName = name + "." + field.Name
		}
		if slices.Contains(allowedToBeZero, fieldName) {
			continue
		}
		f := v.FieldByName(field.Name)
		var elemType reflect.Type
		switch field.Type.Kind() {
		case reflect.Array, reflect.Chan, reflect.Map, reflect.Pointer, reflect.Slice:
			elemType = field.Type.Elem()
		}
		checkValue(t, f, fieldName, field.Type.Kind(), elemType, allowedToBeZero)
	}
}

func checkValue(t *testing.T, value reflect.Value, name string, kind reflect.Kind, elemType reflect.Type, allowedToBeZero []string) {
	if kind != reflect.Invalid {
		switch kind {
		case reflect.Map:
			assert.Falsef(t, value.IsZero(), "map field %s not set when it was not already expected to be left unpopulated by conversion", name)
			keys := value.MapKeys()
			for i := 0; i < len(keys); i++ {
				v := value.MapIndex(keys[i])
				checkValue(t, v, name+"{"+keys[i].String()+"}", elemType.Kind(), nil, allowedToBeZero)
			}
		case reflect.Slice:
			assert.Falsef(t, value.IsZero(), "slice field %s not set when it was not already expected to be left unpopulated by conversion", name)
			for i := 0; i < value.Len(); i++ {
				v := value.Index(i)
				checkValue(t, v, name+"["+strconv.Itoa(i)+"]", elemType.Kind(), nil, allowedToBeZero)
			}
		case reflect.Struct:
			if fields := reflect.VisibleFields(value.Type()); len(fields) != 0 {
				// structs which are defined with no fields are okay
				assert.Falsef(t, value.IsZero(), "slice field %s not set when it was not already expected to be left unpopulated by conversion", name)
			}
			checkAllValueFields(t, value, name, allowedToBeZero)
		case reflect.Pointer:
			assert.Falsef(t, value.IsZero(), "pointer field %s not set when it was not already expected to be left unpopulated by conversion", name)
			checkValue(t, reflect.Indirect(value), name, elemType.Kind(), nil, allowedToBeZero)
		}
	}
}

func TestGoDockerclientConfigFromSchema2Config(t *testing.T) {
	var input manifest.Schema2Config
	fillAllFields(t, &input)
	output := GoDockerclientConfigFromSchema2Config(&input)
	// make exceptions for fields in "output" which have no corresponding field in "input"
	notInSchema2Config := []string{"CPUSet", "CPUShares", "DNS", "Memory", "KernelMemory", "MemorySwap", "MemoryReservation", "Mounts", "PortSpecs", "PublishService", "SecurityOpts", "VolumeDriver", "VolumesFrom"}
	checkAllFields(t, output, notInSchema2Config)
}

func TestSchema2ConfigFromGoDockerclientConfig(t *testing.T) {
	var input dockerclient.Config
	fillAllFields(t, &input)
	output := Schema2ConfigFromGoDockerclientConfig(&input)
	// make exceptions for fields in "output" which have no corresponding field in "input"
	notInDockerConfig := []string{}
	checkAllFields(t, output, notInDockerConfig)
}
