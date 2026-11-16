package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfilePage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	for _, value := range []string{"", "false", "true"} {
		request := httptest.NewRequest("GET", "/cib/live/profile/edit", nil)
		if value != "" {
			request.AddCookie(&http.Cookie{Name: "stonithwarning", Value: value})
		}
		response := httptest.NewRecorder()
		ProfileHandler(response, request)
		require.Equal(t, 200, response.Code)
		html := response.Body.String()
		require.Contains(t, html, `name="profile[language]" disabled aria-describedby="profile-language-help"`)
		require.Contains(t, html, "Language selection will be available in a future release.")
		require.Contains(t, html, `>English</option>`)
		if value == "false" {
			require.Contains(t, html, `value="false" selected`)
		} else {
			require.Contains(t, html, `value="true" selected`)
		}
		require.Empty(t, response.Result().Cookies())
	}
}

func TestProfileApplyPreservesLocale(t *testing.T) {
	for _, method := range []string{"POST", "PUT"} {
		for _, warning := range []string{"true", "false"} {
			form := url.Values{"profile[stonithwarning]": {warning}, "profile[language]": {"de"}}
			request := httptest.NewRequest(method, "/cib/live/profile", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(&http.Cookie{Name: "locale", Value: "fr"})
			response := httptest.NewRecorder()
			ProfileHandler(response, request)
			require.Equal(t, 303, response.Code)
			require.Equal(t, "/cib/live/profile/edit?updated=true", response.Header().Get("Location"))
			cookies := response.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, "stonithwarning", cookies[0].Name)
			require.Equal(t, "/", cookies[0].Path)
			if warning == "true" {
				require.Equal(t, -1, cookies[0].MaxAge)
			} else {
				require.Equal(t, "false", cookies[0].Value)
				require.Zero(t, cookies[0].MaxAge)
			}
		}
	}
}

func TestProfileRevertAndInvalidRequests(t *testing.T) {
	for _, test := range []struct {
		method, body string
		status       int
	}{
		{"POST", "revert=", 303},
		{"POST", "profile%5Bstonithwarning%5D=invalid", 422},
		{"DELETE", "", 405},
	} {
		request := httptest.NewRequest(test.method, "/cib/live/profile", strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		ProfileHandler(response, request)
		require.Equal(t, test.status, response.Code)
		require.Empty(t, response.Result().Cookies())
		if test.status == 303 {
			require.Equal(t, "/cib/live/profile/edit", response.Header().Get("Location"))
		}
	}
}
