package services

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

type dummyOutput struct {
	msgs []*models.StreamMessage
}

func (d *dummyOutput) Send(msg *models.StreamMessage) {
	d.msgs = append(d.msgs, msg)
}

func TestParseAndCollectToolCall(t *testing.T) {
	s := &ChatService{l: logrus.NewEntry(logrus.New())}
	var toolCalls []models.ToolCall
	dummy := &dummyOutput{}
	output := make(chan *models.StreamMessage, 10)
	go func() {
		for msg := range output {
			dummy.Send(msg)
		}
	}()

	tests := []struct {
		name     string
		input    string
		expects  bool
		funcName string
		args     string
	}{
		{
			name:     "Structured JSON",
			input:    `{"name": "myfunc", "arguments": {"a": 1, "b": [2,3]}}`,
			expects:  true,
			funcName: "myfunc",
			args:     `{"a":1,"b":[2,3]}`,
		},
		{
			name:     "Nested Parentheses",
			input:    `Function call: myfunc(a, nested(b, c(d, e)), f)`,
			expects:  true,
			funcName: "myfunc",
			args:     "a, nested(b, c(d, e)), f",
		},
		{
			name:     "JSON Arguments",
			input:    `Function call: myfunc({"foo": [1,2], "bar": {"baz": 3}})`,
			expects:  true,
			funcName: "myfunc",
			args:     `{"foo": [1,2], "bar": {"baz": 3}}`,
		},
		{
			name:    "Invalid Format",
			input:   `Function call: notvalid`,
			expects: false,
		},
		{
			name:    "Completely Invalid",
			input:   `Just some text`,
			expects: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toolCalls = nil
			msg := &models.StreamMessage{Content: tc.input}
			s.parseAndCollectToolCall(msg, &toolCalls, output, "sess1")
			if tc.expects {
				if len(toolCalls) == 0 {
					t.Fatalf("Expected tool call to be parsed, got none")
				}
				if toolCalls[0].Function.Name != tc.funcName {
					t.Errorf("Expected function name %q, got %q", tc.funcName, toolCalls[0].Function.Name)
				}
				if tc.name == "Structured JSON" {
					// Compare JSON arguments ignoring spacing
					var want, got interface{}
					if err := json.Unmarshal([]byte(tc.args), &want); err != nil {
						t.Fatalf("Invalid expected JSON: %v", err)
					}
					if err := json.Unmarshal([]byte(toolCalls[0].Function.Arguments), &got); err != nil {
						t.Fatalf("Invalid parsed JSON: %v", err)
					}
					if !jsonEqual(want, got) {
						t.Errorf("Expected arguments %v, got %v", want, got)
					}
				} else if toolCalls[0].Function.Arguments != tc.args {
					t.Errorf("Expected arguments %q, got %q", tc.args, toolCalls[0].Function.Arguments)
				}
			} else {
				if len(toolCalls) != 0 {
					t.Errorf("Expected no tool call to be parsed, got %+v", toolCalls)
				}
			}
		})
	}
	close(output)
}

func jsonEqual(a, b interface{}) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
