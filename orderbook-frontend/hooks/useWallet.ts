'use client';

import { useState, useCallback, useEffect } from 'react';

declare global {
    interface Window {
        ethereum?: {
            isMetaMask?: boolean;
            request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
            on: (event: string, callback: (...args: unknown[]) => void) => void;
            removeListener: (event: string, callback: (...args: unknown[]) => void) => void;
        };
    }
}

interface UseWalletReturn {
    address: string | null;
    isConnected: boolean;
    isConnecting: boolean;
    error: string | null;
    connect: () => Promise<void>;
    disconnect: () => void;
    signMessage: (message: string) => Promise<string>;
}

export function useWallet(): UseWalletReturn {
    const [address, setAddress] = useState<string | null>(null);
    const [isConnecting, setIsConnecting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const isConnected = address !== null;

    // Check for existing connection on mount
    useEffect(() => {
        const checkConnection = async () => {
            if (typeof window === 'undefined' || !window.ethereum) {
                return;
            }

            try {
                const accounts = await window.ethereum.request({
                    method: 'eth_accounts'
                }) as string[];

                if (accounts.length > 0) {
                    setAddress(accounts[0]);
                }
            } catch (err) {
                console.error('Failed to check wallet connection:', err);
            }
        };

        checkConnection();
    }, []);

    // Listen for account changes
    useEffect(() => {
        if (typeof window === 'undefined' || !window.ethereum) {
            return;
        }

        const handleAccountsChanged = (accounts: unknown) => {
            const accountList = accounts as string[];
            if (accountList.length === 0) {
                setAddress(null);
            } else {
                setAddress(accountList[0]);
            }
        };

        window.ethereum.on('accountsChanged', handleAccountsChanged);

        return () => {
            window.ethereum?.removeListener('accountsChanged', handleAccountsChanged);
        };
    }, []);

    const connect = useCallback(async () => {
        if (typeof window === 'undefined' || !window.ethereum) {
            setError('MetaMask not installed');
            return;
        }

        setIsConnecting(true);
        setError(null);

        try {
            const accounts = await window.ethereum.request({
                method: 'eth_requestAccounts',
            }) as string[];

            if (accounts.length > 0) {
                setAddress(accounts[0]);
            }
        } catch (err) {
            const error = err as { code?: number; message?: string };
            if (error.code === 4001) {
                setError('Connection rejected by user');
            } else {
                setError('Failed to connect wallet');
            }
            console.error('Wallet connection error:', err);
        } finally {
            setIsConnecting(false);
        }
    }, []);

    const disconnect = useCallback(() => {
        setAddress(null);
        setError(null);
    }, []);

    const signMessage = useCallback(async (message: string): Promise<string> => {
        if (!address || typeof window === 'undefined' || !window.ethereum) {
            throw new Error('Wallet not connected');
        }

        try {
            const signature = await window.ethereum.request({
                method: 'personal_sign',
                params: [message, address],
            }) as string;

            return signature;
        } catch (err) {
            const error = err as { code?: number; message?: string };
            if (error.code === 4001) {
                throw new Error('Signature rejected by user');
            }
            throw new Error('Failed to sign message');
        }
    }, [address]);

    return {
        address,
        isConnected,
        isConnecting,
        error,
        connect,
        disconnect,
        signMessage,
    };
}
