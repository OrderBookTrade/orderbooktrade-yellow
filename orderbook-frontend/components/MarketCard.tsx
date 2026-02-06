'use client';

import { Market } from '@/lib/api';

interface MarketCardProps {
    market: Market;
    onSelect?: (market: Market) => void;
    selected?: boolean;
}

export function MarketCard({ market, onSelect, selected }: MarketCardProps) {
    const getStatusBadge = () => {
        switch (market.status) {
            case 'trading':
                return <span className="status-badge trading">🟢 Trading</span>;
            case 'locked':
                return <span className="status-badge locked">🔒 Locked</span>;
            case 'resolved':
                return (
                    <span className={`status-badge resolved ${market.outcome?.toLowerCase()}`}>
                        ✓ {market.outcome}
                    </span>
                );
        }
    };

    const formatDate = (dateStr: string) => {
        return new Date(dateStr).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        });
    };

    const timeUntilResolve = () => {
        const now = new Date();
        const resolves = new Date(market.resolves_at);
        const diff = resolves.getTime() - now.getTime();

        if (diff <= 0) return 'Past due';

        const hours = Math.floor(diff / (1000 * 60 * 60));
        const days = Math.floor(hours / 24);

        if (days > 0) return `${days}d ${hours % 24}h`;
        return `${hours}h`;
    };

    return (
        <div
            className={`market-card ${selected ? 'selected' : ''} ${market.status}`}
            onClick={() => onSelect?.(market)}
        >
            <div className="market-header">
                {getStatusBadge()}
                <span className="resolve-time">{timeUntilResolve()}</span>
            </div>

            <h3 className="market-question">{market.question}</h3>

            {market.description && (
                <p className="market-description">{market.description}</p>
            )}

            <div className="market-footer">
                <span className="resolve-date">
                    Resolves: {formatDate(market.resolves_at)}
                </span>
            </div>
        </div>
    );
}

// Add CSS styles to globals.css
