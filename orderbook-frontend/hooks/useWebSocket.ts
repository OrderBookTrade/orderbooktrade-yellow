'use client';

import { useState, useEffect, useRef, useCallback } from 'react';

export interface OrderLevel {
  price: number;
  quantity: number;
  count: number;
}

export interface OrderbookData {
  bids: OrderLevel[];
  asks: OrderLevel[];
}

// Dual orderbook structure for YES/NO outcomes
export interface DualOrderbookData {
  market_id: string;
  YES: OrderbookData;
  NO: OrderbookData;
}

export interface Trade {
  id: string;
  buy_order_id: string;
  sell_order_id: string;
  buyer_id: string;
  seller_id: string;
  price: number;
  quantity: number;
  timestamp: string;
  outcome_id?: string;
}

interface WebSocketMessage {
  type: 'orderbook' | 'trade' | 'connected';
  data: DualOrderbookData | Trade | { status: string };
}

interface UseWebSocketReturn {
  yesOrderbook: OrderbookData;
  noOrderbook: OrderbookData;
  trades: Trade[];
  connected: boolean;
  error: string | null;
}

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';
const RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;
const MAX_TRADES = 50;

const emptyOrderbook: OrderbookData = { bids: [], asks: [] };

export function useWebSocket(): UseWebSocketReturn {
  const [yesOrderbook, setYesOrderbook] = useState<OrderbookData>(emptyOrderbook);
  const [noOrderbook, setNoOrderbook] = useState<OrderbookData>(emptyOrderbook);
  const [trades, setTrades] = useState<Trade[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectDelayRef = useRef(RECONNECT_DELAY);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      const ws = new WebSocket(WS_URL);

      ws.onopen = () => {
        console.log('WebSocket connected');
        setConnected(true);
        setError(null);
        reconnectDelayRef.current = RECONNECT_DELAY;
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setConnected(false);

        // Schedule reconnection
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectDelayRef.current = Math.min(
            reconnectDelayRef.current * 2,
            MAX_RECONNECT_DELAY
          );
          connect();
        }, reconnectDelayRef.current);
      };

      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        setError('WebSocket connection failed');
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);

          switch (message.type) {
            case 'connected':
              console.log('WebSocket handshake complete');
              break;
            case 'orderbook':
              // Handle dual orderbook format
              const dualData = message.data as DualOrderbookData;
              if (dualData.YES) {
                setYesOrderbook({
                  bids: dualData.YES.bids || [],
                  asks: dualData.YES.asks || [],
                });
              }
              if (dualData.NO) {
                setNoOrderbook({
                  bids: dualData.NO.bids || [],
                  asks: dualData.NO.asks || [],
                });
              }
              break;
            case 'trade':
              setTrades(prev => {
                const newTrades = [message.data as Trade, ...prev];
                return newTrades.slice(0, MAX_TRADES);
              });
              break;
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      wsRef.current = ws;
    } catch (err) {
      console.error('Failed to create WebSocket:', err);
      setError('Failed to connect to server');
    }
  }, []);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  return { yesOrderbook, noOrderbook, trades, connected, error };
}
