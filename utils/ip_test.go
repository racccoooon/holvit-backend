package utils

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func Test_GetRequestIp_NoHeaders(t *testing.T) {
	// arrange
	expected := "127.0.0.1"
	r := http.Request{
		RemoteAddr: expected,
		Header:     http.Header{},
	}

	// act
	ip := GetRequestIp(&r)

	// assert
	assert.Equal(t, expected, ip)
}

func Test_GetRequestIp_Forwarded(t *testing.T) {
	// arrange
	expected := "127.0.0.1"
	r := http.Request{
		RemoteAddr: "1.1.1.1",
		Header: http.Header{
			"X-Forwarded-For": []string{expected},
		},
	}

	// act
	ip := GetRequestIp(&r)

	// assert
	assert.Equal(t, expected, ip)
}

func Test_GetRequestIp_XRealIp(t *testing.T) {
	// arrange
	expected := "127.0.0.1"
	r := http.Request{
		RemoteAddr: "1.1.1.1",
		Header: http.Header{
			"X-Real-Ip": []string{expected},
		},
	}

	// act
	ip := GetRequestIp(&r)

	// assert
	assert.Equal(t, expected, ip)
}

func Test_GetRequestIp_ForwarededAndXRealIp(t *testing.T) {
	// arrange
	expected := "127.0.0.1"
	r := http.Request{
		RemoteAddr: "1.1.1.1",
		Header: http.Header{
			"x-Forwarded-For": []string{"2.2.2.2"},
			"X-Real-Ip":       []string{expected},
		},
	}

	// act
	ip := GetRequestIp(&r)

	// assert
	assert.Equal(t, expected, ip)
}
