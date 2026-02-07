# Phase 1 完成：服务器端 Yellow 认证修复

## ✅ 已完成的修改

### 1. 更新认证消息结构 (`internal/yellow/message.go`)
- ✅ 添加 `AuthAllowance` 结构体
- ✅ 更新 `AuthRequestParams` 包含所有必需字段：
  - `address` - 主钱包地址
  - `session_key` - 临时会话密钥
  - `allowances` - 资产授权列表
  - `expires_at` - 过期时间
  - `scope` - 应用范围
  - `application` - 应用名称
- ✅ 更新 `AuthVerifyParams` 支持 EIP-712 签名验证
- ✅ 更新 `AuthVerifyResult` 包含 JWT token

### 2. 实现 EIP-712 签名 (`internal/yellow/signer.go`)
- ✅ 添加 `SignEIP712Auth()` 方法
- ✅ 实现完整的 EIP-712 TypedData 签名
- ✅ 添加 `GenerateSessionKey()` 生成临时密钥对
- ✅ 支持 allowances 数组的序列化

### 3. 更新客户端认证流程 (`internal/yellow/client.go`)
- ✅ 生成临时 session keypair
- ✅ 发送完整的 `auth_request`
- ✅ 使用 EIP-712 签名 challenge
- ✅ 保存 JWT token
- ✅ 添加详细的日志输出

---

## 🧪 测试步骤

### 前置条件

确保你的 `.env` 文件包含：

```bash
# Yellow Network 配置
PRIVATE_KEY=0x你的私钥
YELLOW_NODE_URL=wss://clearnet-sandbox.yellow.com/ws

# 其他配置...
SERVER_PORT=8080
```

### 方式 1: 运行测试程序（推荐）

```bash
# 进入后端目录
cd orderbook-backend

# 运行测试
go run cmd/test-yellow-auth/main.go
```

**预期输出：**

```
Yellow Network Authentication Test
===================================
✓ Signer initialized
  Address: 0xYourAddress
  Node URL: wss://clearnet-sandbox.yellow.com/ws

🔌 Connecting to Yellow Network...
✓ WebSocket connected

🔐 Authenticating...
Starting Yellow Network authentication...
  Generated session key: 0xSessionAddress
  Sending auth_request...
  Received challenge: challenge_string_here
  Signing challenge with EIP-712...
  Generated signature: 0x1234...
  Sending auth_verify...
✓ Authenticated successfully!
  Session Key: 0xSessionAddress
  JWT Token: eyJhbGciOiJ...
  Expires At: 2026-02-07T17:30:28+08:00

✅ SUCCESS! Authentication complete.

Next steps:
  1. Start the main server: go run cmd/server/main.go
  2. The server will auto-authenticate on startup
  3. Move to Phase 2: Implement frontend user authentication

👋 Test complete. Connection closed.
```

### 方式 2: 启动完整服务器

```bash
cd orderbook-backend
go run cmd/server/main.go
```

**查找这些日志：**

```
Starting Orderbook Backend (Prediction Market Mode)...
✓ Yellow SDK: Signer initialized (address: 0xYourAddress)
  Connecting to Yellow Network: wss://clearnet-sandbox.yellow.com/ws
✓ Yellow SDK: WebSocket connected
Starting Yellow Network authentication...
  Generated session key: 0x...
  Sending auth_request...
  Received challenge: ...
  Signing challenge with EIP-712...
  Sending auth_verify...
✓ Authenticated successfully!
✓ Yellow SDK: Authenticated successfully
🟢 Yellow Network: CONNECTED and ready
```

---

## 🐛 故障排除

### 问题 1: "auth request failed: request timeout"

**可能原因：**
- Yellow Network 节点不可达
- 网络防火墙阻止 WebSocket 连接

**解决方案：**
```bash
# 测试连接
curl -I wss://clearnet-sandbox.yellow.com/ws

# 尝试使用主网（如果有访问权限）
YELLOW_NODE_URL=wss://clearnet.yellow.com/ws go run cmd/test-yellow-auth/main.go
```

### 问题 2: "auth verify error: Invalid signature"

**可能原因：**
- EIP-712 TypedData 结构不匹配
- Domain separator 不正确

**调试步骤：**
1. 检查日志中的 challenge 和 signature
2. 对比 TypeScript 实现的 TypedData 结构
3. 验证 domain name 是否正确

### 问题 3: 编译错误

```bash
# 清理并重新构建
cd orderbook-backend
go mod tidy
go build ./...
```

---

## 📊 与 TypeScript 实现对比

### TypeScript (参考)
```typescript
const authParams = {
    session_key: sessionAddress,
    allowances: [{ asset: 'ytest.usd', amount: '1000000000' }],
    expires_at: BigInt(Date.now() / 1000 + 3600),
    scope: 'test.app',
};

const signer = createEIP712AuthMessageSigner(
    walletClient,
    authParams,
    { name: 'Test app' }
);
```

### Go (我们的实现)
```go
authParams := AuthRequestParams{
    Address:    signer.AddressHex(),
    SessionKey: sessionKey,
    Allowances: []AuthAllowance{{
        Asset:  "ytest.usd",
        Amount: "1000000000",
    }},
    ExpiresAt:   time.Now().Unix() + 3600,
    Scope:       "orderbook.app",
    Application: "OrderbookTrade",
}

signature, err := signer.SignEIP712Auth(
    challenge,
    authParams,
    authParams.Application,
)
```

---

## ✨ 关键改进

### Before (有问题的实现)
```go
// ❌ 缺少参数
AuthRequestParams{
    ParticipantAddress: address,
    Timestamp: timestamp,
}

// ❌ 错误的签名方式
signature := signer.SignMessageHex(challenge)
```

### After (正确的实现)
```go
// ✅ 完整参数
AuthRequestParams{
    Address:     address,
    SessionKey:  sessionKey,
    Allowances:  allowances,
    ExpiresAt:   expiresAt,
    Scope:       scope,
    Application: application,
}

// ✅ EIP-712 签名
signature := signer.SignEIP712Auth(challenge, params, domain)
```

---

## 🎯 下一步：Phase 2

Phase 1 完成后，你应该能够：
- ✅ 服务器成功连接到 Yellow Network
- ✅ Broker 账户认证成功
- ✅ 获得 JWT token

**接下来（Phase 2）：**
1. 在前端实现用户认证（MetaMask 签名）
2. 用户获得自己的 Yellow JWT token
3. 前端使用 token 连接到你的订单簿后端

详见：`docs/YELLOW_INTEGRATION_PLAN.md` Phase 2 部分

---

## 📝 技术细节

### EIP-712 TypedData 结构

```go
Types: {
    "EIP712Domain": [
        {Name: "name", Type: "string"},
        {Name: "version", Type: "string"},
    ],
    "AuthVerify": [
        {Name: "address", Type: "address"},
        {Name: "session_key", Type: "address"},
        {Name: "challenge_message", Type: "string"},
        {Name: "allowances", Type: "Allowance[]"},
        {Name: "expires_at", Type: "uint256"},
        {Name: "scope", Type: "string"},
        {Name: "application", Type: "string"},
    ],
    "Allowance": [
        {Name: "asset", Type: "string"},
        {Name: "amount", Type: "string"},
    ],
}
```

### 签名计算

```
hash = keccak256(
    "\x19\x01" +
    domainSeparator +
    structHash(AuthVerify, message)
)

signature = ECDSA.sign(hash, privateKey)
```

---

## 📚 参考资料

- [EIP-712 规范](https://eips.ethereum.org/EIPS/eip-712)
- [Yellow Network Nitrolite SDK](https://github.com/erc7824/nitrolite)
- [go-ethereum EIP-712 实现](https://github.com/ethereum/go-ethereum/tree/master/signer/core/apitypes)
