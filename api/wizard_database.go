package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

var databaseConfig = resourceWizardConfig{Name: "database", Title: "Database"}

func runDatabaseCommand(ctx context.Context, operation string, params wizardParams) ([]byte, error) {
	return runResourceWizardCommand(ctx, "database", operation, params)
}
func DatabaseWizardHandler(w http.ResponseWriter, r *http.Request) {
	serveResourceWizard(w, r, runDatabaseCommand, databaseConfig)
}

// Advanced database fields can contain credentials. Preserve Ruby's Back values
// using an authenticated, encrypted cookie rather than storing them in plaintext.
var databaseStateCipher = newDatabaseStateCipher()

func newDatabaseStateCipher() cipher.AEAD {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	state, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return state
}
func rememberDatabaseValues(w http.ResponseWriter, r *http.Request, params wizardParams) {
	data, _ := json.Marshal(struct {
		Params  wizardParams
		Expires int64
	}{params, time.Now().Add(time.Hour).Unix()})
	if len(data) > 2400 {
		return
	}
	nonce := make([]byte, databaseStateCipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return
	}
	encrypted := databaseStateCipher.Seal(nonce, nonce, data, []byte("database"))
	http.SetCookie(w, &http.Cookie{Name: "hawk_wizard_database", Value: base64.RawURLEncoding.EncodeToString(encrypted), Path: "/cib/live/wizards/database", MaxAge: 3600, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
}
func recalledDatabaseValues(r *http.Request) wizardParams {
	empty := wizardParams{}
	cookie, err := r.Cookie("hawk_wizard_database")
	if err != nil {
		return empty
	}
	encrypted, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	size := databaseStateCipher.NonceSize()
	if err != nil || len(encrypted) < size {
		return empty
	}
	data, err := databaseStateCipher.Open(nil, encrypted[:size], encrypted[size:], []byte("database"))
	if err != nil {
		return empty
	}
	var state struct {
		Params  wizardParams
		Expires int64
	}
	if json.Unmarshal(data, &state) != nil || state.Expires <= time.Now().Unix() || state.Params == nil {
		return empty
	}
	return state.Params
}
