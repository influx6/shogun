package custom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"gopkg.in/mgo.v2/bson"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/reflection"
)

var (
	red     = color.New(color.FgRed)
	green   = color.New(color.FgGreen)
	white   = color.New(color.FgWhite)
	yellow  = color.New(color.FgHiYellow)
	magenta = color.New(color.FgHiMagenta)
)

// FlatDisplay writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  Message: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
//  Message: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
func FlatDisplay(w io.Writer) metrics.Processors {
	return FlatDisplayWith(w, "Message:", nil)
}

// FlatDisplayWith writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  [Header]: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
//  [Header]: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
func FlatDisplayWith(w io.Writer, header string, filterFn func(metrics.Entry) bool) metrics.Processors {
	return NewEmitter(w, func(en metrics.Entry) []byte {
		if filterFn != nil && !filterFn(en) {
			return nil
		}

		var bu bytes.Buffer
		bu.WriteString("\n")

		if header != "" {
			fmt.Fprintf(&bu, "%s %+s", green.Sprint(header), printAtLevel(en.Level, en.Message))
		} else {
			fmt.Fprintf(&bu, "%+s", printAtLevel(en.Level, en.Message))
		}

		fmt.Fprint(&bu, printSpaceLine(2))

		if en.Function != "" {
			fmt.Fprintf(&bu, "%s: %+s\n", green.Sprint("Function"), en.Function)
			fmt.Fprint(&bu, printSpaceLine(2))
			fmt.Fprintf(&bu, "%s: %+s:%d", green.Sprint("File"), en.File, en.Line)
			fmt.Fprint(&bu, printSpaceLine(2))
		}

		fmt.Fprint(&bu, printSpaceLine(2))

		for key, value := range en.Field {
			fmt.Fprintf(&bu, "%+s: %+s", green.Sprint(key), printValue(value))
			fmt.Fprint(&bu, printSpaceLine(2))
		}

		bu.WriteString("\n")
		return bu.Bytes()
	})
}

//=====================================================================================

// BlockDisplay writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  Message: We must create new standard behaviour
//	Function: BuildPack
//  +-----------------------------+------------------------------+
//  | displayrange.address.bolder | "No 20 tokura flag"          |
//  +-----------------------------+------------------------------+
//  +--------------------------+----------+
//  | displayrange.bolder.size |  20      |
//  +--------------------------+----------+
//
func BlockDisplay(w io.Writer) metrics.Processors {
	return BlockDisplayWith(w, "Message:", nil)
}

// BlockDisplayWith writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  Message: We must create new standard behaviour
//	Function: BuildPack
//  +-----------------------------+------------------------------+
//  | displayrange.address.bolder | "No 20 tokura flag"          |
//  +-----------------------------+------------------------------+
//  +--------------------------+----------+
//  | displayrange.bolder.size |  20      |
//  +--------------------------+----------+
//
func BlockDisplayWith(w io.Writer, header string, filterFn func(metrics.Entry) bool) metrics.Processors {
	return NewEmitter(w, func(en metrics.Entry) []byte {
		if filterFn != nil && !filterFn(en) {
			return nil
		}

		var bu bytes.Buffer
		if header != "" {
			fmt.Fprintf(&bu, "%s %+s\n", green.Sprint(header), printAtLevel(en.Level, en.Message))
		} else {
			fmt.Fprintf(&bu, "%+s\n", printAtLevel(en.Level, en.Message))
		}

		if en.Function != "" {
			fmt.Fprintf(&bu, "%s: %+s\n", green.Sprint("Function"), en.Function)
			fmt.Fprintf(&bu, "%s: %+s:%d\n", green.Sprint("File"), en.File, en.Line)
		}

		print(en.Field, func(key []string, value string) {
			keyVal := strings.Join(key, ".")
			keyLength := len(keyVal) + 2
			valLength := len(value) + 2

			keyLines := printBlockLine(keyLength)
			valLines := printBlockLine(valLength)
			spaceLines := printSpaceLine(1)

			fmt.Fprintf(&bu, "+%s+%s+\n", keyLines, valLines)
			fmt.Fprintf(&bu, "|%s%s%s|%s%s%s|\n", spaceLines, green.Sprint(keyVal), spaceLines, spaceLines, value, spaceLines)
			fmt.Fprintf(&bu, "+%s+%s+", keyLines, valLines)
			fmt.Fprintf(&bu, "\n")

		})

		bu.WriteString("\n")
		return bu.Bytes()
	})
}

//=====================================================================================

// StackDisplay writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  Message: We must create new standard behaviour
//	Function: BuildPack
//  - displayrange.address.bolder: "No 20 tokura flag"
//  - displayrange.bolder.size:  20
//
func StackDisplay(w io.Writer) metrics.Processors {
	return StackDisplayWith(w, "Message:", "-", nil)
}

// StackDisplayWith writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  [Header]: We must create new standard behaviour
//	Function: BuildPack
//  [tag] displayrange.address.bolder: "No 20 tokura flag"
//  [tag] displayrange.bolder.size:  20
//
func StackDisplayWith(w io.Writer, header string, tag string, filterFn func(metrics.Entry) bool) metrics.Processors {
	return NewEmitter(w, func(en metrics.Entry) []byte {
		if filterFn != nil && !filterFn(en) {
			return nil
		}

		var bu bytes.Buffer
		if header != "" {
			fmt.Fprintf(&bu, "%s %+s\n", green.Sprint(header), printAtLevel(en.Level, en.Message))
		} else {
			fmt.Fprintf(&bu, "%+s\n", printAtLevel(en.Level, en.Message))
		}

		if tag == "" {
			tag = "-"
		}

		if en.Function != "" {
			fmt.Fprintf(&bu, "%s: %+s\n", green.Sprint("Function"), en.Function)
			fmt.Fprintf(&bu, "%s: %+s:%d\n", green.Sprint("File"), en.File, en.Line)
		}

		print(en.Field, func(key []string, value string) {
			fmt.Fprintf(&bu, "%s %s: %+s\n", tag, green.Sprintf(strings.Join(key, ".")), value)
		})

		bu.WriteString("\n")
		return bu.Bytes()
	})
}

//=====================================================================================

// Emitter emits all entries into the entries into a sink io.writer after
// transformation from giving transformer function..
type Emitter struct {
	Sink      io.Writer
	Transform func(metrics.Entry) []byte
}

// NewEmitter returns a new instance of Emitter.
func NewEmitter(w io.Writer, transform func(metrics.Entry) []byte) *Emitter {
	return &Emitter{
		Sink:      w,
		Transform: transform,
	}
}

// Handle implements the metrics.metrics interface.
func (ce *Emitter) Handle(e metrics.Entry) error {
	_, err := ce.Sink.Write(ce.Transform(e))
	return err
}

//=====================================================================================

func printAtLevel(lvl metrics.Level, message string) string {
	switch lvl {
	case metrics.ErrorLvl:
		return red.Sprint(message)
	case metrics.InfoLvl:
		return white.Sprint(message)
	case metrics.RedAlertLvl:
		return magenta.Sprint(message)
	case metrics.YellowAlertLvl:
		return yellow.Sprint(message)
	}

	return message
}

func printSpaceLine(length int) string {
	var lines []string

	for i := 0; i < length; i++ {
		lines = append(lines, " ")
	}

	return strings.Join(lines, "")
}

func printBlockLine(length int) string {
	var lines []string

	for i := 0; i < length; i++ {
		lines = append(lines, "-")
	}

	return strings.Join(lines, "")
}

func print(item interface{}, do func(key []string, val string)) {
	printInDepth(item, do, 0)
}

type stringer interface {
	String() string
}

var maxDepth = 1000

func printInDepth(item interface{}, do func(key []string, val string), depth int) {
	if depth >= maxDepth {
		return
	}

	if item == nil {
		return
	}

	itemType := reflect.TypeOf(item)

	if itemType.Kind() == reflect.Ptr {
		itemType = itemType.Elem()
	}

	switch itemType.Kind() {
	case reflect.Array, reflect.Slice:
		printArrays(item, do, depth+1)
	case reflect.Struct:
		if val, err := reflection.ToMap("json", item, true); err == nil {
			if sm, ok := item.(stringer); ok {
				val["object.String"] = sm.String()
			}
			if sm, ok := item.(error); ok {
				val["object.ErrorMessage"] = sm.Error()
			}
			printMap(val, do, depth+1)
		}

	case reflect.Map:
		printMap(item, do, depth+1)
	default:
		do([]string{}, printValue(item))
	}
}

func printMap(items interface{}, do func(key []string, val string), depth int) {
	switch bo := items.(type) {
	case map[string]byte:
		for index, item := range bo {
			do([]string{index}, printValue(int(item)))
		}
	case map[string]float32:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string]float64:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string]int64:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string]int32:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string]int16:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string]time.Time:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string]int:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case bson.M:
		print(map[string]interface{}(bo), do)
	case map[string][]interface{}:
		for index, item := range bo {
			printInDepth(item, func(key []string, value string) {
				if index == "" {
					do(key, value)
					return
				}

				do(append([]string{index}, key...), value)
			}, depth+1)
		}
	case map[string]interface{}:
		for index, item := range bo {
			printInDepth(item, func(key []string, value string) {
				if index == "" {
					do(key, value)
					return
				}

				do(append([]string{index}, key...), value)
			}, depth+1)
		}
	case map[string]string:
		for index, item := range bo {
			do([]string{index}, printValue(item))
		}
	case map[string][]byte:
		for index, item := range bo {
			do([]string{index}, printValue(string(item)))
		}
	case metrics.Field:
		printMap((map[string]interface{})(bo), do, depth+1)
	}
}

func printArrays(items interface{}, do func(index []string, val string), depth int) {
	switch bo := items.(type) {
	case []metrics.Field:
		for index, item := range bo {
			printMap((map[string]interface{})(item), func(key []string, val string) {
				do(append([]string{printValue(index)}, key...), val)
			}, depth+1)
		}
	case []map[string][]byte:
		for index, item := range bo {
			printMap(item, func(key []string, val string) {
				do(append([]string{printValue(index)}, key...), val)
			}, depth+1)
		}
	case []map[string][]interface{}:
		for index, item := range bo {
			printMap(item, func(key []string, val string) {
				do(append([]string{printValue(index)}, key...), val)
			}, depth+1)
		}
	case []map[string]interface{}:
		for index, item := range bo {
			printMap(item, func(key []string, val string) {
				do(append([]string{printValue(index)}, key...), val)
			}, depth+1)
		}
	case []byte:
		do([]string{}, string(bo))
	case []bool:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []interface{}:
		for index, item := range bo {
			printInDepth(item, func(key []string, value string) {
				do(append([]string{printValue(index)}, key...), value)
			}, depth+1)
		}
	case []time.Time:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []string:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []int:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []int64:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []int32:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []int16:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []int8:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []float32:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	case []float64:
		for index, item := range bo {
			do([]string{printValue(index)}, printValue(item))
		}
	}
}

func printValue(item interface{}) string {
	switch bo := item.(type) {
	case stringer:
		return bo.String()
	case string:
		return `"` + bo + `"`
	case error:
		return bo.Error()
	case int:
		return strconv.Itoa(bo)
	case int8:
		return strconv.Itoa(int(bo))
	case int16:
		return strconv.Itoa(int(bo))
	case int64:
		return strconv.Itoa(int(bo))
	case time.Time:
		return bo.UTC().String()
	case rune:
		return strconv.QuoteRune(bo)
	case bool:
		return strconv.FormatBool(bo)
	case byte:
		return strconv.QuoteRune(rune(bo))
	case float64:
		return strconv.FormatFloat(bo, 'f', 4, 64)
	case float32:
		return strconv.FormatFloat(float64(bo), 'f', 4, 64)
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Sprintf("%#v", item)
	}

	return string(data)
}
