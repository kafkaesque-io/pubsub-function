package route

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kafkaesque-io/pubsub-function/src/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Route - HTTP Route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
	AuthFunc    mux.MiddlewareFunc
}

// Routes list of HTTP Routes
type Routes []Route

// PrometheusRoute definition
var PrometheusRoute = Routes{
	Route{
		"Prometeus metrics",
		http.MethodGet,
		"/metrics",
		promhttp.Handler().ServeHTTP,
		middleware.NoAuth,
	},
}

// ReceiverRoutes definition
var ReceiverRoutes = Routes{
	Route{
		"status",
		"GET",
		"/status",
		StatusPage,
		middleware.AuthHeaderRequired,
	},
	Route{
		"Receive",
		"POST",
		"/v1/firehose",
		ReceiveHandler,
		middleware.NoAuth,
	},
}

// RestRoutes definition
var RestRoutes = Routes{
	Route{
		"Get a function",
		"GET",
		"/v2/function/{tenant}/{function}",
		GetFunctionHandler,
		middleware.AuthVerifyJWT,
	},
	Route{
		"Create a function",
		"POST",
		"/v2/function/{tenant}/{function}",
		UpdateFunctionHandler,
		middleware.AuthVerifyJWT,
	},
	Route{
		"Delete a function",
		"DELETE",
		"/v2/function/{tenant}/{function}",
		DeleteFunctionHandler,
		middleware.AuthVerifyJWT,
	},
	Route{
		"Trigger a function",
		"PUT",
		"/v2/function",
		TriggerFunctionHandler,
		middleware.AuthVerifyJWT,
	},
}
