'use client';

import { useState, useEffect } from 'react';
import { useWebSocket, OrderbookData } from '@/hooks/useWebSocket';
import { useWallet } from '@/hooks/useWallet';
import { useYellowAuth } from '@/hooks/useYellowAuth';
import { OrderBook } from '@/components/OrderBook';
import { OrderForm } from '@/components/OrderForm';
import { TradeHistory } from '@/components/TradeHistory';
import { MarketCard } from '@/components/MarketCard';
import { ChannelLog, addLogEntry } from '@/components/ChannelLog';
import { YellowConnect } from '@/components/YellowConnect';
import { Market, listMarkets, createMarket, deposit, mintShares } from '@/lib/api';

type OutcomeTab = 'YES' | 'NO';

export default function Home() {
  const { address, isConnected, isConnecting, connect, error: walletError } = useWallet();
  const yellowAuth = useYellowAuth(address);
  const { yesOrderbook, noOrderbook, trades, connected, error: wsError } = useWebSocket({
    yellowToken: yellowAuth.jwtToken,
    sessionKey: yellowAuth.sessionKey,
  });
  const [selectedPrice, setSelectedPrice] = useState<number | undefined>();
  const [markets, setMarkets] = useState<Market[]>([]);
  const [selectedMarket, setSelectedMarket] = useState<Market | null>(null);
  const [showCreateMarket, setShowCreateMarket] = useState(false);
  const [activeOutcome, setActiveOutcome] = useState<OutcomeTab>('YES');

  // Get the orderbook based on active outcome
  const currentOrderbook: OrderbookData = activeOutcome === 'YES' ? yesOrderbook : noOrderbook;

  // Load markets on mount
  useEffect(() => {
    loadMarkets();
  }, []);

  const loadMarkets = async () => {
    try {
      const data = await listMarkets();
      setMarkets(data);
      if (data.length > 0 && !selectedMarket) {
        setSelectedMarket(data[0]);
      }
    } catch (err) {
      console.error('Failed to load markets:', err);
    }
  };

  const handleCreateDemoMarket = async () => {
    try {
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);

      const market = await createMarket({
        question: 'Will ETH be above $3000 by end of day?',
        description: 'Prediction market demo for EthGlobal - HackMoney hackathon',
        resolves_at: tomorrow.toISOString(),
        creator_id: address || 'demo',
      });

      setMarkets([...markets, market]);
      setSelectedMarket(market);
      addLogEntry('session', 'Market created', market);
    } catch (err) {
      console.error('Failed to create market:', err);
    }
  };

  const handleDeposit = async () => {
    if (!address) return;
    try {
      // Deposit 1000 USDC (in basis points: 10000000)
      await deposit({ user_id: address, amount: 10000000 });
      addLogEntry('state', 'Deposited 1000 USDC');
    } catch (err) {
      console.error('Failed to deposit:', err);
    }
  };

  const handleMint = async () => {
    if (!address || !selectedMarket) return;
    try {
      // Mint 100 share pairs (100 YES + 100 NO)
      const result = await mintShares({
        user_id: address,
        market_id: selectedMarket.id,
        amount: 100
      });
      addLogEntry('state', `Minted 100 YES + 100 NO shares`, result);
    } catch (err) {
      console.error('Failed to mint:', err);
    }
  };

  const handlePriceClick = (price: number) => {
    setSelectedPrice(price);
  };

  const handleOrderPlaced = () => {
    addLogEntry('state', 'Order placed successfully');
  };

  const formatAddress = (addr: string) => {
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
  };

  return (
    <div className="app">
      {/* Header */}
      <header className="header">
        <div className="logo">
          <img src="/logo.png" alt="Orderbook Trade" className="logo-image" />
          <h1>Prediction Market</h1>
          <span className="network-badge">Yellow Network</span>
        </div>

        <div className="header-right">
          {/* Yellow Network Connection */}
          <YellowConnect
            walletAddress={address}
            walletConnected={isConnected}
          />

          {/* Connection Status */}
          <div className={`connection-status ${connected ? 'connected' : 'disconnected'}`}>
            <span className="status-dot" />
            {connected ? 'Live' : 'Connecting...'}
          </div>

          {/* Wallet */}
          {isConnected ? (
            <div className="wallet-actions">
              <button className="deposit-btn" onClick={handleDeposit}>
                + Deposit
              </button>
              <button className="mint-btn" onClick={handleMint} disabled={!selectedMarket}>
                Mint Shares
              </button>
              <div className="wallet-info">
                <span className="wallet-address">{formatAddress(address!)}</span>
              </div>
            </div>
          ) : (
            <button
              className="connect-btn"
              onClick={connect}
              disabled={isConnecting}
            >
              {isConnecting ? 'Connecting...' : 'Connect Wallet'}
            </button>
          )}
        </div>
      </header>

      {/* Error Banner */}
      {(wsError || walletError) && (
        <div className="error-banner">
          {wsError || walletError}
        </div>
      )}

      {/* Market Selection Bar */}
      <div className="market-bar">
        <div className="market-list">
          {markets.map((market) => (
            <MarketCard
              key={market.id}
              market={market}
              selected={selectedMarket?.id === market.id}
              onSelect={setSelectedMarket}
            />
          ))}
          {markets.length === 0 && (
            <div className="no-markets">
              <p>No markets yet</p>
              <button onClick={handleCreateDemoMarket}>Create Demo Market</button>
            </div>
          )}
        </div>
      </div>

      {/* Selected Market Header */}
      {selectedMarket && (
        <div className="selected-market-header">
          <h2>{selectedMarket.question}</h2>
          <span className={`market-status ${selectedMarket.status}`}>
            {selectedMarket.status.toUpperCase()}
          </span>
        </div>
      )}

      {/* Main Content */}
      <main className="main-content">
        {/* Left: Order Book */}
        <section className="panel orderbook-panel">
          <OrderBook
            data={currentOrderbook}
            activeOutcome={activeOutcome}
            onOutcomeChange={setActiveOutcome}
            onPriceClick={handlePriceClick}
          />
        </section>

        {/* Center: Order Form */}
        <section className="panel order-form-panel">
          <OrderForm
            userId={address || ''}
            marketId={selectedMarket?.id || ''}
            selectedPrice={selectedPrice}
            onOrderPlaced={handleOrderPlaced}
          />
        </section>

        {/* Right: Trade History */}
        <section className="panel trade-history-panel">
          <TradeHistory trades={trades} />
        </section>
      </main>

      {/* Footer: Channel Log */}
      <footer className="footer">
        <ChannelLog />
      </footer>
    </div>
  );
}
