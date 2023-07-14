package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Kong/go-pdk"
	"github.com/Kong/go-pdk/server"
)

type Config struct {
	Prefix string
	Uri    string
}

type AuthRequest struct {
	ApiKey string `json:"api_key"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

type AuthResponse struct {
	UserID string `json:"user_id"`
}

type ErrResponse struct {
	Code    int    `json:"statusCode"`
	Message string `json:"msg"`
}

func New() interface{} {
	return &Config{}
}

func (conf Config) Access(kong *pdk.PDK) {
	headerKey := fmt.Sprintf("x-%s-apikey", conf.Prefix)
	// authURI := conf.Uri

	apiKey, err1 := kong.Request.GetHeader(headerKey)
	requestPath, err2 := kong.Request.GetPath()
	requestMethod, err3 := kong.Request.GetMethod()
	if err1 != nil || err2 != nil || err3 != nil {
		returnError(kong, 401, 62001, "APIKEY.NOTFOUND")
		return
	}
	authRequestData, _ := json.Marshal(AuthRequest{
		ApiKey: apiKey,
		Path:   requestPath,
		Method: requestMethod,
	})

	authResponse, err := http.Post(conf.Uri, "application/json", bytes.NewReader(authRequestData))
	if err != nil {
		returnError(kong, 500, 62002, "APIKEY.SERVER_ERROR")
		return
	}
	// defer authResponse.Body.Close()

	var authResponseData AuthResponse
	json.NewDecoder(authResponse.Body).Decode(&authResponseData)

	kong.Log.Debug(authResponse.StatusCode)
	kong.Log.Debug(authResponseData.UserID)

	if authResponse.StatusCode == 401 {
		returnError(kong, 401, 62003, "APIKEY.INVALID")
		return
	}

	if authResponse.StatusCode == 403 {
		returnError(kong, 403, 62005, "APIKEY.OUT_OF_SCOPE")
		return
	}

	if authResponse.StatusCode == 429 {
		returnError(kong, 429, 62004, "APIKEY.LIMIT_REACHED")
		return
	}

	if authResponse.StatusCode >= 500 {
		returnError(kong, 500, 62002, "APIKEY.SERVER_ERROR")
		return
	}

	kong.ServiceRequest.ClearHeader("Authorization")
	kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-role", conf.Prefix), "api")
	kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-uid", conf.Prefix), fmt.Sprintf("%v", authResponseData.UserID))
}

func returnError(kong *pdk.PDK, status int, code int, msg string) {
	errHeaders := make(map[string][]string)
	errHeaders["Content-Type"] = append(errHeaders["Content-Type"], "application/json")

	errBody, _ := json.Marshal(ErrResponse{
		Code:    code,
		Message: msg,
	})
	kong.Response.Exit(status, string(errBody), errHeaders)
}

func main() {
	server.StartServer(New, "0.1", 1000)
}
