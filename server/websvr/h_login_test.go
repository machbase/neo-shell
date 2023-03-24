package websvr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/machbase/neo-shell/util/mock"
	"github.com/stretchr/testify/require"
)

func TestLoginRoute(t *testing.T) {

	dbMock := &mock.DatabaseServerMock{}
	dbMock.UserAuthFunc = func(user, password string) (bool, error) {
		return user == "sys" && password == "manager", nil
	}

	conf := &Config{
		Prefix: "/web/",
	}
	wsvr, err := New(dbMock, conf)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.Default()
	wsvr.Route(router)

	// success case - login
	var b = &bytes.Buffer{}
	loginReq := &LoginReq{
		LoginName: "sys",
		Password:  "manager",
	}
	expectStatus := http.StatusOK
	if err = json.NewEncoder(b).Encode(loginReq); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/web/api/login", b)
	req.Header.Set("Content-type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, expectStatus, w.Code, w.Body.String())

	// wrong password case - login
	b = &bytes.Buffer{}
	loginReq = &LoginReq{
		LoginName: "sys",
		Password:  "wrong",
	}
	expectStatus = http.StatusNotFound
	if err = json.NewEncoder(b).Encode(loginReq); err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/web/api/login", b)
	req.Header.Set("Content-type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, expectStatus, w.Code, w.Body.String())
}
