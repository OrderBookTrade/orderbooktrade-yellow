'use client';

import { useState } from 'react';
import { OrderbookData, OrderLevel } from '@/hooks/useWebSocket';

type OutcomeTab = 'YES' | 'NO';

interface OrderBookProps {
    data: OrderbookData;
    onPriceClick?: (price: number) => void;
    onOutcomeChange?: (outcome: OutcomeTab) => void;
}

export function OrderBook({ data, onPriceClick, onOutcomeChange }: OrderBookProps) {
    const [activeOutcome, setActiveOutcome] = useState<OutcomeTab>('YES');
    const { bids, asks } = data;

    // Calculate max quantity for bar width scaling
    const allLevels = [...bids, ...asks];
    const maxQty = allLevels.length > 0
        ? Math.max(...allLevels.map(l => l.quantity))
        : 1;

    // Calculate spread
    const bestBid = bids.length > 0 ? bids[0].price : 0;
    const bestAsk = asks.length > 0 ? asks[0].price : 0;
    const spread = bestAsk > 0 && bestBid > 0 ? bestAsk - bestBid : 0;

    // Convert price from basis points to display (0-100 cents)
    const formatPrice = (price: number) => (price / 100).toFixed(2);

    // Get last traded price (for display)
    const lastPrice = bestBid > 0 ? formatPrice(bestBid) : '0.00';

    const handleTabChange = (outcome: OutcomeTab) => {
        setActiveOutcome(outcome);
        onOutcomeChange?.(outcome);
    };

    return (
        <div className="orderbook">
            <div className="orderbook-header">
                <h2>Order Book</h2>
                {spread > 0 && (
                    <span className="spread">Spread: {formatPrice(spread)}¢</span>
                )}
            </div>

            {/* Polymarket-style tabs */}
            <div className="orderbook-tabs">
                <button
                    className={`orderbook-tab ${activeOutcome === 'YES' ? 'active yes' : ''}`}
                    onClick={() => handleTabChange('YES')}
                >
                    Trade YES
                </button>
                <button
                    className={`orderbook-tab ${activeOutcome === 'NO' ? 'active no' : ''}`}
                    onClick={() => handleTabChange('NO')}
                >
                    Trade NO
                </button>
            </div>

            {/* Outcome indicator */}
            <div className="orderbook-outcome-label">
                <span className={`outcome-indicator ${activeOutcome.toLowerCase()}`}>
                    {activeOutcome}
                </span>
                <span className="last-price">Last: {lastPrice}¢</span>
                {spread > 0 && <span className="spread-info">Spread: {formatPrice(spread)}¢</span>}
            </div>

            <div className="orderbook-content">
                {/* Labels */}
                <div className="orderbook-labels">
                    <span>Price</span>
                    <span>Shares</span>
                    <span>Total</span>
                </div>

                {/* Asks (Sell orders) - stack from bottom up toward spread */}
                <div className="orderbook-side asks">
                    {asks.map((level, i) => (
                        <OrderBookRow
                            key={`ask-${level.price}-${i}`}
                            level={level}
                            side="ask"
                            maxQty={maxQty}
                            onClick={() => onPriceClick?.(level.price)}
                            formatPrice={formatPrice}
                        />
                    ))}
                </div>

                {/* Spread indicator */}
                <div className="spread-bar">
                    <span className="spread-badge asks">Asks</span>
                    <span className="spread-value">
                        {formatPrice(spread)}¢
                    </span>
                    <span className="spread-badge bids">Bids</span>
                </div>

                {/* Bids (Buy orders) - stack from top down from spread */}
                <div className="orderbook-side bids">
                    {bids.map((level, i) => (
                        <OrderBookRow
                            key={`bid-${level.price}-${i}`}
                            level={level}
                            side="bid"
                            maxQty={maxQty}
                            onClick={() => onPriceClick?.(level.price)}
                            formatPrice={formatPrice}
                        />
                    ))}
                </div>
            </div>
        </div>
    );
}

interface OrderBookRowProps {
    level: OrderLevel;
    side: 'bid' | 'ask';
    maxQty: number;
    onClick: () => void;
    formatPrice: (price: number) => string;
}

function OrderBookRow({ level, side, maxQty, onClick, formatPrice }: OrderBookRowProps) {
    const barWidth = (level.quantity / maxQty) * 100;
    const total = (level.price / 100) * level.quantity;

    return (
        <div
            className={`orderbook-row ${side}`}
            onClick={onClick}
        >
            <div
                className="quantity-bar"
                style={{ width: `${barWidth}%` }}
            />
            <span className="price">{formatPrice(level.price)}¢</span>
            <span className="quantity">{level.quantity.toLocaleString()}</span>
            <span className="total">${total.toFixed(2)}</span>
        </div>
    );
}
