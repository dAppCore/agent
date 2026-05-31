// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"errors"
	"testing"

	core "dappco.re/go"
)

// --- failureResult ---

// TestFailureResult_ErrorValue_Good — when the result Value is an error,
// failureResult wraps it as a Fail.
func TestFailureResult_ErrorValue_Good(t *testing.T) {
	err := core.E("test.op", "something broke", nil)
	result := core.Result{Value: err, OK: false}
	r := failureResult("test.action", "fallback msg", result)

	if r.OK {
		t.Fatal("expected Fail, got OK")
	}
	if r.Value == nil {
		t.Fatal("expected error value, got nil")
	}
	errVal, ok := r.Value.(error)
	if !ok {
		t.Fatalf("expected error type, got %T", r.Value)
	}
	if !core.Contains(errVal.Error(), "something broke") {
		t.Errorf("error message = %q; want containing 'something broke'", errVal.Error())
	}
}

// TestFailureResult_StringValue_Good — when result Value is a non-empty
// string, failureResult uses it as the error message.
func TestFailureResult_StringValue_Good(t *testing.T) {
	result := core.Result{Value: "custom message", OK: false}
	r := failureResult("test.action", "fallback msg", result)

	if r.OK {
		t.Fatal("expected Fail, got OK")
	}
	err, ok := r.Value.(error)
	if !ok {
		t.Fatalf("expected error type, got %T", r.Value)
	}
	if !core.Contains(err.Error(), "custom message") {
		t.Errorf("error message = %q; want containing 'custom message'", err.Error())
	}
}

// TestFailureResult_NilValue_Good — when result Value is nil (and not an
// error), failureResult uses the fallback message.
func TestFailureResult_NilValue_Good(t *testing.T) {
	result := core.Result{Value: nil, OK: false}
	r := failureResult("test.action", "fallback msg", result)

	if r.OK {
		t.Fatal("expected Fail, got OK")
	}
	err, ok := r.Value.(error)
	if !ok {
		t.Fatalf("expected error type, got %T", r.Value)
	}
	if !core.Contains(err.Error(), "fallback msg") {
		t.Errorf("error message = %q; want containing 'fallback msg'", err.Error())
	}
}

// TestFailureResult_EmptyStringValue_Good — when result Value is an
// empty string, failureResult uses the fallback.
func TestFailureResult_EmptyStringValue_Good(t *testing.T) {
	result := core.Result{Value: "", OK: false}
	r := failureResult("test.action", "fallback msg", result)

	if r.OK {
		t.Fatal("expected Fail, got OK")
	}
	err, _ := r.Value.(error)
	if !core.Contains(err.Error(), "fallback msg") {
		t.Errorf("error message = %q; want containing 'fallback msg'", err.Error())
	}
}

// TestFailureResult_BoolValue_Ugly — when result Value is a bool,
// stringValue converts it to "false" (non-empty), so it's used as the
// error message rather than the fallback.
func TestFailureResult_BoolValue_Ugly(t *testing.T) {
	result := core.Result{Value: false, OK: false}
	r := failureResult("test.action", "fallback msg", result)

	if r.OK {
		t.Fatal("expected Fail, got OK")
	}
	err, _ := r.Value.(error)
	if !core.Contains(err.Error(), "false") {
		t.Errorf("error message = %q; want containing 'false'", err.Error())
	}
}

// --- typedResultValue ---

// TestTypedResultValue_OKWithCorrectType_Good — when the result is OK
// and the value matches T, typedResultValue returns it unchanged shape.
func TestTypedResultValue_OKWithCorrectType_Good(t *testing.T) {
	result := core.Ok("hello")
	r := typedResultValue[string]("test.action", "invalid type", result)

	if !r.OK {
		t.Fatalf("expected OK, got Fail: %v", r.Error())
	}
	val, ok := r.Value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", r.Value)
	}
	if val != "hello" {
		t.Errorf("value = %q; want hello", val)
	}
}

// TestTypedResultValue_OKWithInt_Good — typedResultValue works with
// integer types.
func TestTypedResultValue_OKWithInt_Good(t *testing.T) {
	result := core.Ok(42)
	r := typedResultValue[int]("test.action", "invalid int", result)

	if !r.OK {
		t.Fatalf("expected OK, got Fail: %v", r.Error())
	}
	val, ok := r.Value.(int)
	if !ok {
		t.Fatalf("expected int, got %T", r.Value)
	}
	if val != 42 {
		t.Errorf("value = %d; want 42", val)
	}
}

// TestTypedResultValue_NotOK_Bad — when the result is Fail,
// typedResultValue passes through unchanged.
func TestTypedResultValue_NotOK_Bad(t *testing.T) {
	err := errors.New("original error")
	result := core.Fail(err)
	r := typedResultValue[string]("test.action", "invalid", result)

	if r.OK {
		t.Fatal("expected Fail, got OK")
	}
	if !core.Contains(r.Error(), "original error") {
		t.Errorf("error = %q; want containing 'original error'", r.Error())
	}
}

// TestTypedResultValue_WrongType_Bad — when the result is OK but the
// value type doesn't match T, typedResultValue returns Fail.
func TestTypedResultValue_WrongType_Bad(t *testing.T) {
	result := core.Ok(42) // int, but we ask for string
	r := typedResultValue[string]("test.action", "invalid type", result)

	if r.OK {
		t.Fatal("expected Fail for wrong type, got OK")
	}
	if !core.Contains(r.Error(), "invalid type") {
		t.Errorf("error = %q; want containing 'invalid type'", r.Error())
	}
}

// TestTypedResultValue_NilValue_Ugly — when result is OK but Value is
// nil, typedResultValue returns Fail.
func TestTypedResultValue_NilValue_Ugly(t *testing.T) {
	result := core.Result{Value: nil, OK: true}
	r := typedResultValue[string]("test.action", "invalid nil", result)

	if r.OK {
		t.Fatal("expected Fail for nil value, got OK")
	}
}

// TestTypedResultValue_Struct_Good — typedResultValue works with struct
// types.
func TestTypedResultValue_Struct_Good(t *testing.T) {
	type myStruct struct {
		Name string
		Age  int
	}
	result := core.Ok(myStruct{Name: "test", Age: 30})
	r := typedResultValue[myStruct]("test.action", "invalid struct", result)

	if !r.OK {
		t.Fatalf("expected OK, got Fail: %v", r.Error())
	}
	val, ok := r.Value.(myStruct)
	if !ok {
		t.Fatalf("expected myStruct, got %T", r.Value)
	}
	if val.Name != "test" || val.Age != 30 {
		t.Errorf("value = %+v; want {Name:test Age:30}", val)
	}
}

// --- toolHandlerFor ---

// TestToolHandlerFor_Success_Good — a successful handler must return the
// typed value and nil error.
func TestToolHandlerFor_Success_Good(t *testing.T) {
	handler := toolHandlerFor[string, string](
		"test.action", "invalid",
		func(ctx context.Context, input string) core.Result {
			return core.Ok("result: " + input)
		},
	)

	_, out, err := handler(context.Background(), nil, "hello")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if out != "result: hello" {
		t.Errorf("out = %q; want 'result: hello'", out)
	}
}

// TestToolHandlerFor_Failure_Bad — when the handler returns Fail,
// toolHandlerFor returns an error.
func TestToolHandlerFor_Failure_Bad(t *testing.T) {
	handler := toolHandlerFor[string, string](
		"test.action", "invalid",
		func(ctx context.Context, input string) core.Result {
			return core.Fail(core.E("test.action", "handler failed", nil))
		},
	)

	_, _, err := handler(context.Background(), nil, "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !core.Contains(err.Error(), "handler failed") {
		t.Errorf("error = %q; want containing 'handler failed'", err.Error())
	}
}

// TestToolHandlerFor_WrongType_Bad — when the handler returns a value
// of the wrong type, toolHandlerFor returns an error.
func TestToolHandlerFor_WrongType_Bad(t *testing.T) {
	handler := toolHandlerFor[string, int](
		"test.action", "invalid type",
		func(ctx context.Context, input string) core.Result {
			return core.Ok("not an int")
		},
	)

	_, _, err := handler(context.Background(), nil, "hello")
	if err == nil {
		t.Fatal("expected error for wrong type, got nil")
	}
	if !core.Contains(err.Error(), "invalid type") {
		t.Errorf("error = %q; want containing 'invalid type'", err.Error())
	}
}

// TestToolHandlerFor_StructInputOutput_Good — toolHandlerFor works with
// struct input and output types.
func TestToolHandlerFor_StructInputOutput_Good(t *testing.T) {
	type req struct {
		Name string
	}
	type resp struct {
		Greeting string
	}

	handler := toolHandlerFor[req, resp](
		"test.action", "invalid struct",
		func(ctx context.Context, input req) core.Result {
			return core.Ok(resp{Greeting: "Hello, " + input.Name})
		},
	)

	_, out, err := handler(context.Background(), nil, req{Name: "World"})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if out.Greeting != "Hello, World" {
		t.Errorf("Greeting = %q; want 'Hello, World'", out.Greeting)
	}
}

// TestToolHandlerFor_HandlerPanic_Ugly — if the handler function panics,
// the test must not crash (this is an edge-case guard).
func TestToolHandlerFor_HandlerPanic_Ugly(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered from panic as expected: %v", r)
		}
	}()

	handler := toolHandlerFor[string, string](
		"test.action", "invalid",
		func(ctx context.Context, input string) core.Result {
			panic("unexpected panic in handler")
		},
	)

	// This may panic; the defer above catches it.
	handler(context.Background(), nil, "boom")
}
