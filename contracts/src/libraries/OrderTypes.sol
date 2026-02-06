// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

/**
 * @title Order Types
 * @notice Data structures for the orderbook trading system
 */
library OrderTypes {
    struct Order {
        bytes32 id;
        address trader;
        bool isBuy;
        uint256 price; // Price scaled by 1e18
        uint256 amount; // Quantity
        uint256 timestamp;
    }

    struct Trade {
        bytes32 buyOrderId;
        bytes32 sellOrderId;
        uint256 price;
        uint256 amount;
        uint256 timestamp;
    }

    struct OrderbookState {
        bytes32 stateHash; // Hash of the orderbook state
        uint256 tradeCount; // Number of trades executed
        uint256 lastTradeTime; // Timestamp of last trade
    }
}
