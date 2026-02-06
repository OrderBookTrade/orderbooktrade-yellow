const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// ==================== Types ====================

// Market types
export interface Market {
    id: string;
    question: string;
    description?: string;
    status: 'trading' | 'locked' | 'resolved';
    outcome?: 'YES' | 'NO';
    created_at: string;
    resolves_at: string;
    resolved_at?: string;
    creator_id: string;
}

export interface CreateMarketRequest {
    question: string;
    description?: string;
    resolves_at: string; // RFC3339
    creator_id: string;
}

// Order types
export interface PlaceOrderRequest {
    user_id: string;
    market_id: string;
    outcome_id: 'YES' | 'NO';
    side: 'buy' | 'sell';
    price: number; // 0-10000 basis points (probability)
    quantity: number;
}

export interface Order {
    id: string;
    user_id: string;
    market_id: string;
    outcome_id: 'YES' | 'NO';
    side: 'buy' | 'sell';
    price: number;
    quantity: number;
    filled_qty: number;
    status: 'open' | 'partial' | 'filled' | 'cancelled';
    timestamp: string;
}

export interface Trade {
    id: string;
    market_id: string;
    outcome_id: 'YES' | 'NO';
    buy_order_id: string;
    sell_order_id: string;
    buyer_id: string;
    seller_id: string;
    price: number;
    quantity: number;
    timestamp: string;
}

export interface PlaceOrderResponse {
    order: Order;
    trades: Trade[];
}

export interface OrderLevel {
    price: number;
    quantity: number;
    count: number;
}

export interface OrderbookSnapshot {
    bids: OrderLevel[];
    asks: OrderLevel[];
}

// Position types
export interface Position {
    user_id: string;
    market_id: string;
    yes_shares: number;
    no_shares: number;
}

export interface DepositRequest {
    user_id: string;
    amount: number; // In basis points (10000 = 1 USDC)
}

export interface MintSharesRequest {
    user_id: string;
    market_id: string;
    amount: number; // Number of share pairs
}

// Session types
export interface Allocation {
    participant: string;
    token: string;
    amount: string;
}

export interface CreateSessionRequest {
    participants: string[];
    allocations: Allocation[];
}

export interface CreateSessionResponse {
    channel_id: string;
    status: string;
}

export interface SettleRequest {
    channel_id: string;
    type: 'cooperative' | 'dispute';
}

export interface SettleResponse {
    status: string;
    channel_id: string;
    tx_hash?: string;
}

// ==================== API Client ====================

class ApiClient {
    private baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl;
    }

    private async request<T>(
        method: string,
        path: string,
        body?: unknown
    ): Promise<T> {
        const url = `${this.baseUrl}${path}`;

        const options: RequestInit = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };

        if (body) {
            options.body = JSON.stringify(body);
        }

        const response = await fetch(url, options);

        if (!response.ok) {
            const error = await response.json().catch(() => ({ error: 'Unknown error' }));
            throw new Error(error.error || `HTTP ${response.status}`);
        }

        return response.json();
    }

    // Health check
    async health(): Promise<{ status: string }> {
        return this.request('GET', '/api/health');
    }

    // ==================== Market Endpoints ====================

    async createMarket(req: CreateMarketRequest): Promise<Market> {
        return this.request('POST', '/api/market', req);
    }

    async listMarkets(): Promise<Market[]> {
        return this.request('GET', '/api/markets');
    }

    async getMarket(marketId: string): Promise<Market> {
        return this.request('GET', `/api/market/${marketId}`);
    }

    async resolveMarket(marketId: string, outcome: 'YES' | 'NO'): Promise<{
        market: Market;
        total_payout: number;
        positions: number;
    }> {
        return this.request('POST', `/api/market/${marketId}/resolve`, { outcome });
    }

    // ==================== Order Endpoints ====================

    async placeOrder(req: PlaceOrderRequest): Promise<PlaceOrderResponse> {
        return this.request('POST', '/api/order', req);
    }

    async getOrderbook(): Promise<OrderbookSnapshot> {
        return this.request('GET', '/api/orderbook');
    }

    async cancelOrder(orderId: string): Promise<{ status: string; order_id: string }> {
        return this.request('DELETE', `/api/order/${orderId}`);
    }

    async getTrades(): Promise<Trade[]> {
        return this.request('GET', '/api/trades');
    }

    // ==================== Position Endpoints ====================

    async deposit(req: DepositRequest): Promise<{ user_id: string; balance: number }> {
        return this.request('POST', '/api/deposit', req);
    }

    async mintShares(req: MintSharesRequest): Promise<{
        user_id: string;
        market_id: string;
        yes_shares: number;
        no_shares: number;
        balance: number;
    }> {
        return this.request('POST', '/api/mint', req);
    }

    async getPosition(userId: string, marketId?: string): Promise<{
        user_id: string;
        balance: number;
        position?: Position;
    }> {
        const query = marketId ? `?market_id=${marketId}` : '';
        return this.request('GET', `/api/position/${userId}${query}`);
    }

    // ==================== Session Endpoints ====================

    async createSession(req: CreateSessionRequest): Promise<CreateSessionResponse> {
        return this.request('POST', '/api/session', req);
    }

    async closeSession(channelId: string): Promise<{ status: string; channel_id: string }> {
        return this.request('DELETE', `/api/session/${channelId}`);
    }

    async settle(req: SettleRequest): Promise<SettleResponse> {
        return this.request('POST', '/api/settle', req);
    }
}

// Export singleton instance
export const api = new ApiClient(API_URL);

// Convenience functions
export const createMarket = (req: CreateMarketRequest) => api.createMarket(req);
export const listMarkets = () => api.listMarkets();
export const getMarket = (id: string) => api.getMarket(id);
export const resolveMarket = (id: string, outcome: 'YES' | 'NO') => api.resolveMarket(id, outcome);
export const placeOrder = (req: PlaceOrderRequest) => api.placeOrder(req);
export const getOrderbook = () => api.getOrderbook();
export const cancelOrder = (orderId: string) => api.cancelOrder(orderId);
export const getTrades = () => api.getTrades();
export const deposit = (req: DepositRequest) => api.deposit(req);
export const mintShares = (req: MintSharesRequest) => api.mintShares(req);
export const getPosition = (userId: string, marketId?: string) => api.getPosition(userId, marketId);
export const createSession = (req: CreateSessionRequest) => api.createSession(req);
export const closeSession = (channelId: string) => api.closeSession(channelId);
export const settle = (req: SettleRequest) => api.settle(req);
