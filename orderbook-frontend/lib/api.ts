const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Types
export interface PlaceOrderRequest {
    user_id: string;
    side: 'buy' | 'sell';
    price: number;
    quantity: number;
}

export interface Order {
    id: string;
    user_id: string;
    side: 'buy' | 'sell';
    price: number;
    quantity: number;
    filled_qty: number;
    status: 'open' | 'partial' | 'filled' | 'cancelled';
    timestamp: string;
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

// API Client
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

    // Order endpoints
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

    // Session endpoints
    async createSession(req: CreateSessionRequest): Promise<CreateSessionResponse> {
        return this.request('POST', '/api/session', req);
    }

    async closeSession(channelId: string): Promise<{ status: string; channel_id: string }> {
        return this.request('DELETE', `/api/session/${channelId}`);
    }

    // Settlement
    async settle(req: SettleRequest): Promise<SettleResponse> {
        return this.request('POST', '/api/settle', req);
    }
}

// Export singleton instance
export const api = new ApiClient(API_URL);

// Convenience functions
export const placeOrder = (req: PlaceOrderRequest) => api.placeOrder(req);
export const getOrderbook = () => api.getOrderbook();
export const cancelOrder = (orderId: string) => api.cancelOrder(orderId);
export const getTrades = () => api.getTrades();
export const createSession = (req: CreateSessionRequest) => api.createSession(req);
export const closeSession = (channelId: string) => api.closeSession(channelId);
export const settle = (req: SettleRequest) => api.settle(req);
