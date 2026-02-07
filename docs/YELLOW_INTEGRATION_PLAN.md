# Yellow Network- User Authentication

## 目标

将 Yellow Network 的用户认证流程与现有的订单簿撮合系统集成，实现：
1. ✅ 用户通过 MetaMask 钱包完成 Yellow Network 认证
2. ✅ 用户获得 JWT token 后可以使用做市系统
3. ✅ 交易结果同步到 Yellow state channel
4. ✅ 零 gas 费交易，最终链上结算

---

## 问题诊断

### 当前认证失败原因

**Go 实现的问题：**
```go
// ❌ 错误：缺少必需参数
AuthRequestParams{
    ParticipantAddress: address,
    Timestamp: timestamp,
}

// ❌ 错误：使用 EIP-191 签名
signature := signer.SignMessageHex(challenge)
```

**TypeScript 正确实现：**
```typescript
// ✅ 正确：包含所有必需参数
const authParams = {
    session_key: sessionAddress,        // 临时会话密钥
    allowances: [{                      // 资产授权
        asset: 'ytest.usd',
        amount: '1000000000'
    }],
    expires_at: BigInt(Date.now() / 1000 + 3600),  // 过期时间
    scope: 'test.app',                  // 应用范围
};

// ✅ 正确：使用 EIP-712 签名
const signer = createEIP712AuthMessageSigner(
    walletClient,
    authParams,
    { name: 'Test app' }
);
```

---

## 解决方案架构

### 三层架构

```
┌──────────────────────────────────────────────────────────────┐
│                   Layer 1: 前端用户层                          │
│  • 用户钱包认证（MetaMask）                                     │
│  • Yellow EIP-712 签名                                        │
│  • 获取 JWT token                                             │
│  • WebSocket 连接到后端                                        │
└──────────────────────────────────────────────────────────────┘
                            │ JWT token
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                  Layer 2: 订单簿后端层                         │
│  • JWT token 验证                                             │
│  • 用户 session 管理                                          │
│  • 订单撮合引擎                                                │
│  • Broker 账户管理流动性                                       │
└──────────────────────────────────────────────────────────────┘
                            │ State updates
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                 Layer 3: Yellow Network 层                    │
│  • State channel 管理                                         │
│  • Off-chain 余额更新                                         │
│  • 链上最终结算                                                │
└──────────────────────────────────────────────────────────────┘
```

---

## 实施计划

### Phase 1: 修复服务器端认证（Broker 账户）

**目标：** 让服务器的 broker 账户能够成功认证到 Yellow Network

**需要修改的文件：**
1. `orderbook-backend/internal/yellow/message.go` - 添加完整的 auth 参数
2. `orderbook-backend/internal/yellow/signer.go` - 实现 EIP-712 签名
3. `orderbook-backend/internal/yellow/client.go` - 更新认证流程

**关键更新：**

```go
// 1. 新的 AuthRequestParams 结构
type AuthRequestParams struct {
    Address    string              `json:"address"`
    SessionKey string              `json:"session_key"`
    Allowances []AuthAllowance     `json:"allowances"`
    ExpiresAt  int64               `json:"expires_at"`
    Scope      string              `json:"scope"`
    Application string             `json:"application"`
}

type AuthAllowance struct {
    Asset  string `json:"asset"`
    Amount string `json:"amount"`
}

// 2. EIP-712 TypedData 签名
func (s *Signer) SignEIP712Auth(
    challenge string,
    params AuthRequestParams,
    domain EIP712Domain,
) (string, error) {
    // 实现 EIP-712 签名逻辑
}

// 3. 更新 Authenticate 流程
func (c *Client) Authenticate(ctx context.Context) error {
    // 1. 生成临时 session keypair
    sessionKey := generateSessionKey()

    // 2. 发送 auth_request（包含完整参数）
    authReq := NewAuthRequest(c.signer.AddressHex(), sessionKey, ...)
    resp := c.SendRequest(ctx, authReq)

    // 3. 使用 EIP-712 签名 challenge
    sig := c.signer.SignEIP712Auth(challenge, authParams, domain)

    // 4. 发送 auth_verify
    verifyReq := NewAuthVerify(sig)
    resp = c.SendRequest(ctx, verifyReq)

    // 5. 保存 JWT token
    c.jwtToken = resp.Token
}
```

---

### Phase 2: 前端用户认证

**目标：** 让用户在浏览器中完成 Yellow 认证

**新增文件：**
- `orderbook-frontend/lib/yellow-auth.ts` - Yellow 认证 SDK 封装
- `orderbook-frontend/components/YellowConnect.tsx` - 认证 UI 组件

**用户认证流程：**

```typescript
// 1. 用户点击 "Connect Yellow Network"
async function connectYellow() {
    // 生成临时 session keypair
    const sessionPrivateKey = generatePrivateKey();
    const sessionAccount = privateKeyToAccount(sessionPrivateKey);

    // 2. 准备认证参数
    const authParams = {
        session_key: sessionAccount.address,
        allowances: [{ asset: 'ytest.usd', amount: '1000000000' }],
        expires_at: BigInt(Math.floor(Date.now() / 1000) + 3600),
        scope: 'orderbook.app',
    };

    // 3. 连接 WebSocket
    const ws = new WebSocket('wss://clearnet-sandbox.yellow.com/ws');

    // 4. 发送 auth_request
    const authRequestMsg = await createAuthRequestMessage({
        address: userAddress,
        application: 'OrderbookTrade',
        ...authParams
    });
    ws.send(authRequestMsg);

    // 5. 收到 challenge，使用 MetaMask 签名（EIP-712）
    ws.onmessage = async (event) => {
        const response = JSON.parse(event.data);

        if (response.res[1] === 'auth_challenge') {
            const challenge = response.res[2].challenge_message;

            // 用户的 MetaMask 签名
            const signer = createEIP712AuthMessageSigner(
                walletClient,
                authParams,
                { name: 'OrderbookTrade' }
            );

            const verifyMsg = await createAuthVerifyMessageFromChallenge(
                signer,
                challenge
            );
            ws.send(verifyMsg);
        }

        if (response.res[1] === 'auth_verify') {
            // 6. 获得 JWT token
            const jwtToken = response.res[2].jwt_token;
            const sessionKey = response.res[2].session_key;

            // 7. 使用 JWT token 连接到你的后端
            connectToOrderbook(jwtToken, sessionKey);
        }
    };
}

// 8. 连接到你的订单簿后端
function connectToOrderbook(jwtToken: string, sessionKey: string) {
    const ws = new WebSocket('ws://localhost:8080/ws');

    ws.onopen = () => {
        ws.send(JSON.stringify({
            type: 'auth',
            jwt_token: jwtToken,
            session_key: sessionKey
        }));
    };

    // 现在可以发送订单了
    ws.send(JSON.stringify({
        type: 'place_order',
        order: { ... }
    }));
}
```

---

### Phase 3: 后端集成 JWT 验证

**目标：** 验证用户的 Yellow JWT token，管理用户 session

**需要修改的文件：**
1. `orderbook-backend/internal/api/ws_handler.go` - WebSocket 认证
2. `orderbook-backend/internal/yellow/jwt.go` - JWT 验证（新建）
3. `orderbook-backend/internal/api/session_handler.go` - 用户 session 管理

**实现要点：**

```go
// 1. JWT 验证器
type JWTVerifier struct {
    yellowPublicKey *ecdsa.PublicKey
}

func (v *JWTVerifier) VerifyToken(tokenString string) (*UserClaims, error) {
    // 解析 JWT token
    // 验证签名
    // 返回用户地址和 session_key
}

// 2. WebSocket 认证
func (h *WSHandler) HandleConnection(ws *websocket.Conn) {
    var authMsg AuthMessage
    ws.ReadJSON(&authMsg)

    // 验证 JWT token
    claims, err := h.jwtVerifier.VerifyToken(authMsg.JWTToken)
    if err != nil {
        ws.WriteJSON(ErrorResponse{Error: "Invalid token"})
        return
    }

    // 创建用户 session
    session := &UserSession{
        Address:    claims.Address,
        SessionKey: authMsg.SessionKey,
        JWTToken:   authMsg.JWTToken,
        WS:         ws,
    }

    h.sessions.Add(session)

    // 处理订单消息
    for {
        var msg OrderMessage
        ws.ReadJSON(&msg)
        h.handleOrder(session, msg)
    }
}

// 3. 订单撮合后同步到 Yellow
func (h *WSHandler) handleTrade(trade *Trade) {
    // 1. 更新本地余额
    h.positions.UpdateBalance(trade)

    // 2. 同步到 Yellow state channel（如果用户有 Yellow session）
    if h.yellowClient != nil && h.yellowClient.IsAuthenticated() {
        h.syncTradeToYellow(trade)
    }
}

func (h *WSHandler) syncTradeToYellow(trade *Trade) {
    // 计算新的 allocations
    newAllocs := calculateAllocations(trade)

    // 更新 state channel
    err := h.yellowSession.UpdateState(ctx, newAllocs, tradeData)
    if err != nil {
        log.Printf("Failed to sync to Yellow: %v", err)
    }
}
```

---

## 数据流示例

### 完整的交易流程

```
1. 用户 A 认证
   Frontend → Yellow ClearNode → JWT token → Backend

2. 用户 A 下单买入
   Frontend → Backend WebSocket → Orderbook Engine

3. 用户 B 认证
   Frontend → Yellow ClearNode → JWT token → Backend

4. 用户 B 下单卖出
   Frontend → Backend WebSocket → Orderbook Engine

5. 撮合成功
   Engine matches → Update local balances → Notify users via WS

6. 同步到 Yellow（可选）
   Backend → Yellow ClearNode → Update state channel allocations

7. 用户提款
   Frontend → Backend → Yellow ClearNode → Close channel → On-chain settlement
```

---

## 关键技术要点

### 1. EIP-712 Domain

```go
type EIP712Domain struct {
    Name              string `json:"name"`
    Version           string `json:"version"`
    ChainId           int64  `json:"chainId"`
    VerifyingContract string `json:"verifyingContract"`
}

// Yellow Network 的 domain（需要从文档确认）
domain := EIP712Domain{
    Name:    "Yellow Network",
    Version: "1",
    ChainId: 1,  // 主网
}
```

### 2. Session Key 管理

- **服务器 Broker**：使用配置的 PRIVATE_KEY
- **用户**：浏览器生成临时 keypair，只在会话期间有效
- **安全性**：Session key 只能操作 allowances 范围内的资产

### 3. JWT Token 生命周期

```
User connects → Yellow auth → JWT token (1 hour TTL)
    │
    ├─ Token valid → Allow trading
    │
    └─ Token expires → Re-auth required
```

---

## 测试计划

### Phase 1 测试
- [ ] Broker 账户成功认证到 Yellow sandbox
- [ ] 可以创建 test channel
- [ ] 可以进行 resize 操作

### Phase 2 测试
- [ ] 前端可以触发 MetaMask 签名
- [ ] 成功获得 JWT token
- [ ] Token 可以连接到后端 WebSocket

### Phase 3 测试
- [ ] 后端验证 JWT token 成功
- [ ] 用户可以下单交易
- [ ] 交易成功同步到 Yellow state channel

---

## 风险和注意事项

1. **Yellow API 版本**：
   - 确保使用的 RPC 方法和参数与最新文档一致
   - TypeScript SDK (`@erc7824/nitrolite`) 是官方实现，Go 需要对齐

2. **签名格式**：
   - EIP-712 的 TypedData 结构必须完全匹配
   - Domain separator 必须正确

3. **Session 管理**：
   - JWT token 过期需要重新认证
   - WebSocket 断线重连机制

4. **安全性**：
   - 验证 JWT 签名来自 Yellow Network
   - 防止 token 重放攻击
   - Session key 权限最小化

---

## 下一步行动

1. **立即行动**：修复 Phase 1（服务器认证）
2. **中期**：实现 Phase 2（前端用户认证）
3. **最终**：完成 Phase 3（后端集成）

每个 Phase 独立测试通过后再进行下一个。
