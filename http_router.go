package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Route is struct for http route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is slice of Route struct
type Routes []Route

// NewRouter is constructor of Route struct
func NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {

		var handler http.Handler
		handler = route.HandlerFunc

		// logger middle ware
		//handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

var routes = Routes{
	Route{
		Name:        "TaskIndex",
		Method:      "GET",
		Pattern:     "/tasks",
		HandlerFunc: TaskIndex,
	},
	Route{
		Name:        "TaskDelete",
		Method:      "DELETE",
		Pattern:     "/tasks/{taskId}",
		HandlerFunc: TaskDelete,
	},
	Route{
		Name:        "TaskUpdate",
		Method:      "PATCH",
		Pattern:     "/tasks/{taskId}",
		HandlerFunc: TaskUpdate,
	},
	Route{
		Name:        "DashBoard",
		Method:      "GET",
		Pattern:     "/dashboard",
		HandlerFunc: DashBoard,
	},
}
