package internals

import (
	"strconv"
	"strings"
	"time"
)

// Flags defines a type of Flag slice which exposes a method that attempts to
// load values of flags either from env or from a provided flag list of
// `key=value` value pairs.
type Flags []Flag

// Load attempts to load flag values from slice list of `key=value` pairs
// else if flag supports environment variables, will attempt to load throug that
// instead. It returns a map of all loaded values and an error.
func (f Flags) Load(args []string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if len(args) == 0 {
		return values, nil
	}

	for _, flag := range f {
		if flag.Type == BadFlag {
			continue
		}

		val, ok := flag.FromList(args)
		if !ok && !flag.UsesEnv() {
			continue
		}

		if !ok && flag.UsesEnv() {
			val, ok = flag.FromEnv()
			if !ok {
				continue
			}
		}

		if val == "" || strings.TrimSpace(val) == "" {
			continue
		}

		switch flag.Type {
		case Float64Flag:
			vald, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case DurationFlag:
			vald, err := time.ParseDuration(val)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case TBoolFlag, BoolFlag:
			values[flag.Name] = val
		case StringFlag:
			values[flag.Name] = val
		case UintFlag:
			vald, err := strconv.ParseUint(val, 0, 64)
			if err != nil {
				return values, err
			}

			values[flag.Name] = uint(vald)
		case Uint64Flag:
			vald, err := strconv.ParseUint(val, 0, 64)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case IntFlag:
			vald, err := strconv.ParseInt(val, 0, 64)
			if err != nil {
				return values, err
			}

			values[flag.Name] = int(vald)
		case Int64Flag:
			vald, err := strconv.ParseInt(val, 0, 64)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case IntSliceFlag:
			vald, err := StringToIntSlice(val)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case Int64SliceFlag:
			vald, err := StringToInt64Slice(val)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case BoolSliceFlag:
			vald, err := StringToBoolSlice(val)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case Float64SliceFlag:
			vald, err := StringToFloat64Slice(val)
			if err != nil {
				return values, err
			}

			values[flag.Name] = vald
		case StringSliceFlag:
			values[flag.Name] = strings.Split(val, ",")
		}
	}

	return values, nil
}

// StringToBoolSlice returns a bool slice from a comma seperated string.
func StringToBoolSlice(arg string) ([]bool, error) {
	var vals []bool

	for _, val := range strings.Split(arg, ",") {
		bval, err := strconv.ParseBool(val)
		if err != nil {
			return vals, err
		}

		vals = append(vals, bval)
	}

	return vals, nil
}

// StringToIntSlice returns a int slice from a comma seperated string.
func StringToIntSlice(arg string) ([]int, error) {
	var vals []int

	for _, val := range strings.Split(arg, ",") {
		intval, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return vals, err
		}

		vals = append(vals, int(intval))
	}

	return vals, nil
}

// StringToFloat64Slice returns a int64 slice from a comma seperated string.
func StringToFloat64Slice(arg string) ([]float64, error) {
	var vals []float64

	for _, val := range strings.Split(arg, ",") {
		intval, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return vals, err
		}

		vals = append(vals, intval)
	}

	return vals, nil
}

// StringToInt64Slice returns a int64 slice from a comma seperated string.
func StringToInt64Slice(arg string) ([]int64, error) {
	var vals []int64

	for _, val := range strings.Split(arg, ",") {
		intval, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return vals, err
		}

		vals = append(vals, intval)
	}

	return vals, nil
}

// StringToSlice turns a comma seperate string and returns a slice of all parts.
func StringToSlice(arg string) []string {
	return strings.Split(arg, ",")
}

// FilterFlags runs through a list of values and filters
// out values with `-` or `--` prefix has flags and others without `-`/`--`
// as non flags.
func FilterFlags(args []string) (flags, nonflags []string) {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		// Ignore -- only content
		if arg == "--" || arg == "" {
			continue
		}

		if strings.HasPrefix(arg, "--") {
			flags = append(flags, strings.TrimPrefix(arg, "--"))
			continue
		}

		if strings.HasPrefix(arg, "-") {
			flags = append(flags, strings.TrimPrefix(arg, "-"))
			continue
		}

		nonflags = append(nonflags, arg)
	}

	return
}
