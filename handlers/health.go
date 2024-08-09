package handlers

import "net/http"

func Health(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(`{"alive": true}`))
	if err != nil {
		panic(err)
	}
}
