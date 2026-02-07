'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { createPublicClient, createWalletClient, custom, http, type Address } from 'viem';
import { sepolia } from 'viem/chains';
import { generatePrivateKey, privateKeyToAccount } from 'viem/accounts';
import {
    createECDSAMessageSigner,
    createEIP712AuthMessageSigner,
    createAuthRequestMessage,
    createAuthVerifyMessageFromChallenge,
} from '@erc7824/nitrolite';

const YELLOW_WS_URL = process.env.NEXT_PUBLIC_YELLOW_WS_URL || 'wss://clearnet-sandbox.yellow.com/ws';
const APPLICATION_NAME = 'Test app'; // Must match Yellow Network's expected value

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
            const expiresAtTimestamp = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
            const authParams = {
                session_key: sessionKey,
                allowances: [{
                    asset: 'ytest.usd',
                    amount: '1000000000' // Large allowance for testing
                }],
                expires_at: BigInt(expiresAtTimestamp),
                scope: 'test.app', // Must match Yellow Network's expected value
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
                address: walletAddress as Address,
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

                            // Step 6: Sign challenge with MetaMask (EIP-712) using SDK helper
                            console.log('[Yellow Auth] Signing with EIP-712 using SDK...');

                            if (!window.ethereum) {
                                throw new Error('MetaMask not found');
                            }

                            // Request accounts from wallet provider
                            const accounts = await window.ethereum.request({
                                method: 'eth_requestAccounts'
                            }) as string[];

                            if (!accounts || accounts.length === 0) {
                                throw new Error('No accounts found in wallet');
                            }

                            const walletClient = createWalletClient({
                                account: accounts[0] as Address,
                                chain: sepolia,
                                transport: custom(window.ethereum)
                            });

                            // Create EIP-712 signer using SDK
                            // Note: The SDK helper handles the typed data construction and signing
                            const signer = createEIP712AuthMessageSigner(
                                walletClient,
                                authParams,
                                { name: APPLICATION_NAME }
                            );

                            console.log('[Yellow Auth] Requesting wallet signature and creating verify message...');

                            // Create verify message using SDK
                            // This internaly requests the signature and builds the verify message
                            const verifyMsg = await createAuthVerifyMessageFromChallenge(signer, challenge);

                            console.log('[Yellow Auth] Sending auth_verify message...');
                            ws.send(verifyMsg);

                            // Wait for auth_verify response
                            const verifyTimeout = setTimeout(() => reject(new Error('Verify timeout')), 30000);

                            ws.onmessage = (event) => {
                                try {
                                    const response = JSON.parse(event.data);
                                    console.log('[Yellow Auth] Verify response:', response);
                                    console.log('[Yellow Auth] Response type:', response.res ? response.res[1] : 'no res');
                                    console.log('[Yellow Auth] Response data:', response.res ? response.res[2] : 'no data');

                                    if (response.res && response.res[1] === 'auth_verify') {
                                        clearTimeout(verifyTimeout);
                                        const result = response.res[2];
                                        console.log('[Yellow Auth] Auth verify result:', result);

                                        // Try multiple possible field names
                                        const sessionKeyResult = result.session_key || result.sessionKey || sessionKey;
                                        const jwtTokenResult = result.jwt_token || result.jwtToken || result.token || '';
                                        const expiresAtResult = result.expires_at || result.expiresAt || (Math.floor(Date.now() / 1000) + 3600);

                                        console.log('[Yellow Auth] Extracted values:', {
                                            sessionKey: sessionKeyResult,
                                            jwtToken: jwtTokenResult ? jwtTokenResult.slice(0, 20) + '...' : 'none',
                                            expiresAt: expiresAtResult
                                        });

                                        resolve({
                                            sessionKey: sessionKeyResult,
                                            jwtToken: jwtTokenResult,
                                            expiresAt: expiresAtResult,
                                        });
                                    } else if (response.error) {
                                        clearTimeout(verifyTimeout);
                                        reject(new Error(response.error.message || 'Auth verification failed'));
                                    } else {
                                        console.log('[Yellow Auth] Received non-verify message, waiting...');
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
