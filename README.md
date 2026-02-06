# OrderbookTrade-Yellow
CLOB matching engine on Yellow — 0 gas order matching via Nitrolite state channels, on-chain settlement.

[HackMoney 2026](https://ethglobal.com/events/hackmoney2026)Submission — Yellow Network Prize Track





## What is OrderbookTrade-Yellow

**OrderbookTrade-Yellow** is a **CLOB** based matching engine that runs entirely off-chain through Yellow Network state channels. 

Users deposit once, trade unlimited times with zero gas, and settle on-chain when they're done .



And **OrderbookTrade** is an infrastructure for prediction markets ,enabling newly launched markets to go live with CEX-like trading UX .





## Project Architecture

```
User (Maker/Taker)
    │
    ▼  Wallet signature
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



## How We aggregate Yellow SDK



## Project Structure



