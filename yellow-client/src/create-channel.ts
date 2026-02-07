import {
    NitroliteClient,
    WalletStateSigner,
    createTransferMessage,
    createGetConfigMessage,
    createECDSAMessageSigner,
    createEIP712AuthMessageSigner,
    createAuthVerifyMessageFromChallenge,
    createCreateChannelMessage,
    createResizeChannelMessage,
    createGetLedgerBalancesMessage,
    createAuthRequestMessage,
    createCloseChannelMessage
} from '@erc7824/nitrolite';
import type {
    RPCNetworkInfo,
    RPCAsset,
    RPCData
} from '@erc7824/nitrolite';
import { createPublicClient, createWalletClient, http } from 'viem';
import { sepolia } from 'viem/chains';
import { privateKeyToAccount, generatePrivateKey } from 'viem/accounts';
import WebSocket from 'ws';
import 'dotenv/config';
import * as readline from 'readline';

//1 . script start
console.log('Starting script...');

const askQuestion = (query: string): Promise<string> => {
    const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
    });
    return new Promise(resolve => rl.question(query, ans => {
        rl.close();
        resolve(ans);
    }));
};

let PRIVATE_KEY = process.env.PRIVATE_KEY as `0x${string}`;

if (!PRIVATE_KEY) {
    console.log('PRIVATE_KEY not found in .env');
    const inputKey = await askQuestion('Please enter your Private Key: ');
    if (!inputKey) {
        throw new Error('Private Key is required');
    }
    PRIVATE_KEY = inputKey.startsWith('0x') ? inputKey as `0x${string}` : `0x${inputKey}` as `0x${string}`;
}

// 2. create account
const account = privateKeyToAccount(PRIVATE_KEY);

console.log('Account:', account.address);

const ALCHEMY_RPC_URL = process.env.ALCHEMY_RPC_URL;
const FALLBACK_RPC_URL = 'https://1rpc.io/sepolia';


// 3. create publicClient and walletClient
const publicClient = createPublicClient({
    chain: sepolia,
    transport: http(ALCHEMY_RPC_URL || FALLBACK_RPC_URL),
});

const walletClient = createWalletClient({
    chain: sepolia,
    transport: http(),
    account,
});

interface Config {
    assets?: RPCAsset[];
    networks?: RPCNetworkInfo[];
    [key: string]: any;
}


async function fetchConfig(): Promise<Config> {
    const signer = createECDSAMessageSigner(PRIVATE_KEY);
    const message = await createGetConfigMessage(signer);

    const ws = new WebSocket('wss://clearnet-sandbox.yellow.com/ws');

    return new Promise((resolve, reject) => {
        ws.onopen = () => {
            ws.send(message);
        };

        ws.onmessage = (event) => {
            try {
                const response = JSON.parse(event.data.toString());
                if (response.res && response.res[2]) {
                    resolve(response.res[2] as Config);
                    ws.close();
                } else if (response.error) {
                    reject(new Error(response.error.message || 'Unknown RPC error'));
                    ws.close();
                }
            } catch (err) {
                reject(err);
                ws.close();
            }
        };

        ws.onerror = (error) => {
            reject(error);
            ws.close();
        };
    });

}
const ws = new WebSocket('wss://clearnet-sandbox.yellow.com/ws');

// 4. fetch config from yellow node
const config = await fetchConfig();
console.log("config assets length", config.assets?.length)


// 5. create nitrolite client
const client = new NitroliteClient({
    publicClient,
    walletClient,
    // Use WalletStateSigner for signing states
    stateSigner: new WalletStateSigner(walletClient),
    // Contract addresses
    addresses: {
        custody: '0x019B65A265EB3363822f2752141b3dF16131b262',
        adjudicator: '0x7c7ccbc98469190849BCC6c926307794fDfB11F2',
    },
    chainId: sepolia.id,
    challengeDuration: 3600n, // 1 hour challenge period
});


// 6. create session account
const sessionPrivateKey = generatePrivateKey();
const sessionAccount = privateKeyToAccount(sessionPrivateKey);
const sessionAddress = sessionAccount.address;

// Helper: Create a signer for the session key
const sessionSigner = createECDSAMessageSigner(sessionPrivateKey);

// 7.create auth request params
const authParams = {
    session_key: sessionAddress,        // Session key you generated
    allowances: [{                      // Add allowance for ytest.usd
        asset: 'ytest.usd',
        amount: '1000000000'            // Large amount
    }],
    expires_at: BigInt(Math.floor(Date.now() / 1000) + 3600), // 1 hour in seconds
    scope: 'test.app',
};

const authRequestMsg = await createAuthRequestMessage({
    address: account.address,           // Your main wallet address
    application: 'Test app',            // Match domain name
    ...authParams
});

let activeChannelId: string | undefined;


// 9. create trigger resize message

const triggerResize = async (channelId: string, token: string, skipResize: boolean = false) => {
    console.log('  Using existing channel:', channelId);

    const amountToFund = 20n;
    if (!skipResize) console.log('\nRequesting resize to fund channel with 20 tokens...');

    if (!skipResize) {
        const resizeMsg = await createResizeChannelMessage(
            sessionSigner,
            {
                channel_id: channelId as `0x${string}`,
                // resize_amount: 10n, // <-- This requires L1 funds in Custody (which we don't have)
                allocate_amount: amountToFund,  // <-- This pulls from Unified Balance (Faucet) (Variable name adjusted)
                funds_destination: account.address,
            }
        );

        ws.send(resizeMsg);
        console.log('  Waiting for resize confirmation...');
    }

}

// 8. listen message and handle
ws.onmessage = async (event) => {
    // 8.1 parse message
    const response = JSON.parse(event.data.toString());
    console.log('Received WS message:', JSON.stringify(response, null, 2));

    if (response.error) {
        console.error('RPC Error:', response.error);
        process.exit(1); // Exit on error to prevent infinite loops
    }

    // 8.2 handle auth_challenge message
    if (response.res && response.res[1] === 'auth_challenge') {
        const challenge = response.res[2].challenge_message;

        const signer = createEIP712AuthMessageSigner(
            walletClient,
            authParams,
            { name: 'Test app' }
        );

        const verifyMsg = await createAuthVerifyMessageFromChallenge(
            signer,
            challenge
        );

        // auth_challenge ，auth_verify
        ws.send(verifyMsg);
    }


    // 8.3 handle auth_verify message
    if (response.res && response.res[1] === 'auth_verify') {
        console.log('Auth verified successfully!');
        const sessionKey = response.res[2].session_key;
        console.log('  Session key:', sessionKey);
        console.log('  JWT token received');

        // Query Ledger Balances
        const ledgerMsg = await createGetLedgerBalancesMessage(
            sessionSigner,
            account.address,
            Date.now()
        );
        ws.send(ledgerMsg);
        
        console.log('  Sent get_ledger_balances request...');
    }


    // 8.4 get chennels
    if (response.res && response.res[1] === 'channels') {
        const channels = response.res[2].channels;
        console.log('channels', channels);
        const openChannel = channels.find((c: any) => c.status === 'open');

        const chainId = sepolia.id;
        const supportedAsset = (config.assets as any)?.find((a: any) => a.chain_id === chainId);
        const token = supportedAsset ? supportedAsset.token : '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238';


        console.log("openChannel", openChannel)
        const l1Channels = await client.getOpenChannels();
        console.log("l1Channels", l1Channels)

        if (openChannel) {
            console.log('✓ Found existing open channel');
            const currentAmount = BigInt(openChannel.amount || 0);
            console.log("open channel amount", openChannel.amount)

            if (BigInt(openChannel.amount) >= 20n) {
                console.log(`  Channel already funded with ${openChannel.amount} USDC.`);
                console.log('  Skipping resize to avoid "Insufficient Balance" errors.');
                await triggerResize(openChannel.channel_id, token, true);
            } else {
                await triggerResize(openChannel.channel_id, token, false);
            }

            console.log("start to resize and trigger ")
        } else {
            console.log('  No existing open channel found on Node.');

            // Check L1 directly to see if it's just a sync delay
            console.log('  Checking L1 directly for comparison...');

            console.log('  L1 open channels:', l1Channels);

            if (l1Channels.length > 0) {
                console.log('  ⚠️ Sync Delay Detected: Channel exists on L1 but Node hasn\'t seen it yet.');
                console.log('  You might want to wait 15-30 seconds or use the first L1 channel ID for manual trigger.');
                // Optional: If we want to be aggressive, we could use the L1 ID, 
                // but usually better to wait for Node to index it.
            } else {
                console.log('  Creating new channel since not found on L1 or Node...');
                console.log('  Using token:', token, 'for chain:', chainId);
                const createChannelMsg = await createCreateChannelMessage(
                    sessionSigner,
                    {
                        chain_id: 11155111, // Sepolia
                        token: token,
                    }
                );
                ws.send(createChannelMsg);
            }
        }


    }

    // 8.5 handle create_channel message
    if (response.res && response.res[1] === 'create_channel') {
        const { channel_id, channel, state, server_signature } = response.res[2];
        activeChannelId = channel_id;

        console.log('✓ Channel prepared:', channel_id);
        console.log('  State object:', JSON.stringify(state, null, 2));

        const unsignedInitialState = {
            intent: state.intent,
            version: BigInt(state.version),
            data: state.state_data, // Map state_data to data
            allocations: state.allocations.map((a: any) => ({
                destination: a.destination,
                token: a.token,
                amount: BigInt(a.amount),
            })),
        };

        // Submit to blockchain
        const createResult = await client.createChannel({
            channel,
            unsignedInitialState,
            serverSignature: server_signature,
        });

        const txHash = typeof createResult === 'string' ? createResult : createResult.txHash;

        console.log('✓ Channel created on-chain:', txHash);
        console.log('  Waiting for transaction confirmation...');
        await publicClient.waitForTransactionReceipt({ hash: txHash });
        console.log('✓ Transaction confirmed');


    }


    // 8.6 resize_channel 
    if (response.res && response.res[1] === 'resize_channel') {
        const { channel_id, state, server_signature } = response.res[2];
        console.log('✓ Resize prepared');
        console.log('  Server returned allocations:', JSON.stringify(state.allocations, null, 2));

        const resizeState = {
            intent: state.intent,
            version: BigInt(state.version),
            data: state.state_data || state.data, // Handle potential naming differences
            allocations: state.allocations.map((a: any) => ({
                destination: a.destination,
                token: a.token,
                amount: BigInt(a.amount),
            })),
            channelId: channel_id,
            serverSignature: server_signature,
        };

        console.log('DEBUG: resizeState:', JSON.stringify(resizeState, (key, value) =>
            typeof value === 'bigint' ? value.toString() : value, 2));


        let proofStates: any[] = [];
        try {
            const onChainData = await client.getChannelData(channel_id as `0x${string}`);
            console.log('DEBUG: On-chain channel data:', JSON.stringify(onChainData, (key, value) =>
                typeof value === 'bigint' ? value.toString() : value, 2));
            if (onChainData.lastValidState) {
                proofStates = [onChainData.lastValidState];
            }
        } catch (e) {
            console.log('DEBUG: Failed to fetch on-chain data:', e);
        }

        console.log("++++++++++++++++++++++ resize ")
    }

}


// script entry point
async function waitForOpen(ws: WebSocket): Promise<void> {
    return new Promise((resolve) => {
        if (ws.readyState === WebSocket.OPEN) {
            ws.send(authRequestMsg);
            resolve();
        } else {
            ws.on('open', () => {
                ws.send(authRequestMsg);
                resolve();
            });
        }
    });
}

await waitForOpen(ws);

