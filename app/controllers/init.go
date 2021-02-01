package controllers

import (
	"os"

	"github.com/adyen/adyen-go-api-library/v3/src/adyen"
	"github.com/adyen/adyen-go-api-library/v3/src/common"
	"github.com/joho/godotenv"
	"github.com/revel/revel"
)

var (
	client          *adyen.APIClient
	merchantAccount string
)

func AddCredentials() {
	godotenv.Load("./.env")
	client = adyen.NewClient(&common.Config{
		ApiKey:      os.Getenv("API_KEY"),
		Environment: common.TestEnv,
	})

	merchantAccount = os.Getenv("MERCHANT_ACCOUNT")
}

func init() {
	revel.OnAppStart(AddCredentials)
}
