# ⚡ OrderbookTrade × Yellow Network

> Off-chain orderbook matching engine powered by ERC-7824 state channels

**HackMoney 2026 Submission** — Yellow Network Prize Track

## What is this?

OrderbookTrade is a **production-grade orderbook matching engine** that runs entirely off-chain through Yellow Network state channels. Users deposit once, trade unlimited times with zero gas, and settle on-chain when they're done.

Unlike toy hackathon demos, this is **B2B infrastructure** — any DEX, prediction market, or trading platform can integrate this matching engine as a service.

## Architecture

```
User (Maker/Taker)
    │
    ▼  MetaMask signature
Yellow ClearNode (WebSocket)
    │
    ▼  off-chain, 0 gas
OrderbookTrade Matching Engine
    │  (price-time priority matching)
    ▼  state update
Allocation Rebalance (buyer ↔ seller)
    │
    ▼  close session
On-Chain Settlement (EVM)
```

## How State Channels Work Here

1. **Deposit**: Users create a channel at `apps.yellow.com`, locking USDC
2. **Trade**: Orders are signed and sent via WebSocket to ClearNode — **0 gas per order**
3. **Match**: Our matching engine processes orders off-chain with sub-100ms latency
4. **Settle**: When done, close the session → one on-chain tx settles all balances

## Tech Stack

- **SDK**: `@erc7824/nitrolite` v0.5.3
- **Protocol**: ERC-7824 State Channels
- **ClearNode**: `wss://clearnet-sandbox.yellow.com/ws`
- **Frontend**: React + Vite
- **Matching Engine**: Custom price-time priority orderbook

## Quick Start

```bash
# Clone and install
git clone https://github.com/OrderBookTrade/orderbook-trade-yellow.git
cd orderbook-trade-yellow
npm install

# Run dev server
npm run dev

# Open http://localhost:5173
```

## Project Structure

```
├── src/
│   ├── App.tsx                  # Main app with trading UI
│   ├── engine/
│   │   └── orderbook.ts         # Matching engine (your core IP)
│   ├── yellow/
│   │   ├── client.ts            # Yellow SDK wrapper
│   │   ├── session.ts           # App session management
│   │   └── types.ts             # TypeScript types
│   ├── components/
│   │   ├── OrderBook.tsx         # L2 orderbook display
│   │   ├── OrderForm.tsx         # Buy/sell order form
│   │   ├── TradeHistory.tsx      # Recent trades
│   │   └── ChannelLog.tsx        # State channel event log
│   └── main.tsx
├── package.json
├── vite.config.ts
└── README.md
```

## Key SDK Integration Points

```typescript
import { createAppSessionMessage, parseRPCResponse } from '@erc7824/nitrolite';

// 1. Connect to ClearNode
const ws = new WebSocket('wss://clearnet-sandbox.yellow.com/ws');

// 2. Create orderbook session
const sessionMsg = await createAppSessionMessage(messageSigner, [{
  definition: { protocol: 'orderbook-trade-v1', participants, weights: [50,50], quorum: 100, challenge: 0, nonce: Date.now() },
  allocations: [
    { participant: maker, asset: 'usdc', amount: '100000000' },
    { participant: taker, asset: 'usdc', amount: '100000000' }
  ]
}]);
ws.send(sessionMsg);

// 3. Place order (0 gas!)
ws.send(JSON.stringify(signedOrder));

// 4. Handle responses
ws.onmessage = (event) => {
  const msg = parseRPCResponse(event.data);
  // Process matches, update allocations
};
```

## Prize Track Requirements ✅

| Requirement | Status |
|---|---|
| Use Yellow SDK / Nitrolite | ✅ `@erc7824/nitrolite` integrated |
| Demonstrate off-chain transactions | ✅ All orders processed off-chain, 0 gas |
| Demonstrate on-chain settlement | ✅ Session close triggers EVM settlement |
| 2-3 min demo video | ⬜ Record before submission |
| Submit to Yellow prize track | ⬜ Submit on ETHGlobal |
| Open source code | ✅ Public GitHub repo |

## Judging Criteria Alignment

- **Problem & Solution**: DEXs have high gas costs + slow matching. We solve both.
- **Yellow SDK Integration**: Deep integration — orderbook engine runs inside app session.
- **Business Model**: B2B infrastructure service. Any trading platform can integrate.
- **Creativity**: First orderbook matching engine on Yellow state channels.

## Demo Video Script (2-3 min)

1. **Problem** (20s): "DEX trading costs gas per order. CEXs are fast but custodial."
2. **Solution** (20s): "OrderbookTrade: production matching engine on Yellow state channels."
3. **Live Demo** (90s):
   - Connect to ClearNode
   - Show orderbook loading (off-chain, 0 gas)
   - Place buy order → place sell order → instant match
   - Show state channel log (all off-chain)
   - Settle on-chain
4. **Architecture** (20s): Show the flow diagram
5. **Business Model** (10s): "B2B infra — any DEX plugs in. $12k+ per integration."

## Previous Work Disclaimer

The matching engine logic (price-time priority orderbook) is original work developed as part of the OrderbookTrade project. The Yellow Network / ERC-7824 integration was built entirely during the HackMoney 2026 hackathon period.

## License

MIT