package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Kong/go-pdk"
	"github.com/Kong/go-pdk/server"
	"github.com/golang-jwt/jwt"
)

type RedisConfig struct {
	Dsn string
	Db  int
}
type ClaimConfig struct {
	Name  string
	Redis string
}
type JWTConfig struct {
	Prefix string
	Claims []ClaimConfig
}
type Config struct {
	Publickey string
	Redis     RedisConfig
	JWT       JWTConfig
}

type ErrResponse struct {
	Code    int    `json:"statusCode"`
	Message string `json:"msg"`
}

func New() interface{} {
	return &Config{}
}

func (conf Config) Access(kong *pdk.PDK) {
	publickey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(conf.Publickey))
	if err != nil {
		kong.Log.Err(err.Error())
		panic(err)
	}

	for _, claim := range conf.JWT.Claims {
		kong.ServiceRequest.ClearHeader(fmt.Sprintf("x-%s-%s", conf.JWT.Prefix, claim.Name))
	}
	kong.ServiceRequest.ClearHeader(fmt.Sprintf("x-%s-jwt", conf.JWT.Prefix))

	authHeader, err := kong.Request.GetHeader("Authorization")
	if err != nil {
		kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-role", conf.JWT.Prefix), "guest")
		return
	}

	reqToken := strings.Replace(authHeader, "Bearer ", "", 1)
	authClaims := jwt.MapClaims{}
	authToken, err := jwt.ParseWithClaims(reqToken, &authClaims, func(jwtToken *jwt.Token) (interface{}, error) {
		if _, ok := jwtToken.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", jwtToken.Header["alg"])
		}
		return publickey, nil
	})

	if err != nil && err.Error() == "Token is expired" {
		returnError(kong, 401, 42002, "TOKEN.EXPIRED")
		return
	}

	if err != nil || !authToken.Valid {
		returnError(kong, 401, 42001, "TOKEN.INVALID")
		return
	}

	for v := range conf.JWT.Claims {
		if authClaims[conf.JWT.Claims[v].Name] == nil {
			returnError(kong, 401, 44001, "TOKEN.CLAIM.MISSING")
			return
		}
	}

	kong.ServiceRequest.ClearHeader("Authorization")
	kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-role", conf.JWT.Prefix), "user")
	kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-jwt", conf.JWT.Prefix), reqToken)
	for _, claim := range conf.JWT.Claims {
		kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-%s", conf.JWT.Prefix, claim.Name), fmt.Sprintf("%v", authClaims[claim.Name]))
	}
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
