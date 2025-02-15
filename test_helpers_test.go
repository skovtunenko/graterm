package graterm

import (
	"reflect"
	"testing"
)

// assertNotNil fails if the actual value is nil.
func assertNotNil(t *testing.T, actual interface{}) {
	t.Helper()
	if actual == nil {
		t.Fatal("expected non-nil value")
	}
}

// assertEqual fails if expected and actual are not equal.
func assertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// assertError fails if err is nil.
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
}

// assertNoError fails if err is not nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertTrue fails if value is not true.
func assertTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatal("expected true")
	}
}
