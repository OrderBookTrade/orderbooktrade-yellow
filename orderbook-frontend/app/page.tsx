'use client';

import { useState } from 'react';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useWallet } from '@/hooks/useWallet';
import { OrderBook } from '@/components/OrderBook';
import { OrderForm } from '@/components/OrderForm';
import { TradeHistory } from '@/components/TradeHistory';
import { ChannelLog, addLogEntry } from '@/components/ChannelLog';

export default function Home() {
  const { orderbook, trades, connected, error: wsError } = useWebSocket();
  const { address, isConnected, isConnecting, connect, error: walletError } = useWallet();
  const [selectedPrice, setSelectedPrice] = useState<number | undefined>();

  // Log connection events
  if (connected && !wsError) {
    // Only log once
  }

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
          <span className="logo-icon">📈</span>
          <h1>OrderBook.Trade</h1>
          <span className="network-badge">Yellow Network</span>
        </div>

        <div className="header-right">
          {/* Connection Status */}
          <div className={`connection-status ${connected ? 'connected' : 'disconnected'}`}>
            <span className="status-dot" />
            {connected ? 'Live' : 'Connecting...'}
          </div>

          {/* Wallet */}
          {isConnected ? (
            <div className="wallet-info">
              <span className="wallet-address">{formatAddress(address!)}</span>
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

      {/* Main Content */}
      <main className="main-content">
        {/* Left: Order Book */}
        <section className="panel orderbook-panel">
          <OrderBook
            data={orderbook}
            onPriceClick={handlePriceClick}
          />
        </section>

        {/* Center: Order Form */}
        <section className="panel order-form-panel">
          <OrderForm
            userId={address || ''}
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
