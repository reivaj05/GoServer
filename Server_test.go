package GoServer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Headers map[string]string

type ServerTestSuite struct {
	suite.Suite
	serverNotFoundServer     *httptest.Server
	customGetEndpointServer  *httptest.Server
	customPostEndpointServer *httptest.Server
	assert                   *assert.Assertions
	client                   *http.Client
}

func (suite *ServerTestSuite) SetupSuite() {
	suite.assert = assert.New(suite.T())

	suite.serverNotFoundServer = httptest.NewServer(handlerWrapper(notFoundHandler))
	suite.customGetEndpointServer = httptest.NewServer(handlerWrapper(suite.customGetEndpointHandler))
	suite.customPostEndpointServer = httptest.NewServer(handlerWrapper(suite.customPostEndpointHandler))

	suite.client = &http.Client{}
}

func (suite *ServerTestSuite) TestStartWrongPort() {
	err := Start("", nil)
	suite.assert.NotNil(err)
}

func (suite *ServerTestSuite) TestStart() {
	go Start("9090", []*Endpoint{
		&Endpoint{
			Method:  "GET",
			Path:    "/",
			Handler: suite.customGetEndpointHandler,
		}})
}

func (suite *ServerTestSuite) TestStartWrongEndpoint() {
	err := Start("", []*Endpoint{
		&Endpoint{
			Method: "WRONG",
		}})
	suite.assert.NotNil(err)
}

func (suite *ServerTestSuite) TestInitServerSuccessfully() {
	err := initServer("8080")
	suite.assert.Nil(err)
}

func (suite *ServerTestSuite) TestInitServerWithoutPort() {
	err := initServer("")
	suite.assert.NotNil(err)
	suite.assert.Equal(missingPortError, err.Error())
}

func (suite *ServerTestSuite) TestInitServerWithWrongError() {
	err := initServer("NAN")
	suite.assert.NotNil(err)
	suite.assert.Equal(nanPortError, err.Error())
}

func (suite *ServerTestSuite) TestCreateRouterWithGoodEndpoints() {
	endpoint := &Endpoint{
		Method:  "GET",
		Path:    "/",
		Handler: suite.customGetEndpointHandler,
	}
	router, err := createRouter([]*Endpoint{endpoint})
	suite.assert.Nil(err)
	suite.assert.NotNil(router)
}

func (suite *ServerTestSuite) TestCreateRouterWithEndpointsAndWrongMethod() {
	endpoint := &Endpoint{
		Method: "WRONG",
	}
	router, err := createRouter([]*Endpoint{endpoint})
	suite.assert.NotNil(err)
	suite.assert.Nil(router)
}

func (suite *ServerTestSuite) TestCreateRouterWithEndpointsAndEmptyPath() {
	endpoint := &Endpoint{
		Method: "GET",
		Path:   "",
	}
	router, err := createRouter([]*Endpoint{endpoint})
	suite.assert.NotNil(err)
	suite.assert.Nil(router)
}

func (suite *ServerTestSuite) TestCreateRouterWithEndpointsAndNilHandler() {
	endpoint := &Endpoint{
		Method:  "GET",
		Path:    "/",
		Handler: nil,
	}
	router, err := createRouter([]*Endpoint{endpoint})
	suite.assert.NotNil(err)
	suite.assert.Nil(router)
}

func (suite *ServerTestSuite) TestNotFoundHandler() {
	reponse, status := suite.performRequest("GET", suite.serverNotFoundServer.URL, nil, nil)
	suite.assert.Equal(ResourceNotFound, reponse)
	suite.assert.Equal(http.StatusNotFound, status)
}

func (suite *ServerTestSuite) TestCustomGetEndpointWrongHeaders() {
	reponse, status := suite.performRequest("GET", suite.customGetEndpointServer.URL, nil, nil)
	msg := `{"error": "Missing header application/json"}`
	suite.assert.Equal(msg, reponse)
	suite.assert.Equal(http.StatusBadRequest, status)
}

func (suite *ServerTestSuite) TestCustomGetEndpointWithHeaders() {
	reponse, status := suite.performRequest("GET", suite.customGetEndpointServer.URL, nil,
		map[string]string{
			"Content-Type": "application/json",
			"Origin":       "localhost"})
	suite.assert.Equal("{}", reponse)
	suite.assert.Equal(http.StatusOK, status)
}

func (suite *ServerTestSuite) TestCustomPostEndpointWithHeaders() {
	body := bytes.NewBuffer([]byte(`{"request": "request"}`))
	response, status := suite.performRequest("POST", suite.customPostEndpointServer.URL, body, map[string]string{
		"Content-Type": "application/json",
	})
	suite.assert.Equal(`{"response": "response"}`, response)
	suite.assert.Equal(http.StatusOK, status)
}

func (suite *ServerTestSuite) performRequest(method, url string, body *bytes.Buffer, headers map[string]string) (string, int) {
	var request *http.Request
	if body == nil {
		request, _ = http.NewRequest(method, url, nil)
	} else {
		request, _ = http.NewRequest(method, url, body)
	}
	suite.addHeadersToRequest(request, headers)
	response, _ := suite.client.Do(request)
	res, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return string(res), response.StatusCode
}

func (suite *ServerTestSuite) addHeadersToRequest(
	request *http.Request, headers map[string]string) {

	for key, value := range headers {
		request.Header.Add(key, value)
	}
}

func (suite *ServerTestSuite) customGetEndpointHandler(w http.ResponseWriter, r *http.Request) {
	err := AreRequestHeadersWrong(r, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		SendResponseWithStatus(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	SendResponseWithStatus(w, "{}", http.StatusOK)

}

func (suite *ServerTestSuite) customPostEndpointHandler(w http.ResponseWriter, r *http.Request) {
	err := AreRequestHeadersWrong(r, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		SendResponseWithStatus(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	body, err := ReadBodyRequest(r)
	if err != nil {
		SendResponseWithStatus(w, `{"error": "Error reading request body"}`, http.StatusInternalServerError)
		return
	}
	suite.assert.Equal(`{"request": "request"}`, body)
	SendResponseWithStatus(w, `{"response": "response"}`, http.StatusOK)
}

func (suite *ServerTestSuite) TestGetQueryParams() {
	mockRequest, _ := http.NewRequest("GET", "http://localhost/?id=1&db=10", nil)
	params := GetQueryParams(mockRequest)
	suite.assert.NotNil(params)
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
