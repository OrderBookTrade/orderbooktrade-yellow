# Prediction Market API Documentation

Base URL: `http://localhost:8080`

---

## Health Check

```bash
GET /api/health
```

**Response:**
```json
{"status": "ok"}
```

---

## Market APIs

### Create Market

```bash
POST /api/market
Content-Type: application/json

{
  "question": "Will ETH be above $3000 by end of day?",
  "description": "Prediction market demo",
  "resolves_at": "2026-02-08T00:00:00Z",
  "creator_id": "admin"
}
```

**Response:**
```json
{
  "id": "mkt_abc123",
  "question": "Will ETH be above $3000 by end of day?",
  "description": "Prediction market demo",
  "status": "trading",
  "created_at": "2026-02-07T12:00:00Z",
  "resolves_at": "2026-02-08T00:00:00Z",
  "creator_id": "admin"
}
```

### List Markets

```bash
GET /api/markets
```

**Response:**
```json
[
  {
    "id": "mkt_abc123",
    "question": "Will ETH be above $3000 by end of day?",
    "status": "trading",
    ...
  }
]
```

### Get Market

```bash
GET /api/market/{id}
```

### Resolve Market (Admin)

```bash
POST /api/market/{id}/resolve
Content-Type: application/json

{
  "outcome": "YES"
}
```

**Response:**
```json
{
  "market": { ... },
  "total_payout": 10000,
  "positions": 5
}
```

---

## Position APIs

### Deposit USDC

```bash
POST /api/deposit
Content-Type: application/json

{
  "user_id": "0xabc123...",
  "amount": 10000000
}
```

> **Note:** Amount is in basis points. 10000000 = 1000 USDC

**Response:**
```json
{
  "user_id": "0xabc123...",
  "balance": 10000000
}
```

### Mint Shares

```bash
POST /api/mint
Content-Type: application/json

{
  "user_id": "0xabc123...",
  "market_id": "mkt_abc123",
  "amount": 100
}
```

> Mints equal YES and NO shares. Cost = amount * 10000 basis points

**Response:**
```json
{
  "user_id": "0xabc123...",
  "market_id": "mkt_abc123",
  "yes_shares": 100,
  "no_shares": 100,
  "balance": 9000000
}
```

### Get Position

```bash
GET /api/position/{userId}?market_id={marketId}
```

**Response:**
```json
{
  "user_id": "0xabc123...",
  "balance": 9000000,
  "position": {
    "market_id": "mkt_abc123",
    "yes_shares": 100,
    "no_shares": 50
  }
}
```

---

## Order APIs

### Place Order

```bash
POST /api/order
Content-Type: application/json

{
  "user_id": "0xabc123...",
  "market_id": "mkt_abc123",
  "outcome_id": "YES",
  "side": "buy",
  "price": 6000,
  "quantity": 10
}
```

> **Price:** 0-10000 basis points (6000 = 60¢ = 60% probability)
> **Side:** "buy" or "sell"
> **Outcome:** "YES" or "NO"

**Response:**
```json
{
  "order": {
    "id": "ord_xyz789",
    "user_id": "0xabc123...",
    "market_id": "mkt_abc123",
    "outcome_id": "YES",
    "side": "buy",
    "price": 6000,
    "quantity": 10,
    "filled_qty": 0,
    "status": "open"
  },
  "trades": []
}
```

### Get Orderbook

```bash
GET /api/orderbook
```

**Response:**
```json
{
  "bids": [
    {"price": 6000, "quantity": 10, "count": 1}
  ],
  "asks": [
    {"price": 6500, "quantity": 5, "count": 1}
  ]
}
```

### Cancel Order

```bash
DELETE /api/order/{orderId}
```

### Get Trades

```bash
GET /api/trades
```

---

## Testing Flow (cURL)

```bash
# 1. Health check
curl http://localhost:8080/api/health

# 2. Create market
curl -X POST http://localhost:8080/api/market \
  -H "Content-Type: application/json" \
  -d '{"question":"ETH > $3000?","resolves_at":"2026-02-08T00:00:00Z","creator_id":"admin"}'

# 3. Deposit (save user_id for later)
curl -X POST http://localhost:8080/api/deposit \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user1","amount":10000000}'

# 4. Place buy order (replace market_id)
curl -X POST http://localhost:8080/api/order \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user1","market_id":"MARKET_ID","outcome_id":"YES","side":"buy","price":6000,"quantity":10}'

# 5. Place opposing sell order (will match!)
curl -X POST http://localhost:8080/api/order \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user2","market_id":"MARKET_ID","outcome_id":"YES","side":"sell","price":5500,"quantity":5}'

# 6. Check trades
curl http://localhost:8080/api/trades
```

---

## WebSocket

```
ws://localhost:8080/ws
```

Receives real-time orderbook updates after each trade.
