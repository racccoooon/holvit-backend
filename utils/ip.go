package utils

import (
	"github.com/jackc/pgtype"
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

func InetFromString(address string) pgtype.Inet {
	inet := pgtype.Inet{}

	ipStr := address

	lastClosingBracketPosition := strings.LastIndex(ipStr, "]")
	isIpv6 := lastClosingBracketPosition != -1
	if isIpv6 {
		if colonIndex := strings.LastIndex(ipStr, ":"); colonIndex > lastClosingBracketPosition {
			ipStr = ipStr[:colonIndex]
		}

		if strings.HasPrefix(ipStr, "[") && strings.HasSuffix(ipStr, "]") {
			ipStr = ipStr[1 : len(ipStr)-1] // Remove square brackets for IPv6
		}
	} else {
		if colonIndex := strings.LastIndex(ipStr, ":"); colonIndex > -1 {
			ipStr = ipStr[:colonIndex]
		}
	}

	err := inet.Set(ipStr)
	if err != nil {
		panic(err)
	}

	return inet
}
