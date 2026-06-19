package dashboard

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	sessionCookieName = "pizza_session"
	sessionCookieTTL  = 24 * time.Hour
)

// ensureSessionID returns the browser's stable demo session ID, creating a
// cookie-backed one when the request has none or carries an invalid value.
func ensureSessionID(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(sessionCookieName); err == nil && validSessionID(c.Value) {
		return c.Value
	}
	id := uuid.NewString()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int(sessionCookieTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	return id
}

func validSessionID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func workflowIDForOrder(sessionID string, orderID int) string {
	return fmt.Sprintf("order-%s-%d", sessionID, orderID)
}

func workflowBelongsToSession(sessionID, workflowID string) bool {
	return strings.HasPrefix(workflowID, "order-"+sessionID+"-")
}
