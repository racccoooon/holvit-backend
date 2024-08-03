package utils

import (
	"net/http"
	"strings"
)

func GetRequestIp(r *http.Request) string {
	ip := r.RemoteAddr

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}

	realIP := r.Header.Get("X-Real-Ip")
	if realIP != "" {
		ip = realIP
	}

	return ip
}
