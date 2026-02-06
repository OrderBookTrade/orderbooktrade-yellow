'use client';

import { OrderbookData, OrderLevel } from '@/hooks/useWebSocket';

interface OrderBookProps {
    data: OrderbookData;
    onPriceClick?: (price: number) => void;
}

export function OrderBook({ data, onPriceClick }: OrderBookProps) {
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

    return (
        <div className="orderbook">
            <div className="orderbook-header">
                <h2>Order Book</h2>
                {spread > 0 && (
                    <span className="spread">Spread: {formatPrice(spread)}¢</span>
                )}
            </div>

            <div className="orderbook-content">
                {/* Asks (Sell orders) - displayed in reverse order */}
                <div className="orderbook-side asks">
                    <div className="orderbook-labels">
                        <span>Price</span>
                        <span>Qty</span>
                    </div>
                    {asks.slice().reverse().map((level, i) => (
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
                    <span className="spread-value">
                        {formatPrice(spread)}¢
                    </span>
                </div>

                {/* Bids (Buy orders) */}
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
            <span className="quantity">{level.quantity}</span>
        </div>
    );
}
