package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureSessionIDReusesValidCookie(t *testing.T) {
	sessionID := "4a1c8f0b-41b2-4c43-8991-75c5950bc04a"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	rec := httptest.NewRecorder()

	if got := ensureSessionID(rec, req); got != sessionID {
		t.Fatalf("ensureSessionID() = %q, want %q", got, sessionID)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("valid existing session should not set a replacement cookie")
	}
}

func TestEnsureSessionIDReplacesInvalidCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "not-a-session"})
	rec := httptest.NewRecorder()

	got := ensureSessionID(rec, req)
	if !validSessionID(got) {
		t.Fatalf("ensureSessionID() returned invalid session %q", got)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value != got {
		t.Fatalf("replacement cookie = %+v, want %s=%s", cookies, sessionCookieName, got)
	}
}

func TestWorkflowIDForOrderAndLabel(t *testing.T) {
	sessionID := "4a1c8f0b-41b2-4c43-8991-75c5950bc04a"
	workflowID := workflowIDForOrder(sessionID, 42)

	if !workflowBelongsToSession(sessionID, workflowID) {
		t.Fatalf("%q should belong to session %q", workflowID, sessionID)
	}
	if workflowBelongsToSession("fd10f8d0-70b3-47d8-93f3-c589f7675ea0", workflowID) {
		t.Fatalf("%q must not belong to another session", workflowID)
	}
	if got := orderLabel(workflowID); got != "42" {
		t.Fatalf("orderLabel(%q) = %q, want 42", workflowID, got)
	}
	if got := orderLabel("order-42"); got != "order-42" {
		t.Fatalf("legacy order label = %q, want order-42", got)
	}
}
