package attributes

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

const (
	KeyHttpReqProtocol  = "http.request.protocol"
	KeyHttpRespDuration = "http.request.duration"
)

// HTTPRoute returns an attribute KeyValue conforming to the "http.route"
// semantic conventions. It represents the matched route, that is, the path
// template in the format used by the respective server framework.
// Examples: '/users/:userID?', '{controller}/{action}/{id?}'
// Note: MUST NOT be populated when this is not supported by the HTTP
// server framework as the route attribute should have low-cardinality and
// the URI path can NOT substitute it.
func HttpRoute(value string) attribute.KeyValue {
	return semconv.HTTPRoute(value)
}

// HTTPRequestProtocol returns an attribute KeyValue conforming to the
// "http.request.protocol" semantic conventions. It represents the version of
// the HTTP protocol used by the request, such as "1.1", "2", or "3".
func HttpReqProtocol(value string) attribute.KeyValue {
	return String(KeyHttpReqProtocol, value)
}

// HTTPRequestMethodKey is the attribute Key conforming to the
// "http.request.method" semantic conventions. It represents the hTTP
// request method.
func HttpReqMethod(value string) attribute.KeyValue {
	return semconv.HTTPRequestMethodKey.String(value)
}

// HTTPRequestSize returns an attribute KeyValue conforming to the
// "http.request.size" semantic conventions. It represents the total size of
// the request in bytes. This should be the total number of bytes sent over the
// wire, including the request line (HTTP/1.1), framing (HTTP/2 and HTTP/3),
// headers, and request body if any.
func HttpReqSizeBytes(value int) attribute.KeyValue {
	return semconv.HTTPRequestSize(value)
}

// HTTPResponseStatusCode returns an attribute KeyValue conforming to the
// "http.response.status_code" semantic conventions. It represents the [HTTP
// response status code](https://tools.ietf.org/html/rfc7231#section-6).
func HttpRespStatusCode(value int) attribute.KeyValue {
	return semconv.HTTPResponseStatusCode(value)
}

// HTTPResponseSize returns an attribute KeyValue conforming to the
// "http.response.size" semantic conventions. It represents the total size of
// the response in bytes. This should be the total number of bytes sent over
// the wire, including the status line (HTTP/1.1), framing (HTTP/2 and HTTP/3),
// headers, and response body and trailers if any.
func HttpRespSizeBytes(value int) attribute.KeyValue {
	return semconv.HTTPResponseSize(value)
}

// HttpRespDurationMilliseconds returns an attribute KeyValue conforming to
// the "http.request.duration" semantic convention. It represents the duration
// of the HTTP request in milliseconds, from receiving the first byte of the
// request headers to sending the last byte of the response payload.
func HttpRespDurationMilliseconds(value int) attribute.KeyValue {
	return Int(KeyHttpRespDuration, value)
}
