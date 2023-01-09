package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type NewResponseWriters struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *NewResponseWriters {
	return &NewResponseWriters{w, http.StatusOK}
}

var totalRequest = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests.",
	},
	[]string{"path"},
)

func (read *NewResponseWriters) WirteHeader(code int) {
	read.statusCode = code
	read.ResponseWriter.WriteHeader(code)
}

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status",
		Help: "Status of HTTP response",
	},
	[]string{"status"},
)

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_time_seconds",
	Help: "Duration of HTTP requests",
}, []string{"path"})

func prometheusMiddleware(route http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routes := mux.CurrentRoute(r)
		path, _ := routes.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		readwrite := NewResponseWriter(w)
		route.ServeHTTP(readwrite, r)
		statusCode := readwrite.statusCode
		responseStatus.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		totalRequest.WithLabelValues(path).Inc()
		timer.ObserveDuration()
	})
}

func init() {
	prometheus.Register(totalRequest)
	prometheus.Register(responseStatus)
	prometheus.Register(httpDuration)
}

func main() {
	router := mux.NewRouter()
	router.Use(prometheusMiddleware)
	router.Path("/metric").Handler(promhttp.Handler())
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("/")))
	fmt.Println("Serving requests on Port 9000")
	err := http.ListenAndServe(":9000", router)
	log.Fatal(err)
}
