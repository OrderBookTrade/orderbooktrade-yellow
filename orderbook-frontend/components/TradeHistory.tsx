'use client';

import { Trade } from '@/hooks/useWebSocket';

interface TradeHistoryProps {
    trades: Trade[];
}

export function TradeHistory({ trades }: TradeHistoryProps) {
    const formatPrice = (price: number) => (price / 100).toFixed(2);

    const formatTime = (timestamp: string) => {
        const date = new Date(timestamp);
        return date.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });
    };

    return (
        <div className="trade-history">
            <h2>Recent Trades</h2>

            {trades.length === 0 ? (
                <div className="empty-state">
                    <p>No trades yet</p>
                </div>
            ) : (
                <div className="trades-list">
                    <div className="trades-header">
                        <span>Price</span>
                        <span>Qty</span>
                        <span>Time</span>
                    </div>

                    {trades.map((trade, index) => (
                        <div
                            key={trade.id || index}
                            className="trade-row"
                        >
                            <span className="trade-price">
                                {formatPrice(trade.price)}¢
                            </span>
                            <span className="trade-quantity">
                                {trade.quantity}
                            </span>
                            <span className="trade-time">
                                {formatTime(trade.timestamp)}
                            </span>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
