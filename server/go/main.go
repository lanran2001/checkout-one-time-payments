package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/setupintent"
)

type SetupIntentResponse struct {
	ClientSecret string `json:"clientSecret"`
	CustomerId   string `json:"customerId"`
}

type ChargeResponse struct {
	Success       bool                  `json:"success"`
	PaymentIntent *stripe.PaymentIntent `json:"paymentIntent"`
}

func init() {
	stripe.Key = "sk_test_51PGYMiRoUfb6BI4plZXZUQMlXEs1VC9UXWXOhoh20oLIcOMoDxqD4gdzCYFYjLzc7BUPjk3qDLpVDUqSBj8FmzaQ008BDyK1QF"
}

func handleCreateSetupIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 创建customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String("555@qq.com"),
	}

	newCustomer, err := customer.New(customerParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("newCustomer.ID:", newCustomer.ID)

	// 创建SetupIntent
	params := &stripe.SetupIntentParams{
		Customer: stripe.String(newCustomer.ID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
	}

	intent, err := setupintent.New(params)
	if err != nil {
		log.Printf("Error creating SetupIntent: %v", err)
		json.NewEncoder(w).Encode(SetupIntentResponse{})
		return
	}

	// 4. 返回 ClientSecret
	w.Header().Set("Content-Type", "application/json")
	fmt.Println("intent.ClientSecret:", intent.ClientSecret)
	json.NewEncoder(w).Encode(SetupIntentResponse{
		ClientSecret: intent.ClientSecret,
		CustomerId:   newCustomer.ID,
	})
}

// SavePaymentMethodRequest 保存支付方式的请求
type SavePaymentMethodRequest struct {
	PaymentMethodId string `json:"paymentMethodId"`
	SetupIntentId   string `json:"setupIntentId"`
	CustomerId      string `json:"customerId"`
}
type SavePaymentMethodResponse struct {
	CustomerId string `json:"customerId"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

func handleSavePaymentMethod(w http.ResponseWriter, r *http.Request) {
	var req SavePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("customerId:", req.CustomerId)
	fmt.Println("paymentMethodId:", req.PaymentMethodId)
	if req.CustomerId == "" || req.PaymentMethodId == "" {
		http.Error(w, "err: customerId or paymentMethodId is empty", http.StatusBadRequest)
		return
	}

	// 1. 获取或创建 customer
	// 实际应用中，你应该根据用户ID获取对应的customer
	pmParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(req.CustomerId),
	}
	pm, err := paymentmethod.Attach(req.PaymentMethodId, pmParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. 更新 Customer 的默认支付方式
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}

	cus, err := customer.Update(req.CustomerId, customerParams)
	if err != nil {
		// 如果更新失败，尝试解除已附加的支付方式
		_, detachErr := paymentmethod.Detach(req.PaymentMethodId, nil)
		if detachErr != nil {
			// 记录清理错误，但不返回给用户
			fmt.Printf("Error detaching payment method after failed update: %v\n", detachErr)
		}
		return
	}

	// 3. 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SavePaymentMethodResponse{
		CustomerId: cus.ID,
		Status:     "success",
	})
}

// ChargeRequest 扣费请求
type ChargeRequest struct {
	CustomerId      string `json:"customerId"`
	PaymentMethodId string `json:"paymentMethodId"`
}

func handleCharge(w http.ResponseWriter, r *http.Request) {
	var req ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 从数据库或会话中获取customerId
	customerId := req.CustomerId // 替换为实际的customer ID

	// 获取customer的默认payment method
	cus, err := customer.Get(customerId, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("cus:%+v\n", cus)
	//fmt.Println("cus.DefaultSource.ID:", cus.DefaultSource.ID)

	// 创建PaymentIntent并直接确认
	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(200),
		Currency:      stripe.String(string(stripe.CurrencySGD)),
		Customer:      stripe.String(customerId),
		PaymentMethod: stripe.String(req.PaymentMethodId),
		Confirm:       stripe.Bool(true),
		OffSession:    stripe.Bool(true), // 表明这是后台扣费
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回支付结果
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          pi.Status,
		"paymentIntentId": pi.ID,
		"amount":          pi.Amount,
	})
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
func main() {
	// 添加静态文件服务
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// 原有的API路由
	http.HandleFunc("/create-setup-intent", handleCreateSetupIntent)
	http.HandleFunc("/save-payment-method", handleSavePaymentMethod)
	http.HandleFunc("/charge", handleCharge)

	log.Printf("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
