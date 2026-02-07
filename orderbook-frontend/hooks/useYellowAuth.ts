'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { createPublicClient, createWalletClient, custom, http } from 'viem';
import { sepolia } from 'viem/chains';
import { generatePrivateKey, privateKeyToAccount } from 'viem/accounts';
import {
    createECDSAMessageSigner,
    createEIP712AuthMessageSigner,
    createAuthRequestMessage,
    createAuthVerifyMessageFromChallenge,
} from '@erc7824/nitrolite';

const YELLOW_WS_URL = process.env.NEXT_PUBLIC_YELLOW_WS_URL || 'wss://clearnet-sandbox.yellow.com/ws';
const APPLICATION_NAME = 'OrderbookTrade';

interface YellowAuthState {
    isConnected: boolean;
    isAuthenticating: boolean;
    sessionKey: string | null;
    jwtToken: string | null;
    expiresAt: number | null;
    error: string | null;
}

interface UseYellowAuthReturn extends YellowAuthState {
    connect: () => Promise<void>;
    disconnect: () => void;
}

export function useYellowAuth(walletAddress: string | null): UseYellowAuthReturn {
    const [state, setState] = useState<YellowAuthState>({
        isConnected: false,
        isAuthenticating: false,
        sessionKey: null,
        jwtToken: null,
        expiresAt: null,
        error: null,
    });

    const wsRef = useRef<WebSocket | null>(null);
    const sessionPrivateKeyRef = useRef<string | null>(null);

    // Disconnect when wallet changes
    useEffect(() => {
        if (!walletAddress && state.isConnected) {
            disconnect();
        }
    }, [walletAddress]);

    const connect = useCallback(async () => {
        if (!walletAddress) {
            setState(prev => ({ ...prev, error: 'Please connect your wallet first' }));
            return;
        }

        if (typeof window === 'undefined' || !window.ethereum) {
            setState(prev => ({ ...prev, error: 'MetaMask not found' }));
            return;
        }

        setState(prev => ({ ...prev, isAuthenticating: true, error: null }));

        try {
            console.log('[Yellow Auth] Starting authentication...');

            // Step 1: Generate session keypair
            const sessionPrivateKey = generatePrivateKey();
            sessionPrivateKeyRef.current = sessionPrivateKey;
            const sessionAccount = privateKeyToAccount(sessionPrivateKey);
            const sessionKey = sessionAccount.address;

            console.log('[Yellow Auth] Generated session key:', sessionKey);

            // Step 2: Create auth parameters
            const authParams = {
                session_key: sessionKey,
                allowances: [{
                    asset: 'ytest.usd',
                    amount: '1000000000' // Large allowance for testing
                }],
                expires_at: BigInt(Math.floor(Date.now() / 1000) + 3600), // 1 hour
                scope: 'orderbook.app',
            };

            // Step 3: Connect to Yellow WebSocket
            const ws = new WebSocket(YELLOW_WS_URL);
            wsRef.current = ws;

            await new Promise<void>((resolve, reject) => {
                const timeout = setTimeout(() => reject(new Error('Connection timeout')), 10000);

                ws.onopen = () => {
                    clearTimeout(timeout);
                    console.log('[Yellow Auth] WebSocket connected');
                    resolve();
                };

                ws.onerror = (error) => {
                    clearTimeout(timeout);
                    reject(new Error('WebSocket connection failed'));
                };
            });

            // Step 4: Send auth_request
            console.log('[Yellow Auth] Sending auth_request...');
            const sessionSigner = createECDSAMessageSigner(sessionPrivateKey);
            const authRequestMsg = await createAuthRequestMessage({
                address: walletAddress,
                application: APPLICATION_NAME,
                ...authParams
            });

            ws.send(authRequestMsg);

            // Step 5: Wait for auth_challenge
            const authResult = await new Promise<{
                sessionKey: string;
                jwtToken: string;
                expiresAt: number;
            }>((resolve, reject) => {
                const timeout = setTimeout(() => reject(new Error('Auth timeout')), 60000);

                ws.onmessage = async (event) => {
                    try {
                        const response = JSON.parse(event.data);
                        console.log('[Yellow Auth] Received:', response);

                        if (response.res && response.res[1] === 'auth_challenge') {
                            clearTimeout(timeout);
                            const challenge = response.res[2].challenge_message;
                            console.log('[Yellow Auth] Received challenge, requesting signature...');

                            // Step 6: Sign challenge with MetaMask (EIP-712)
                            const publicClient = createPublicClient({
                                chain: sepolia,
                                transport: http(),
                            });

                            const walletClient = createWalletClient({
                                chain: sepolia,
                                transport: custom(window.ethereum),
                                account: walletAddress as `0x${string}`,
                            });

                            const eip712Signer = createEIP712AuthMessageSigner(
                                walletClient,
                                authParams,
                                { name: APPLICATION_NAME }
                            );

                            console.log('[Yellow Auth] Signing with EIP-712...');
                            const verifyMsg = await createAuthVerifyMessageFromChallenge(
                                eip712Signer,
                                challenge
                            );

                            // Step 7: Send auth_verify
                            console.log('[Yellow Auth] Sending auth_verify...');
                            ws.send(verifyMsg);

                            // Wait for auth_verify response
                            const verifyTimeout = setTimeout(() => reject(new Error('Verify timeout')), 30000);

                            ws.onmessage = (event) => {
                                try {
                                    const response = JSON.parse(event.data);
                                    console.log('[Yellow Auth] Verify response:', response);

                                    if (response.res && response.res[1] === 'auth_verify') {
                                        clearTimeout(verifyTimeout);
                                        const result = response.res[2];
                                        resolve({
                                            sessionKey: result.session_key,
                                            jwtToken: result.jwt_token || '',
                                            expiresAt: result.expires_at,
                                        });
                                    } else if (response.error) {
                                        clearTimeout(verifyTimeout);
                                        reject(new Error(response.error.message || 'Auth verification failed'));
                                    }
                                } catch (err) {
                                    console.error('[Yellow Auth] Parse error:', err);
                                }
                            };
                        } else if (response.error) {
                            clearTimeout(timeout);
                            reject(new Error(response.error.message || 'Auth request failed'));
                        }
                    } catch (err) {
                        console.error('[Yellow Auth] Message handling error:', err);
                    }
                };

                ws.onerror = () => {
                    clearTimeout(timeout);
                    reject(new Error('WebSocket error during auth'));
                };
            });

            console.log('[Yellow Auth] ✓ Authentication successful!');
            console.log('[Yellow Auth] Session Key:', authResult.sessionKey);
            console.log('[Yellow Auth] JWT Token:', authResult.jwtToken ? authResult.jwtToken.slice(0, 20) + '...' : 'none');

            setState({
                isConnected: true,
                isAuthenticating: false,
                sessionKey: authResult.sessionKey,
                jwtToken: authResult.jwtToken,
                expiresAt: authResult.expiresAt,
                error: null,
            });

        } catch (err) {
            console.error('[Yellow Auth] Failed:', err);
            const errorMessage = err instanceof Error ? err.message : 'Authentication failed';
            setState(prev => ({
                ...prev,
                isAuthenticating: false,
                error: errorMessage,
            }));

            if (wsRef.current) {
                wsRef.current.close();
                wsRef.current = null;
            }
        }
    }, [walletAddress]);

    const disconnect = useCallback(() => {
        console.log('[Yellow Auth] Disconnecting...');

        if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
        }

        sessionPrivateKeyRef.current = null;

        setState({
            isConnected: false,
            isAuthenticating: false,
            sessionKey: null,
            jwtToken: null,
            expiresAt: null,
            error: null,
        });
    }, []);

    // Cleanup on unmount
    useEffect(() => {
        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, []);

    return {
        ...state,
        connect,
        disconnect,
    };
}
