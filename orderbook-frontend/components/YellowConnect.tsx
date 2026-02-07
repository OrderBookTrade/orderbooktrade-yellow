'use client';

import { useYellowAuth } from '@/hooks/useYellowAuth';

interface YellowConnectProps {
    walletAddress: string | null;
    walletConnected: boolean;
}

export function YellowConnect({ walletAddress, walletConnected }: YellowConnectProps) {
    const {
        isConnected,
        isAuthenticating,
        sessionKey,
        jwtToken,
        expiresAt,
        error,
        connect,
        disconnect,
    } = useYellowAuth(walletAddress);

    const formatAddress = (addr: string) => {
        return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
    };

    const formatExpiry = (timestamp: number) => {
        const date = new Date(timestamp * 1000);
        const now = new Date();
        const diff = date.getTime() - now.getTime();
        const minutes = Math.floor(diff / 60000);

        if (minutes < 0) return 'Expired';
        if (minutes < 60) return `${minutes}m`;
        const hours = Math.floor(minutes / 60);
        return `${hours}h ${minutes % 60}m`;
    };

    return (
        <div className="yellow-connect">
            {!walletConnected ? (
                <div className="yellow-status disabled">
                    <div className="status-icon">🟡</div>
                    <div className="status-text">
                        <div className="status-title">Yellow Network</div>
                        <div className="status-subtitle">Connect wallet first</div>
                    </div>
                </div>
            ) : !isConnected ? (
                <div className="yellow-status disconnected">
                    <div className="status-icon">⚪</div>
                    <div className="status-text">
                        <div className="status-title">Yellow Network</div>
                        <div className="status-subtitle">Not connected</div>
                    </div>
                    <button
                        className="yellow-connect-btn"
                        onClick={connect}
                        disabled={isAuthenticating}
                    >
                        {isAuthenticating ? 'Connecting...' : 'Connect Yellow'}
                    </button>
                </div>
            ) : (
                <div className="yellow-status connected">
                    <div className="status-icon">🟢</div>
                    <div className="status-text">
                        <div className="status-title">Yellow Network</div>
                        <div className="status-subtitle">
                            {sessionKey && `Session: ${formatAddress(sessionKey)}`}
                            {expiresAt && ` • Expires: ${formatExpiry(expiresAt)}`}
                        </div>
                    </div>
                    <button
                        className="yellow-disconnect-btn"
                        onClick={disconnect}
                    >
                        Disconnect
                    </button>
                </div>
            )}

            {error && (
                <div className="yellow-error">
                    ⚠️ {error}
                </div>
            )}

            {isAuthenticating && (
                <div className="yellow-progress">
                    <div className="progress-step">
                        <div className="spinner" />
                        <span>Waiting for MetaMask signature...</span>
                    </div>
                    <div className="progress-hint">
                        Please sign the EIP-712 message to authenticate with Yellow Network
                    </div>
                </div>
            )}

            <style jsx>{`
                .yellow-connect {
                    display: flex;
                    flex-direction: column;
                    gap: 8px;
                }

                .yellow-status {
                    display: flex;
                    align-items: center;
                    gap: 12px;
                    padding: 12px 16px;
                    background: rgba(255, 255, 255, 0.05);
                    border: 1px solid rgba(255, 255, 255, 0.1);
                    border-radius: 8px;
                }

                .yellow-status.connected {
                    background: rgba(0, 255, 0, 0.05);
                    border-color: rgba(0, 255, 0, 0.2);
                }

                .yellow-status.disconnected {
                    background: rgba(255, 255, 255, 0.02);
                }

                .yellow-status.disabled {
                    opacity: 0.5;
                }

                .status-icon {
                    font-size: 24px;
                    line-height: 1;
                }

                .status-text {
                    flex: 1;
                }

                .status-title {
                    font-weight: 600;
                    font-size: 14px;
                    color: rgba(255, 255, 255, 0.9);
                }

                .status-subtitle {
                    font-size: 12px;
                    color: rgba(255, 255, 255, 0.6);
                    margin-top: 2px;
                }

                .yellow-connect-btn,
                .yellow-disconnect-btn {
                    padding: 8px 16px;
                    border-radius: 6px;
                    font-size: 14px;
                    font-weight: 500;
                    cursor: pointer;
                    border: none;
                    transition: all 0.2s;
                }

                .yellow-connect-btn {
                    background: linear-gradient(135deg, #ffd700 0%, #ffed4e 100%);
                    color: #000;
                }

                .yellow-connect-btn:hover:not(:disabled) {
                    transform: translateY(-1px);
                    box-shadow: 0 4px 12px rgba(255, 215, 0, 0.3);
                }

                .yellow-connect-btn:disabled {
                    opacity: 0.6;
                    cursor: not-allowed;
                }

                .yellow-disconnect-btn {
                    background: rgba(255, 255, 255, 0.1);
                    color: rgba(255, 255, 255, 0.8);
                }

                .yellow-disconnect-btn:hover {
                    background: rgba(255, 255, 255, 0.15);
                }

                .yellow-error {
                    padding: 8px 12px;
                    background: rgba(255, 0, 0, 0.1);
                    border: 1px solid rgba(255, 0, 0, 0.3);
                    border-radius: 6px;
                    color: #ff6b6b;
                    font-size: 12px;
                }

                .yellow-progress {
                    padding: 12px;
                    background: rgba(255, 215, 0, 0.05);
                    border: 1px solid rgba(255, 215, 0, 0.2);
                    border-radius: 6px;
                }

                .progress-step {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    font-size: 13px;
                    color: rgba(255, 255, 255, 0.9);
                }

                .progress-hint {
                    margin-top: 6px;
                    font-size: 11px;
                    color: rgba(255, 255, 255, 0.6);
                }

                .spinner {
                    width: 16px;
                    height: 16px;
                    border: 2px solid rgba(255, 215, 0, 0.3);
                    border-top-color: #ffd700;
                    border-radius: 50%;
                    animation: spin 0.8s linear infinite;
                }

                @keyframes spin {
                    to {
                        transform: rotate(360deg);
                    }
                }
            `}</style>
        </div>
    );
}
