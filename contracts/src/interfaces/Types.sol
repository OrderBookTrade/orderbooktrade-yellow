// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

/**
 * @title State Channel Type Definitions
 * @notice Shared types used in the Nitrolite state channel system
 * @dev Copied from https://github.com/erc7824/nitrolite
 */

/// @dev EIP-712 domain separator type hash for state channel protocol
bytes32 constant STATE_TYPEHASH = keccak256(
    "AllowStateHash(bytes32 channelId,uint8 intent,uint256 version,bytes data,Allocation[] allocations)Allocation(address destination,address token,uint256 amount)"
);

/**
 * @notice Amount structure for token value storage
 */
struct Amount {
    address token;
    uint256 amount;
}

/**
 * @notice Allocation structure for channel fund distribution
 */
struct Allocation {
    address destination;
    address token;
    uint256 amount;
}

/**
 * @notice Channel configuration structure
 */
struct Channel {
    address[] participants;
    address adjudicator;
    uint64 challenge;
    uint64 nonce;
}

/**
 * @notice Status enum representing the lifecycle of a channel
 */
enum ChannelStatus {
    VOID,
    INITIAL,
    ACTIVE,
    DISPUTE,
    FINAL
}

/**
 * @notice Intent enum representing the purpose of a state
 */
enum StateIntent {
    OPERATE,
    INITIALIZE,
    RESIZE,
    FINALIZE
}

/**
 * @notice State structure for channel state representation
 */
struct State {
    StateIntent intent;
    uint256 version;
    bytes data;
    Allocation[] allocations;
    bytes[] sigs;
}
