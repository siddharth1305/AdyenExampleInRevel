package controllers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/adyen/adyen-go-api-library/v3/src/checkout"
	"github.com/adyen/adyen-go-api-library/v3/src/common"
	"github.com/google/uuid"
	"github.com/revel/revel"
)

type Api struct {
	*revel.Controller
}

var paymentDataStore = map[string]string{}

func (a Api) PaymentMethodsHandler() revel.Result {
	var req checkout.PaymentMethodsRequest

	if err := a.Params.BindJSON(&req); err != nil {
		a.handleError("PaymentMethodsHandler", err, nil)
		return nil
	}

	req.MerchantAccount = merchantAccount
	req.Channel = "Web"
	fmt.Println("In Payments method")
	res, httpRes, err := client.Checkout.PaymentMethods(&req)
	if err != nil {
		a.handleError("PaymentMethodsHandler", err, httpRes)
		return nil
	}
	fmt.Println("In PaymentMethodsHandler method")
	fmt.Println(res)
	return a.RenderJSON(res)
}

func (a Api) PaymentsHandler() revel.Result {
	var req checkout.PaymentRequest

	if err := a.Params.BindJSON(&req); err != nil {
		a.handleError("PaymentsHandler", err, nil)
		return nil
	}

	fmt.Println(req)

	req.MerchantAccount = merchantAccount // required
	pmType := req.PaymentMethod["type"].(string)
	req.Amount = checkout.Amount{
		Currency: findCurrency(pmType),
		Value:    1000, // value is 10€ in minor units
	}
	orderRef := uuid.Must(uuid.NewRandom())
	req.Reference = orderRef.String() // required
	req.Channel = "Web"               // required
	req.AdditionalData = map[string]interface{}{
		// required for 3ds2 native flow
		"allow3DS2": true,
	}
	req.Origin = "http://localhost:9000" // required for 3ds2 native flow
	req.ShopperIP = a.ClientIP           // required by some issuers for 3ds2

	// we pass the orderRef in return URL to get paymentData during redirects
	req.ReturnUrl = fmt.Sprintf("http://localhost:9000/api/handleShopperRedirect?orderRef=%s", orderRef) // required for 3ds2 redirect flow
	// Required for Klarna:
	if strings.Contains(pmType, "klarna") {
		req.CountryCode = "DE"
		req.ShopperReference = "12345"
		req.ShopperEmail = "youremail@email.com"
		req.ShopperLocale = "en_US"
		req.LineItems = &[]checkout.LineItem{
			{
				Quantity:           1,
				AmountExcludingTax: 331,
				TaxPercentage:      2100,
				Description:        "Sunglasses",
				Id:                 "Item 1",
				TaxAmount:          69,
				AmountIncludingTax: 400,
			},
			{
				Quantity:           1,
				AmountExcludingTax: 248,
				TaxPercentage:      2100,
				Description:        "Headphones",
				Id:                 "Item 2",
				TaxAmount:          52,
				AmountIncludingTax: 300,
			},
		}
	}

	log.Printf("Request for %s API::\n%+v\n", "Payments", req)
	res, httpRes, err := client.Checkout.Payments(&req)
	log.Printf("Response for %s API::\n%+v\n", "Payments", res)
	log.Printf("HTTP Resssaponse for %s API::\n%+v\n", "Payments", httpRes)
	if err != nil {
		log.Printf("Error in payments")
		a.handleError("PaymentsHandler", err, httpRes)
		return nil
	}
	if res.Action != nil && res.Action.PaymentData != "" {
		log.Printf("Setting payment data cache for %s\n", orderRef)
		paymentDataStore[orderRef.String()] = res.Action.PaymentData
		return a.RenderJSON(res)
	} else {
		fmt.Println(res.PspReference)
		fmt.Println(res.ResultCode.String())
		fmt.Println(res.RefusalReason)
		return a.RenderJSON(map[string]string{
			"pspReference":  res.PspReference,
			"resultCode":    res.ResultCode.String(),
			"refusalReason": res.RefusalReason,
		})
	}
}

func (a Api) PaymentDetailsHandler() revel.Result {
	var req checkout.DetailsRequest

	if err := a.Params.BindJSON(&req); err != nil {
		a.handleError("PaymentDetailsHandler", err, nil)
		return nil
	}
	log.Printf("Request for %s API::\n%+v\n", "PaymentDetails", req)
	res, httpRes, err := client.Checkout.PaymentsDetails(&req)
	log.Printf("HTTP Response for %s API::\n%+v\n", "PaymentDetails", httpRes)
	if err != nil {
		log.Println("In error")
		a.handleError("PaymentDetailsHandler", err, httpRes)
		return nil
	}
	log.Println("In flow")
	if res.Action != nil {
		log.Println("In not nil")
		return a.RenderJSON(res)
	} else {
		fmt.Println("In not nil")
		return a.RenderJSON(map[string]string{
			"pspReference":  res.PspReference,
			"resultCode":    res.ResultCode.String(),
			"refusalReason": res.RefusalReason,
		})
	}
}

type Redirect struct {
	MD             string
	PaRes          string
	Payload        string `form:"payload"`
	RedirectResult string `form:"redirectResult"`
}

func (a Api) RedirectHandler() revel.Result {
	var redirect Redirect
	log.Println("Redirect received")

	orderRef := a.Params.Route.Get("orderRef")
	paymentData := paymentDataStore[orderRef]
	log.Printf("cached paymentData for %s: %s", orderRef, paymentData)

	if err := a.Params.BindJSON(&redirect); err != nil {
		a.handleError("RedirectHandler", err, nil)
		return nil
	}

	var details map[string]interface{}
	if redirect.Payload != "" {
		details = map[string]interface{}{
			"payload": redirect.Payload,
		}
	} else if redirect.RedirectResult != "" {
		details = map[string]interface{}{
			"redirectResult": redirect.RedirectResult,
		}
	} else {
		details = map[string]interface{}{
			"MD":    redirect.MD,
			"PaRes": redirect.PaRes,
		}
	}

	req := checkout.DetailsRequest{Details: details, PaymentData: paymentData}

	log.Printf("Request for %s API::\n%+v\n", "PaymentDetails", req)
	res, httpRes, err := client.Checkout.PaymentsDetails(&req)
	log.Printf("HTTP Response for %s API::\n%+v\n", "PaymentDetails", httpRes)
	if err != nil {
		a.handleError("RedirectHandler", err, httpRes)
		return nil
	}
	log.Printf("Response for %s API::\n%+v\n", "PaymentDetails", res)

	if res.PspReference != "" {
		var redirectURL string
		// Conditionally handle different result codes for the shopper
		switch res.ResultCode {
		case common.Authorised:
			redirectURL = "/result/success"
			break
		case common.Pending:
		case common.Received:
			redirectURL = "/result/pending"
			break
		case common.Refused:
			redirectURL = "/result/failed"
			break
		default:
			{
				reason := res.RefusalReason
				if reason == "" {
					reason = res.ResultCode.String()
				}
				redirectURL = fmt.Sprintf("/result/error?reason=%s", url.QueryEscape(reason))
				break
			}
		}
		a.Redirect(
			http.StatusFound,
			redirectURL,
		)
		return nil
	}
	a.Params.BindJSON(httpRes.Status)
	return nil
}

func (a Api) handleError(method string, err error, httpRes *http.Response) {
	log.Printf("Error in %s: %s\n", method, err.Error())
	a.RenderJSON(err.Error())
}

func findCurrency(typ string) string {
	switch typ {
	case "ach":
		return "USD"
	case "wechatpayqr":
	case "alipay":
		return "CNY"
	case "dotpay":
		return "PLN"
	case "boletobancario":
	case "boletobancario_santander":
		return "BRL"
	default:
		return "EUR"
	}
	return ""
}
