package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Kong/go-pdk"
	"github.com/Kong/go-pdk/server"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
)

var publickey *rsa.PublicKey
var ctx = context.Background()
var redisClient *redis.Client

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
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New() interface{} {
	return &Config{}
}

func (conf Config) Access(kong *pdk.PDK) {
	if publickey == nil {
		kong.Log.Info("Loading public key")
		getKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(conf.Publickey))
		if err != nil {
			kong.Log.Err(err.Error())
			panic(err)
		}
		publickey = getKey
	}

	if redisClient == nil {
		kong.Log.Info(fmt.Sprintf("Connect REDIS(%s) using DB(%d)", conf.Redis.Dsn, conf.Redis.Db))
		redisClient = redis.NewClient(&redis.Options{
			Addr:        conf.Redis.Dsn,
			DB:          conf.Redis.Db,
			Password:    "",
			DialTimeout: 300 * time.Millisecond,
			ReadTimeout: 200 * time.Millisecond,
		})
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

	if err != nil || !authToken.Valid {
		returnError(kong, 401, "E1000")
		return
	}
	for v := range conf.JWT.Claims {
		if authClaims[conf.JWT.Claims[v].Name] == nil {
			returnError(kong, 401, "E1001")
			return
		}
	}

	for _, claim := range conf.JWT.Claims {
		if claim.Redis != "" {
			blacklistVal, err := redisClient.Get(ctx, fmt.Sprintf(claim.Redis, authClaims[claim.Name])).Result()
			if err != nil {
				kong.Log.Err(fmt.Sprintf("Redis ERR: %s", err.Error()))
			}
			if blacklistVal != "" {
				returnError(kong, 403, "E2000")
				return
			}
		}
	}

	kong.ServiceRequest.ClearHeader("Authorization")
	kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-role", conf.JWT.Prefix), "user")
	kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-jwt", conf.JWT.Prefix), reqToken)
	for _, claim := range conf.JWT.Claims {
		kong.ServiceRequest.SetHeader(fmt.Sprintf("x-%s-%s", conf.JWT.Prefix, claim.Name), fmt.Sprintf("%v", authClaims[claim.Name]))
	}
}

func returnError(kong *pdk.PDK, status int, code string) {
	errHeaders := make(map[string][]string)
	errHeaders["Content-Type"] = append(errHeaders["Content-Type"], "application/json")

	errBody, _ := json.Marshal(ErrResponse{
		Code:    code,
		Message: "Unauthorized",
	})
	kong.Response.Exit(status, string(errBody), errHeaders)
}

func main() {
	server.StartServer(New, "0.1", 1000)
}
