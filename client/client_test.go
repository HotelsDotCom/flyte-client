/*
Copyright (C) 2018 Expedia Group.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
	"bytes"
	"github.com/stretchr/testify/require"
	"github.com/HotelsDotCom/go-logger"
)

func Test_NewClient_ShouldRetryOnErrorGettingFlyteApiLinks(t *testing.T) {
	// given the mock flyte-api will first return an error response getting api links...then after retrying will return the expected response
	apiLinksFailCount := 1
	handler := func(w http.ResponseWriter, r *http.Request) {
		if apiLinksFailCount > 0 {
			apiLinksFailCount -= apiLinksFailCount
			w.Write(bytes.NewBufferString(flyteApiErrorResponse).Bytes())
			return
		}
		w.Write(bytes.NewBufferString(flyteApiLinksResponse).Bytes())
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// and code to record the log message/s
	logMsg := ""
	loggerFn := logger.Errorf
	logger.Errorf = func(msg string, args ...interface{}) { logMsg = fmt.Sprintf(msg, args...) }
	defer func() { logger.Errorf = loggerFn }()

	baseUrl, _ := url.Parse(server.URL)

	// when
	client := NewClient(baseUrl, 10 * time.Second)

	// then a log error message will have been recorded...
	assert.Contains(t, logMsg, "cannot get api links:")
	// ...but the links are available after the retry
	healthCheckURL, _ := client.GetFlyteHealthCheckURL()
	assert.Equal(t, "http://example.com/v1/health", healthCheckURL.String())
}

func Test_GetFlyteHealthCheckURL_ShouldSelectFlyteHealthCheckUrlFromFlyteApiLinks(t *testing.T) {
	// given
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.NewBufferString(flyteApiLinksResponse).Bytes())
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	baseUrl, _ := url.Parse(server.URL)
	client := NewClient(baseUrl, 10 * time.Second)

	// when
	healthCheckURL, err := client.GetFlyteHealthCheckURL()

	// then
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/v1/health", healthCheckURL.String())
}

func Test_GetFlyteHealthCheckURL_ShouldReturnErrorWhenItCannotGetHealthCheckURLFromFlyteApiLinks(t *testing.T) {
	// given
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.NewBufferString(flyteApiNoLinksResponse).Bytes())
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	baseUrl, _ := url.Parse(server.URL)
	client := NewClient(baseUrl, 10 * time.Second)

	// when
	_, err := client.GetFlyteHealthCheckURL()

	// then
	assert.Equal(t, "could not find link with rel \"info/health\" in []", err.Error())
}

func Test_TakeAction_ShouldReturnSpecificErrorTypeAndMessageWhenResourceIsNotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	u, _ := url.Parse(server.URL + "/take/action/url")

	c := &client{takeActionURL: u, httpClient: &http.Client{
		Timeout: 5 * time.Second,
	}}

	_, e := c.TakeAction()

	if assert.Error(t, e) {
		assert.IsType(t, NotFoundError{}, e)
		assert.Equal(t, fmt.Sprintf("Resource not found at %s/take/action/url", server.URL), e.Error())
	}
}

var flyteApiLinksResponse = `{
	"links": [
		{
		"href": "http://example.com/v1",
		"rel": "self"
		},
		{
		"href": "http://example.com/",
		"rel": "up"
		},
		{
		"href": "http://example.com/swagger#!/info/v1",
		"rel": "help"
		},
		{
		"href": "http://example.com/v1/health",
		"rel": "http://example.com/swagger#!/info/health"
		},
		{
		"href": "http://example.com/v1/packs",
		"rel": "http://example.com/swagger#!/pack/listPacks"
		},
		{
		"href": "http://example.com/v1/flows",
		"rel": "http://example.com/swagger#!/flow/listFlows"
		},
		{
		"href": "http://example.com/v1/datastore",
		"rel": "http://example.com/swagger#!/datastore/listDataItems"
		},
		{
		"href": "http://example.com/v1/audit/flows",
		"rel": "http://example.com/swagger#!/audit/findFlows"
		},
		{
		"href": "http://example.com/v1/swagger",
		"rel": "http://example.com/swagger"
		}
	]
}`

var flyteApiNoLinksResponse = `{
	"links": []
}`

var flyteApiErrorResponse = `{
	"error!" 
}`