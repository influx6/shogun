package ast

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// AnnotationDeclaration defines a annotation type which holds detail about a giving annotation.
type AnnotationDeclaration struct {
	Name      string                 `json:"name"`
	Template  string                 `json:"template"`
	Arguments []string               `json:"arguments"`
	Params    map[string]string      `json:"params"`
	Attrs     map[string]interface{} `json:"attrs"`
	Defer     bool                   `json:"defer"`
}

// HasArg returns true/false if the giving AnnotationDeclaration has a giving key in its Arguments.
func (ad AnnotationDeclaration) HasArg(name string) bool {
	for _, item := range ad.Arguments {
		if item == name {
			return true
		}
	}

	return false
}

// Param returns the associated  param value with giving key ("name").
func (ad AnnotationDeclaration) Param(name string) string {
	return ad.Params[name]
}

// Attr returns the associated  param value with giving key ("name").
func (ad AnnotationDeclaration) Attr(name string) interface{} {
	return ad.Attrs[name]
}

// ReadAnnotationsFromCommentry returns a slice of all annotation passed from the provided list.
func ReadAnnotationsFromCommentry(r io.Reader) []AnnotationDeclaration {
	var annotations []AnnotationDeclaration

	reader := bufio.NewReader(r)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		trimmedline := string(cleanWord([]byte(line)))
		if trimmedline == "" {
			continue
		}

		// Do we have a annotation here?
		if !strings.HasPrefix(trimmedline, "@") {
			continue
		}

		params := make(map[string]string, 0)

		if !strings.Contains(trimmedline, "(") {
			annotations = append(annotations, AnnotationDeclaration{Name: trimmedline, Params: params, Attrs: make(map[string]interface{})})
			continue
		}

		argIndex := strings.IndexRune(trimmedline, '(')
		argName := trimmedline[:argIndex]
		argContents := trimmedline[argIndex:]

		// Do we have a template associated with this annotation, if not, split the
		// commas and let those be our arguments.
		if !strings.HasSuffix(argContents, "{") {

			argContents = strings.TrimPrefix(strings.TrimSuffix(argContents, ")"), "(")

			var parts []string

			for _, part := range strings.Split(argContents, ",") {
				trimmed := strings.TrimSpace(part)
				if trimmed == "" {
					continue
				}

				parts = append(parts, trimmed)

				// If we are dealing with key value pairs then split, trimspace and set
				// in params. We only expect 2 values, any more and we wont consider the rest.
				if kvPieces := strings.Split(trimmed, "=>"); len(kvPieces) > 1 {
					val := strings.TrimSpace(kvPieces[1])
					params[strings.TrimSpace(kvPieces[0])] = val
				}
			}

			var deferred bool

			// Find out if we are to be deferred
			defered, ok := params["defer"]
			if !ok {
				defered = params["Defer"]
			}

			deferred, _ = strconv.ParseBool(defered)

			annotations = append(annotations, AnnotationDeclaration{
				Arguments: parts,
				Name:      argName,
				Params:    params,
				Defer:     deferred,
				Attrs:     make(map[string]interface{}),
			})

			continue
		}

		templateIndex := strings.IndexRune(argContents, '{')
		templateArgs := argContents[:templateIndex]
		templateArgs = strings.TrimPrefix(strings.TrimSuffix(templateArgs, ")"), "(")

		var parts []string

		for _, part := range strings.Split(templateArgs, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}

			parts = append(parts, trimmed)

			// If we are dealing with key value pairs then split, trimspace and set
			// in params. We only expect 2 values, any more and we wont consider the rest.
			if kvPieces := strings.Split(trimmed, "=>"); len(kvPieces) > 1 {
				val := strings.TrimSpace(kvPieces[1])
				params[strings.TrimSpace(kvPieces[0])] = val
			}
		}

		template := strings.TrimSpace(readTemplate(reader))

		var asJSON bool
		for _, item := range parts {
			if item == "asJSON" {
				asJSON = true
				break
			}
		}

		if _, ok := params["asJSON"]; ok {
			asJSON = true
		}

		var attrs map[string]interface{}

		if asJSON {
			if err := json.Unmarshal([]byte(template), &attrs); err == nil {
				template = ""
			}
		} else {
			attrs = make(map[string]interface{})
		}

		annotations = append(annotations, AnnotationDeclaration{
			Arguments: parts,
			Name:      argName,
			Template:  template,
			Params:    params,
			Attrs:     attrs,
		})

	}

	return annotations
}

var ending = []byte("})")
var newline = []byte("\n")
var empty = []byte("")
var singleComment = []byte("//")
var multiComment = []byte("/*")
var multiCommentItem = []byte("*")
var commentry = regexp.MustCompile(`\s*?([\/\/*|\*|\/]+)`)

func readTemplate(reader *bufio.Reader) string {
	var bu bytes.Buffer

	var seenEnd bool

	for {
		// Do we have another pending prefix, if so, we are at the ending, so return.
		if seenEnd {
			data, _ := reader.Peek(100)
			dataVal := commentry.ReplaceAllString(string(data), "")
			dataVal = string(cleanWord([]byte(dataVal)))

			// fmt.Printf("Peek2: %+q -> %+q\n", dataVal, data)

			if strings.HasPrefix(string(dataVal), "@") {
				return bu.String()
			}

			// If it's all space, then return.
			// if strings.TrimSpace(string(dataVal)) == "" {
			return bu.String()
			// }
		}

		twoWord, err := reader.ReadString('\n')
		if err != nil {
			bu.WriteString(twoWord)
			return bu.String()
		}

		twoWorded := cleanWord([]byte(twoWord))
		// fmt.Printf("Ending: %+q -> %t\n", twoWorded, bytes.HasPrefix(twoWorded, ending))

		if bytes.HasPrefix(twoWorded, ending) {
			seenEnd = true
			continue
		}

		bu.WriteString(twoWord)
	}

}

func cleanWord(word []byte) []byte {
	word = bytes.TrimSpace(word)
	word = bytes.TrimPrefix(word, singleComment)
	word = bytes.TrimPrefix(word, multiComment)
	word = bytes.TrimPrefix(word, multiCommentItem)
	return bytes.TrimSpace(word)
}
