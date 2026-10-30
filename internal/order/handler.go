package order

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nilesh/order-service/internal/order/rpc"
)

type Handler struct {
	DB         *sql.DB
	ProductURL string
	ServiceKey string
	HTTP       *http.Client
	RPC        *rpc.Clients
}

type createOrderInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var input createOrderInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(400, "invalid request body")
	}
	if input.ProductID == "" || input.Quantity < 1 || input.Quantity > 100 {
		return fiber.NewError(400, "product_id and quantity between 1 and 100 are required")
	}
	user, err := h.RPC.UserDetails(c.Context(), c.Locals("userID").(string))
	if err != nil {
		return fiber.NewError(503, "user service unavailable")
	}
	if _, err = h.RPC.ProductDetails(c.Context(), input.ProductID); err != nil {
		return fiber.NewError(404, "product not found")
	}
	reserveBody, _ := json.Marshal(fiber.Map{"quantity": input.Quantity})
	request, err := http.NewRequestWithContext(c.Context(), http.MethodPost, fmt.Sprintf("%s/internal/v1/products/%s/reserve", h.ProductURL, input.ProductID), bytes.NewReader(reserveBody))
	if err != nil {
		return fiber.NewError(500, "could not prepare inventory request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Service-Key", h.ServiceKey)
	response, err := h.HTTP.Do(request)
	if err != nil {
		return fiber.NewError(503, "product service unavailable")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return fiber.NewError(409, "inventory could not be reserved")
	}
	var reservation struct {
		UnitPrice int64 `json:"unit_price"`
	}
	decodeErr := json.NewDecoder(response.Body).Decode(&reservation)
	response.Body.Close()
	if decodeErr != nil || reservation.UnitPrice <= 0 {
		h.releaseInventory(input.ProductID, input.Quantity)
		return fiber.NewError(502, "product service returned an invalid reservation")
	}
	tx, err := h.DB.BeginTx(c.Context(), nil)
	if err != nil {
		h.releaseInventory(input.ProductID, input.Quantity)
		return fiber.NewError(500, "could not create order")
	}
	defer tx.Rollback()
	committed := false
	defer func() {
		if !committed {
			h.releaseInventory(input.ProductID, input.Quantity)
		}
	}()
	unitPrice := reservation.UnitPrice
	total := unitPrice * int64(input.Quantity)
	var orderID string
	err = tx.QueryRowContext(c.Context(), `INSERT INTO orders(user_id,status,total_amount,currency) VALUES($1,'placed',$2,'INR') RETURNING id`, c.Locals("userID"), total).Scan(&orderID)
	if err != nil {
		return fiber.NewError(500, "could not create order")
	}
	_, err = tx.ExecContext(c.Context(), `INSERT INTO order_items(order_id,product_id,quantity,unit_price) VALUES($1,$2,$3,$4)`, orderID, input.ProductID, input.Quantity, unitPrice)
	if err != nil {
		return fiber.NewError(500, "could not save order items")
	}
	payload, _ := json.Marshal(fiber.Map{
		"event_id":     fmt.Sprintf("%s-%d", orderID, time.Now().UnixNano()),
		"order_id":     orderID,
		"user_id":      c.Locals("userID"),
		"email":        user.Email,
		"total_amount": total,
		"currency":     "INR",
	})
	_, err = tx.ExecContext(c.Context(), `INSERT INTO outbox_events(event_type,payload) VALUES('order.placed',$1)`, payload)
	if err != nil {
		return fiber.NewError(500, "could not record order event")
	}
	if err = tx.Commit(); err != nil {
		return fiber.NewError(500, "could not finalize order")
	}
	committed = true
	return c.Status(201).JSON(fiber.Map{"id": orderID, "status": "placed", "total_amount": total, "currency": "INR"})
}

func (h *Handler) List(c *fiber.Ctx) error {
	rows, err := h.DB.QueryContext(c.Context(), `SELECT id,status,total_amount,currency,created_at FROM orders WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100`, c.Locals("userID"))
	if err != nil {
		return fiber.NewError(500, "could not list orders")
	}
	defer rows.Close()
	result := make([]fiber.Map, 0)
	for rows.Next() {
		var id, status, currency string
		var amount int64
		var created time.Time
		if err := rows.Scan(&id, &status, &amount, &currency, &created); err != nil {
			return fiber.NewError(500, "could not read orders")
		}
		result = append(result, fiber.Map{"id": id, "status": status, "total_amount": amount, "currency": currency, "created_at": created})
	}
	if err := rows.Err(); err != nil {
		return fiber.NewError(500, "could not read orders")
	}
	return c.JSON(result)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	var id, status, currency string
	var amount int64
	var created time.Time
	err := h.DB.QueryRowContext(c.Context(), `SELECT id,status,total_amount,currency,created_at FROM orders WHERE id=$1 AND user_id=$2`, c.Params("id"), c.Locals("userID")).Scan(&id, &status, &amount, &currency, &created)
	if err == sql.ErrNoRows {
		return fiber.NewError(404, "order not found")
	}
	if err != nil {
		return fiber.NewError(500, "could not load order")
	}
	return c.JSON(fiber.Map{"id": id, "status": status, "total_amount": amount, "currency": currency, "created_at": created})
}

func (h *Handler) releaseInventory(productID string, quantity int) {
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/internal/v1/products/%s/release?quantity=%d", h.ProductURL, productID, quantity), nil)
	if err != nil {
		return
	}
	request.Header.Set("X-Service-Key", h.ServiceKey)
	response, err := h.HTTP.Do(request)
	if err == nil {
		response.Body.Close()
	}
}
