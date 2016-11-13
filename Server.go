package GoServer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	missingPortError = "port can't be empty"
	nanPortError     = "port must be a number"
	notFoundMsg      = `{"error": "Not found"}`
)

var allowedMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"OPTIONS": true,
}

// Endpoint ...
type Endpoint struct {
	method  string
	path    string
	handler func(w http.ResponseWriter, r *http.Request)
}

// Start Spins up a new HTTP server to handle requests
func Start(port string, endpoints []*Endpoint) error {
	router, err := createRouter(endpoints)
	if err != nil {
		return err
	}
	if err := initServer(port); err != nil {
		return err
	}
	return http.ListenAndServe(":"+port, router)
}

func initServer(port string) error {
	return checkForErrors(port)
}

func checkForErrors(port string) error {
	if port == "" {
		return fmt.Errorf(missingPortError)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf(nanPortError)
	}
	return nil
}

func createRouter(endpoints []*Endpoint) (*mux.Router, error) {
	router := mux.NewRouter().StrictSlash(true)
	router.NotFoundHandler = handlerWrapper(notFoundHandler)
	if err := addEndpointsToRouter(router, endpoints); err != nil {
		return nil, err
	}
	// router.Methods("GET").Path("/").Handler(handlerWrapper(usageHandler))
	// router.Methods("POST").Path("/").Handler(handlerWrapper(augmentationHandler))
	return router, nil
}

func addEndpointsToRouter(router *mux.Router, endpoints []*Endpoint) error {
	for _, endpoint := range endpoints {
		if isEndpointOK(endpoint) {
			method, path, handler := endpoint.method, endpoint.path, endpoint.handler
			router.Methods(method).Path(path).Handler(handlerWrapper(handler))
		} else {
			return fmt.Errorf("Wrong format for endpoint %v", endpoint)
		}
	}
	return nil
}

func isEndpointOK(endpoint *Endpoint) bool {
	if !isHTTPMethodAllowed(endpoint.method) {
		return false
	}
	if endpoint.path == "" || endpoint.handler == nil {
		return false
	}
	return true
}

func isHTTPMethodAllowed(method string) bool {
	if _, ok := allowedMethods[method]; !ok {
		return false
	}
	return true
}

func handlerWrapper(handler http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		http.HandlerFunc(handler).ServeHTTP(w, r)
	})
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	SendResponseWithStatus(w, notFoundMsg, http.StatusNotFound)
}

func AreRequestHeadersWrong(request *http.Request, headers map[string]string) error {
	for key, value := range headers {
		if request.Header.Get(key) != value {
			return fmt.Errorf("Missing header %s", value)
		}
	}
	return nil
}

func ReadBodyRequest(request *http.Request) (string, error) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return "", err
	}
	err = request.Body.Close()
	return string(body), err
}

func SendResponseWithStatus(
	w http.ResponseWriter, response string, status int) error {

	w.WriteHeader(status)
	_, err := fmt.Fprintf(w, response)
	return err
}
