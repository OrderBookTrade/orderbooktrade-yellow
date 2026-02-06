'use client';

import { useState, FormEvent } from 'react';
import { placeOrder } from '@/lib/api';

interface OrderFormProps {
    userId: string;
    selectedPrice?: number;
    onOrderPlaced?: () => void;
}

export function OrderForm({ userId, selectedPrice, onOrderPlaced }: OrderFormProps) {
    const [side, setSide] = useState<'buy' | 'sell'>('buy');
    const [price, setPrice] = useState(selectedPrice?.toString() || '50');
    const [quantity, setQuantity] = useState('10');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    // Update price when selectedPrice changes
    if (selectedPrice !== undefined && price !== (selectedPrice / 100).toString()) {
        setPrice((selectedPrice / 100).toString());
    }

    // Prediction market constraint: YES + NO = 100
    const yesPrice = parseFloat(price) || 0;
    const noPrice = 100 - yesPrice;

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setError(null);
        setSuccess(null);
        setLoading(true);

        try {
            const priceValue = Math.round(parseFloat(price) * 100); // Convert to basis points
            const quantityValue = parseInt(quantity, 10);

            if (priceValue < 0 || priceValue > 10000) {
                throw new Error('Price must be between 0 and 100');
            }
            if (quantityValue <= 0) {
                throw new Error('Quantity must be positive');
            }

            const result = await placeOrder({
                user_id: userId,
                side,
                price: priceValue,
                quantity: quantityValue,
            });

            const fillInfo = result.trades.length > 0
                ? ` (${result.trades.length} trade${result.trades.length > 1 ? 's' : ''} filled)`
                : ' (pending)';

            setSuccess(`Order placed${fillInfo}`);
            onOrderPlaced?.();

            // Clear success message after 3 seconds
            setTimeout(() => setSuccess(null), 3000);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to place order');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="order-form">
            <h2>Place Order</h2>

            {/* Side Toggle */}
            <div className="side-toggle">
                <button
                    type="button"
                    className={`side-btn ${side === 'buy' ? 'active buy' : ''}`}
                    onClick={() => setSide('buy')}
                >
                    Buy YES
                </button>
                <button
                    type="button"
                    className={`side-btn ${side === 'sell' ? 'active sell' : ''}`}
                    onClick={() => setSide('sell')}
                >
                    Sell YES
                </button>
            </div>

            <form onSubmit={handleSubmit}>
                {/* Price Input */}
                <div className="form-group">
                    <label>
                        Price (¢)
                        <span className="hint">YES: {yesPrice.toFixed(1)}¢ | NO: {noPrice.toFixed(1)}¢</span>
                    </label>
                    <input
                        type="number"
                        value={price}
                        onChange={(e) => setPrice(e.target.value)}
                        min="0"
                        max="100"
                        step="0.1"
                        disabled={loading}
                    />
                    <input
                        type="range"
                        value={price}
                        onChange={(e) => setPrice(e.target.value)}
                        min="0"
                        max="100"
                        step="1"
                        className="price-slider"
                        disabled={loading}
                    />
                </div>

                {/* Quantity Input */}
                <div className="form-group">
                    <label>Quantity</label>
                    <input
                        type="number"
                        value={quantity}
                        onChange={(e) => setQuantity(e.target.value)}
                        min="1"
                        step="1"
                        disabled={loading}
                    />
                </div>

                {/* Order Summary */}
                <div className="order-summary">
                    <div className="summary-row">
                        <span>Total Cost</span>
                        <span>{((parseFloat(price) / 100) * parseInt(quantity || '0', 10)).toFixed(2)} USDC</span>
                    </div>
                    <div className="summary-row">
                        <span>Potential Profit</span>
                        <span className="profit">
                            {((1 - parseFloat(price) / 100) * parseInt(quantity || '0', 10)).toFixed(2)} USDC
                        </span>
                    </div>
                </div>

                {/* Submit Button */}
                <button
                    type="submit"
                    className={`submit-btn ${side}`}
                    disabled={loading || !userId}
                >
                    {loading ? 'Placing...' : `${side === 'buy' ? 'Buy' : 'Sell'} YES @ ${yesPrice.toFixed(1)}¢`}
                </button>

                {!userId && (
                    <p className="wallet-warning">Connect wallet to place orders</p>
                )}
            </form>

            {/* Status Messages */}
            {error && <div className="message error">{error}</div>}
            {success && <div className="message success">{success}</div>}
        </div>
    );
}
