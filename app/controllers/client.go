package controllers

import (
	"fmt"
	"os"

	"github.com/revel/revel"
)

type Client struct {
	*revel.Controller
}

/*

Card Type	Card Number	Expiry Date	Security Code (CVC/CVV/CID)
American Express	3714 4963 5398 431	03/2030	7373
*/

func (c Client) setPageAndData(data map[string]interface{}) revel.Result {
	c.ViewArgs = data
	return c.RenderTemplate("index.html")
}

func (c Client) IndexHandler() revel.Result {
	fmt.Println("Loading main page")
	var data map[string]interface{}
	data = map[string]interface{}{
		"page": "main",
	}
	return c.setPageAndData(data)
}

func (c Client) PreviewHandler() revel.Result {
	fmt.Println("Loading preview page")
	var data map[string]interface{}
	data = map[string]interface{}{
		"page": "preview",
		"type": c.Params.Route.Get("type"),
	}
	return c.setPageAndData(data)
}

func (c Client) CheckoutHandler() revel.Result {
	fmt.Println("Loading payment page")
	var data map[string]interface{}
	data = map[string]interface{}{
		"page":      "payment",
		"type":      c.Params.Route.Get("type"),
		"clientKey": os.Getenv("CLIENT_KEY"),
	}
	return c.setPageAndData(data)
}

func (c Client) ResultHandler() revel.Result {
	fmt.Println("Loading result page")
	status := c.Params.Route.Get("status")
	refusalReason := c.Params.Query.Get("reason")
	fmt.Println(status)
	fmt.Println(refusalReason)

	var msg, img string
	switch status {
	case "pending":
		msg = "Your order has been received! Payment completion pending."
		img = "success"
		break
	case "failed":
		msg = "The payment was refused. Please try a different payment method or card."
		img = "failed"
		break
	case "error":
		msg = fmt.Sprintf("Error! Reason: %s", refusalReason)
		img = "failed"
		break
	default:
		msg = "Your order has been successfully placed."
		img = "success"
	}
	var data map[string]interface{}
	data = map[string]interface{}{
		"page":   "result",
		"status": status,
		"msg":    msg,
		"img":    img,
	}
	return c.setPageAndData(data)
}
