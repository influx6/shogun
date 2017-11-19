package tests

import (
	"fmt"
	"log"
	"os"
	"testing"
)

// succeedMark is the Unicode codepoint for a check mark.
const succeedMark = "\u2713"

var logger = log.New(os.Stdout, "", log.Lshortfile)

// Info logs the info message using the giving message and values.
func Info(message string, val ...interface{}) {
	if testing.Verbose() {
		logger.Output(2, fmt.Sprintf("\t-\t %s\n", fmt.Sprintf(message, val...)))
	}
}

// Passed logs the failure message using the giving message and values.
func Passed(message string, val ...interface{}) {
	if testing.Verbose() {
		logger.Output(2, fmt.Sprintf("\t%s\t %s\n", succeedMark, fmt.Sprintf(message, val...)))
	}
}

// failedMark is the Unicode codepoint for an X mark.
const failedMark = "\u2717"

// Failed logs the failure message using the giving message and values.
// Failed calls os.Exit(1) after printing.
func Failed(message string, val ...interface{}) {
	if testing.Verbose() {
		logger.Output(2, fmt.Sprintf("\t%s\t %s\n", failedMark, fmt.Sprintf(message, val...)))
	}

	os.Exit(1)
}

// FailedWithError logs the failure message using the giving message and values.
// It also shows the error under the comment.
// FailedWithError calls os.Exit(1) after printing.
func FailedWithError(err error, message string, val ...interface{}) {
	if testing.Verbose() {
		logger.Output(2, fmt.Sprintf("\t%s\t %s\n", failedMark, fmt.Sprintf(message, val...)))
		if err != nil {
			logger.Output(2, fmt.Sprintf("\t%s\t Error: %+q\n", "-", err))
		}
	}

	os.Exit(1)
}

// Errored logs the error message using the giving message and values.
func Errored(message string, val ...interface{}) {
	if testing.Verbose() {
		logger.Output(2, fmt.Sprintf("\t%s\t %s\n", failedMark, fmt.Sprintf(message, val...)))
	}
}

// ErroredWithError logs the failure message using the giving message and values.
// It also shows the error under the comment.
func ErroredWithError(err error, message string, val ...interface{}) {
	if testing.Verbose() {
		logger.Output(2, fmt.Sprintf("\t%s\t %s\n", failedMark, fmt.Sprintf(message, val...)))
		if err != nil {
			logger.Output(2, fmt.Sprintf("\t%s\t Error: %+q\n", "-", err))
		}
	}
}
