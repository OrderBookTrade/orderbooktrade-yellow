# Yellow Network 认证调试指南

## 当前问题

用户看到认证流程进行到 auth_verify，但没有成功完成。

## 日志分析

从控制台日志：
```
[Yellow Auth] Received: {res: Array(4), sig: Array(1)}
[Yellow Auth] Received challenge, requesting signature...
[Yellow Auth] Signing with EIP-712...
[Yellow Auth] Sending auth_verify...
[Yellow Auth] Verify response: ...
```

## 可能的问题

### 1. 响应格式不匹配

Yellow Network 的响应格式是：
```javascript
{
  res: [requestId, method, result, timestamp],
  sig: [signature]
}
```

检查要点：
- `res[1]` 应该是方法名，如 `'auth_verify'`
- `res[2]` 应该是结果对象

### 2. 字段名差异

可能的字段名：
- `session_key` vs `sessionKey`
- `jwt_token` vs `jwtToken` vs `token`
- `expires_at` vs `expiresAt`

## 调试步骤

### 步骤 1: 查看完整响应

打开浏览器控制台，输入：
```javascript
// 全局变量存储最后的响应
window.lastYellowResponse = null;
```

然后在代码中添加：
```javascript
window.lastYellowResponse = response;
console.log('[Yellow Auth] Full response:', JSON.stringify(response, null, 2));
```

### 步骤 2: 检查 Yellow ClearNode 响应

使用浏览器开发工具：
1. 打开 DevTools → Network 标签
2. 过滤 WS (WebSocket)
3. 找到 `clearnet-sandbox.yellow.com` 连接
4. 查看 Messages 标签
5. 找到 auth_verify 的响应

### 步骤 3: 手动测试 Yellow 连接

创建一个独立的测试文件：

```html
<!DOCTYPE html>
<html>
<head>
    <title>Yellow Auth Test</title>
</head>
<body>
    <h1>Yellow Network Auth Test</h1>
    <button onclick="testAuth()">Test Auth</button>
    <pre id="log"></pre>

    <script type="module">
        import { generatePrivateKey, privateKeyToAccount } from 'viem/accounts';
        import {
            createECDSAMessageSigner,
            createAuthRequestMessage,
        } from '@erc7824/nitrolite';

        window.testAuth = async function() {
            const log = document.getElementById('log');
            const appendLog = (msg) => {
                log.textContent += msg + '\n';
            };

            try {
                // Generate session key
                const sessionPrivateKey = generatePrivateKey();
                const sessionAccount = privateKeyToAccount(sessionPrivateKey);
                appendLog('Session key: ' + sessionAccount.address);

                // Connect to Yellow
                const ws = new WebSocket('wss://clearnet-sandbox.yellow.com/ws');

                ws.onopen = async () => {
                    appendLog('WebSocket connected');

                    // Get wallet address (you need MetaMask)
                    const accounts = await window.ethereum.request({
                        method: 'eth_requestAccounts'
                    });
                    const address = accounts[0];
                    appendLog('Wallet: ' + address);

                    // Send auth_request
                    const sessionSigner = createECDSAMessageSigner(sessionPrivateKey);
                    const authParams = {
                        session_key: sessionAccount.address,
                        allowances: [{ asset: 'ytest.usd', amount: '1000000000' }],
                        expires_at: BigInt(Math.floor(Date.now() / 1000) + 3600),
                        scope: 'test.app',
                    };

                    const authRequestMsg = await createAuthRequestMessage({
                        address: address,
                        application: 'Test',
                        ...authParams
                    });

                    appendLog('Sending auth_request...');
                    ws.send(authRequestMsg);
                };

                ws.onmessage = (event) => {
                    appendLog('Received: ' + event.data);
                    const response = JSON.parse(event.data);
                    appendLog('Parsed: ' + JSON.stringify(response, null, 2));
                };

                ws.onerror = (error) => {
                    appendLog('Error: ' + error);
                };

            } catch (err) {
                appendLog('Error: ' + err.message);
            }
        };
    </script>
</body>
</html>
```

## 常见问题

### 问题 1: auth_verify 响应未收到

**症状：**
- 发送了 auth_verify
- 没有收到对应的响应

**可能原因：**
- Yellow ClearNode 拒绝了签名
- 签名格式不正确
- EIP-712 domain 不匹配

**解决方案：**
1. 检查 EIP-712 签名的 domain name
2. 确认签名的 TypedData 结构与 Yellow 要求一致

### 问题 2: 响应中缺少 jwt_token

**症状：**
- 收到了 auth_verify 响应
- 但 `result.jwt_token` 是 undefined

**可能原因：**
- Yellow sandbox 可能不返回 JWT token
- 或者字段名不同

**解决方案：**
```javascript
// 尝试多种可能的字段名
const jwtToken = result.jwt_token || result.jwtToken || result.token || '';
```

### 问题 3: EIP-712 签名失败

**症状：**
- MetaMask 弹出签名请求
- 签名后 Yellow 返回错误

**可能原因：**
- Domain separator 不正确
- TypedData 结构不匹配

**解决方案：**
查看 Yellow Network 的最新文档，确认 EIP-712 domain：
```javascript
{
  name: 'Yellow Network',  // 或 'OrderbookTrade'?
  version: '1',
  chainId: ???,  // Sepolia: 11155111
  verifyingContract: ???
}
```

## 快速修复

如果 Yellow sandbox 不返回 JWT token，可以使用简化版本：

```typescript
// In useYellowAuth.ts
if (response.res && response.res[1] === 'auth_verify') {
    const result = response.res[2];

    // Log everything we received
    console.log('[Yellow Auth] Full result:', JSON.stringify(result, null, 2));

    // Try multiple field names
    const sessionKey = result.session_key || result.sessionKey || sessionAccount.address;
    const jwtToken = result.jwt_token || result.jwtToken || result.token || 'mock_token_for_testing';
    const expiresAt = result.expires_at || result.expiresAt || (Date.now() / 1000 + 3600);

    resolve({
        sessionKey,
        jwtToken,
        expiresAt,
    });
}
```

## 联系 Yellow Network

如果问题持续，可以：
1. 查看 Yellow Network 文档：https://docs.yellow.org
2. 检查 @erc7824/nitrolite SDK 的示例
3. 在 Yellow Discord/Telegram 寻求帮助

## 临时解决方案

如果只是为了测试，可以暂时跳过 Yellow 认证：

```typescript
// In app/page.tsx
const yellowAuth = {
    isConnected: false,  // 设为 false 跳过 Yellow
    jwtToken: null,
    sessionKey: null,
};
```

这样你可以先测试订单簿功能，稍后再修复 Yellow 认证。
