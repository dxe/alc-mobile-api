package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/lestrrat-go/test-mysqld"
	"os"
	"testing"
	"net/http"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/avast/retry-go/v3"
)

func TestServer(t *testing.T) {
	/* Set up MySQL */
	mysqld, err := mysqltest.NewMysqld(nil)
	if err != nil {
		t.Fatalf("failed to start mysqld: %s", err)
	}
	db, err := sqlx.Open("mysql", mysqld.Datasource("test", "", "", 0))
	if err != nil {
		t.Fatalf("failed to open MySQL connection: %s", err)
	}

	/* Set up ENV variables */
	os.Setenv("OAUTH_CLIENT_ID", "testVal")
	os.Setenv("OAUTH_CLIENT_SECRET", "testVal")
	os.Setenv("BASE_URL", "testVal")
	os.Setenv("S3_REGION", "testVal")
	os.Setenv("S3_AUTH_ID", "testVal")
	os.Setenv("S3_SECRET", "testVal")
	go main0(db)


	var response *http.Response
	retry.Do(func () error {
		resp, err := http.Get("http://localhost:8080/healthcheck")
		fmt.Println("{}, {}", resp, err)
		if err == nil {
			response = resp
		}
		return err
	})
	assert.Equal(t, response.StatusCode, 200)
}
