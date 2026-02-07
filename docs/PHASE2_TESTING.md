# Phase 2 完成：前端用户 Yellow Network 认证

## ✅ 已完成的修改

### 前端 (Frontend)

#### 1. useYellowAuth Hook (`hooks/useYellowAuth.ts`) ✅
- 完整的 Yellow Network 认证流程
- 生成临时 session keypair
- 发送 `auth_request` 到 Yellow ClearNode
- 通过 MetaMask 签名 EIP-712 challenge
- 获取并存储 JWT token
- 管理认证状态和错误处理

#### 2. YellowConnect 组件 (`components/YellowConnect.tsx`) ✅
- 显示 Yellow Network 连接状态
- 连接/断开按钮
- Session key 和过期时间显示
- 错误消息展示
- 签名进度指示器

#### 3. useWebSocket Hook 更新 ✅
- 添加 `yellowToken` 和 `sessionKey` 参数
- 连接时自动发送 Yellow auth 到后端
- 支持带 Yellow 认证的 WebSocket 连接

#### 4. 主页面集成 (`app/page.tsx`) ✅
- 导入并使用 `useYellowAuth` hook
- 将 Yellow auth 状态传递给 WebSocket
- UI 中添加 `YellowConnect` 组件

### 后端 (Backend)

#### 1. JWT 验证器 (`internal/yellow/jwt.go`) ✅
- `ParseJWT()` - 解析 Yellow JWT token
- `ValidateToken()` - 验证 token 并创建 session
- `YellowAuthMessage` - WebSocket auth 消息结构
- `ParseYellowAuth()` - 解析前端发来的 auth 消息

#### 2. WebSocket Handler 更新 (`internal/api/ws_handler.go`) ✅
- `Client` 结构体添加 Yellow session 字段
- `readPump()` 处理传入的 Yellow auth 消息
- `handleYellowAuth()` 验证 JWT 并存储 session
- 发送成功/失败响应到前端

---

## 🎯 认证流程

### 完整的用户认证流程

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 用户连接 MetaMask                                         │
│    Frontend → MetaMask → Get wallet address                │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. 用户点击 "Connect Yellow"                                 │
│    Frontend → useYellowAuth.connect()                      │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. 生成 Session Keypair                                     │
│    const sessionPrivateKey = generatePrivateKey()          │
│    const sessionKey = privateKeyToAccount(sk).address      │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. 连接 Yellow ClearNode WebSocket                          │
│    ws://clearnet-sandbox.yellow.com/ws                     │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. 发送 auth_request                                         │
│    { address, session_key, allowances, expires_at, ... }   │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. Yellow 返回 auth_challenge                                │
│    { challenge_message: "random_string" }                  │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. MetaMask 签名 EIP-712                                     │
│    User approves signature in MetaMask popup                │
│    createEIP712AuthMessageSigner() → signature             │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. 发送 auth_verify                                          │
│    { signature, challenge, address, ... }                  │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 9. Yellow 返回 JWT Token                                     │
│    { session_key, jwt_token, expires_at }                  │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 10. 连接后端 WebSocket                                       │
│     ws://localhost:8080/ws                                 │
│     Send: { type: "yellow_auth", jwt_token, session_key }  │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 11. 后端验证 JWT Token                                       │
│     Backend → ValidateToken() → Create session             │
│     Send: { type: "yellow_auth_success" }                  │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 12. ✅ 用户已认证，可以开始交易！                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 🧪 测试步骤

### 前置条件

1. **Phase 1 已完成**：服务器端 Yellow 认证成功
2. **前端依赖已安装**：
   ```bash
   cd orderbook-frontend
   npm install
   ```

3. **环境变量配置**：

   `.env.local` (前端):
   ```bash
   NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
   NEXT_PUBLIC_YELLOW_WS_URL=wss://clearnet-sandbox.yellow.com/ws
   ```

   `.env` (后端):
   ```bash
   SERVER_PORT=8080
   PRIVATE_KEY=0x你的服务器私钥
   YELLOW_NODE_URL=wss://clearnet-sandbox.yellow.com/ws
   ```

### 步骤 1: 启动后端

```bash
cd orderbook-backend
go run cmd/server/main.go
```

**查找这些日志：**
```
✓ Yellow SDK: Authenticated successfully
🟢 Yellow Network: CONNECTED and ready
Server starting on :8080
```

### 步骤 2: 启动前端

```bash
cd orderbook-frontend
npm run dev
```

访问 http://localhost:3000

### 步骤 3: 连接钱包

1. 点击 **"Connect Wallet"** 按钮
2. MetaMask 弹出，选择账户并批准
3. 看到你的地址显示在页面右上角

### 步骤 4: 连接 Yellow Network

1. 在 Yellow Network 组件中，点击 **"Connect Yellow"** 按钮
2. **预期行为**：
   - 状态变为 "Connecting..."
   - 出现进度提示："Waiting for MetaMask signature..."
   - MetaMask 弹出 **签名请求**（EIP-712）
   - 内容类似：
     ```
     OrderbookTrade

     Address: 0xYourAddress
     Session Key: 0xSessionAddress
     Challenge Message: random_challenge_string
     Allowances: [...]
     Expires At: 1234567890
     Scope: orderbook.app
     Application: OrderbookTrade
     ```

3. **在 MetaMask 中点击 "签名"**

4. **预期结果**：
   - Yellow Network 组件状态变为 🟢 "Connected"
   - 显示 Session Key（截断显示）
   - 显示过期时间（如 "59m"）

### 步骤 5: 验证后端接收

**后端日志应显示：**
```
[Yellow Auth] Received Yellow auth: session_key=0x...
✓ Yellow auth successful for address: 0xYourAddress
```

**前端控制台应显示：**
```
[Yellow Auth] Starting authentication...
[Yellow Auth] Generated session key: 0x...
[Yellow Auth] WebSocket connected
[Yellow Auth] Sending auth_request...
[Yellow Auth] Received challenge, requesting signature...
[Yellow Auth] Signing with EIP-712...
[Yellow Auth] Sending auth_verify...
[Yellow Auth] ✓ Authentication successful!
[Yellow Auth] Session Key: 0x...
[Yellow Auth] JWT Token: eyJhbGciOiJ...
[WebSocket] Sending Yellow auth...
```

---

## 🔍 UI 状态说明

### Yellow Connect 组件的三种状态

#### 1. 未连接钱包
```
🟡 Yellow Network
   Connect wallet first
```
- 灰色/禁用状态
- 用户需要先连接 MetaMask

#### 2. 已连接钱包，未连接 Yellow
```
⚪ Yellow Network
   Not connected

   [Connect Yellow] 按钮
```
- 白色圆点
- 显示连接按钮

#### 3. 已连接 Yellow
```
🟢 Yellow Network
   Session: 0x1234...5678 • Expires: 59m

   [Disconnect] 按钮
```
- 绿色圆点
- 显示 session key 和过期时间
- 可以断开连接

### 认证进度提示

当用户点击 "Connect Yellow" 后：
```
┌─────────────────────────────────────────┐
│ 🔄 Waiting for MetaMask signature...   │
│                                         │
│ Please sign the EIP-712 message to     │
│ authenticate with Yellow Network        │
└─────────────────────────────────────────┘
```

---

## 🐛 故障排除

### 问题 1: MetaMask 签名请求未弹出

**可能原因：**
- MetaMask 未解锁
- 浏览器阻止了弹窗

**解决方案：**
1. 确保 MetaMask 已解锁
2. 检查浏览器地址栏是否有弹窗被阻止的提示
3. 刷新页面重试

### 问题 2: "Invalid Yellow authentication"

**可能原因：**
- JWT token 格式错误
- Token 已过期

**解决方案：**
1. 检查前端控制台的错误信息
2. 确认 Yellow ClearNode 连接成功
3. 重新连接 Yellow Network

### 问题 3: WebSocket 连接失败

**可能原因：**
- 后端未启动
- 端口被占用
- CORS 问题

**解决方案：**
```bash
# 检查后端是否运行
lsof -i :8080

# 检查后端日志
go run cmd/server/main.go

# 检查前端 WS_URL 配置
cat orderbook-frontend/.env.local
```

### 问题 4: Yellow auth timeout

**可能原因：**
- Yellow ClearNode 不可达
- 网络防火墙阻止 WebSocket

**解决方案：**
```bash
# 测试 Yellow 连接
curl -I wss://clearnet-sandbox.yellow.com/ws

# 检查浏览器控制台网络请求
# DevTools → Network → WS → 查看连接状态
```

---

## 📊 数据流图

### 前端 → Yellow → 后端

```
┌─────────────┐
│  Browser    │
│  (React)    │
└──────┬──────┘
       │
       │ useYellowAuth.connect()
       │
       ▼
┌─────────────────────┐      ┌──────────────────┐
│ Yellow ClearNode    │◄────►│ MetaMask         │
│ (wss://...)         │      │ (EIP-712 Sign)   │
└──────┬──────────────┘      └──────────────────┘
       │
       │ JWT Token
       │
       ▼
┌─────────────────────┐
│ Your Backend WS     │
│ (ws://localhost)    │
└──────┬──────────────┘
       │
       │ Validate & Create Session
       │
       ▼
┌─────────────────────┐
│ Orderbook Engine    │
│ (Trading ready)     │
└─────────────────────┘
```

---

## ✨ 关键代码片段

### 前端：触发认证

```typescript
import { useYellowAuth } from '@/hooks/useYellowAuth';

function MyComponent() {
  const { address } = useWallet();
  const {
    isConnected,
    isAuthenticating,
    jwtToken,
    error,
    connect
  } = useYellowAuth(address);

  return (
    <button onClick={connect} disabled={isAuthenticating}>
      {isAuthenticating ? 'Connecting...' : 'Connect Yellow'}
    </button>
  );
}
```

### 前端：使用 Yellow 认证的 WebSocket

```typescript
const yellowAuth = useYellowAuth(address);
const { connected } = useWebSocket({
  yellowToken: yellowAuth.jwtToken,
  sessionKey: yellowAuth.sessionKey,
});
```

### 后端：验证 JWT

```go
// In ws_handler.go
func (c *Client) handleYellowAuth(msg *yellow.YellowAuthMessage) {
    session, err := yellow.ValidateToken(msg.JWTToken)
    if err != nil {
        // Send error
        return
    }

    c.yellowAddress = session.Address
    c.yellowSessionKey = msg.SessionKey
    // User authenticated!
}
```

---

## 🎯 下一步：Phase 3

Phase 2 完成后，你现在有：
- ✅ 用户可以通过 MetaMask 认证到 Yellow Network
- ✅ 用户获得自己的 JWT token
- ✅ 后端可以验证用户的 Yellow session

**接下来（Phase 3）：**
1. 将交易撮合结果同步到 Yellow state channel
2. 实现 Yellow channel 的创建和管理
3. 支持链上结算

详见：`docs/YELLOW_INTEGRATION_PLAN.md` Phase 3 部分

---

## 📚 技术参考

### EIP-712 TypedData 示例

MetaMask 签名请求的内容：

```json
{
  "types": {
    "EIP712Domain": [
      { "name": "name", "type": "string" },
      { "name": "version", "type": "string" }
    ],
    "AuthVerify": [
      { "name": "address", "type": "address" },
      { "name": "session_key", "type": "address" },
      { "name": "challenge_message", "type": "string" },
      { "name": "allowances", "type": "Allowance[]" },
      { "name": "expires_at", "type": "uint256" },
      { "name": "scope", "type": "string" },
      { "name": "application", "type": "string" }
    ],
    "Allowance": [
      { "name": "asset", "type": "string" },
      { "name": "amount", "type": "string" }
    ]
  },
  "primaryType": "AuthVerify",
  "domain": {
    "name": "OrderbookTrade",
    "version": "1"
  },
  "message": {
    "address": "0xUserAddress",
    "session_key": "0xSessionAddress",
    "challenge_message": "random_string",
    "allowances": [
      { "asset": "ytest.usd", "amount": "1000000000" }
    ],
    "expires_at": "1234567890",
    "scope": "orderbook.app",
    "application": "OrderbookTrade"
  }
}
```

---

## 🎉 成功标志

如果你看到以下所有内容，Phase 2 就完成了！

- ✅ 前端显示 Yellow Connect 组件
- ✅ 用户可以点击 "Connect Yellow"
- ✅ MetaMask 弹出 EIP-712 签名请求
- ✅ 签名后，Yellow 状态变为 🟢 "Connected"
- ✅ 后端日志显示 "Yellow auth successful"
- ✅ WebSocket 连接包含 Yellow session 信息

**恭喜！你的用户现在可以通过 Yellow Network 进行零 gas 费交易了！** 🚀
