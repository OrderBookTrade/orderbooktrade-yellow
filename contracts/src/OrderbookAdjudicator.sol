// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {IAdjudicator} from "./interfaces/IAdjudicator.sol";
import {Channel, State, Allocation} from "./interfaces/Types.sol";

/**
 * @title OrderbookAdjudicator
 * @notice Validates orderbook state transitions for Yellow Network state channels
 * @dev Implements IAdjudicator interface for the orderbook trading application
 */
contract OrderbookAdjudicator is IAdjudicator {
    error InvalidVersion();
    error AllocationsMismatch();
    error InvalidSignatureCount();

    /**
     * @notice Validates a candidate state based on orderbook rules
     * @dev Checks version progression and allocation conservation
     * @param chan The channel configuration
     * @param candidate The proposed state to be validated
     * @param proofs Array of previous states (typically the last valid state)
     * @return valid True if the candidate state is valid
     */
    function adjudicate(Channel calldata chan, State calldata candidate, State[] calldata proofs)
        external
        pure
        returns (bool valid)
    {
        // If there are proofs, verify version is strictly increasing
        if (proofs.length > 0) {
            State memory lastState = proofs[proofs.length - 1];
            if (candidate.version <= lastState.version) {
                return false;
            }

            // Verify allocation conservation (funds don't appear/disappear)
            if (!_checkAllocationConservation(lastState.allocations, candidate.allocations)) {
                return false;
            }
        }

        // Verify we have signatures from all participants
        if (candidate.sigs.length != chan.participants.length) {
            return false;
        }

        return true;
    }

    /**
     * @notice Check that total allocations are conserved between states
     * @dev Sum of all allocations must remain constant (per token)
     */
    function _checkAllocationConservation(Allocation[] memory oldAlloc, Allocation[] memory newAlloc)
        internal
        pure
        returns (bool)
    {
        if (oldAlloc.length != newAlloc.length) {
            return false;
        }

        // For simplicity, we assume same token across all allocations
        // and just check total amount conservation
        uint256 oldTotal = 0;
        uint256 newTotal = 0;

        for (uint256 i = 0; i < oldAlloc.length; i++) {
            oldTotal += oldAlloc[i].amount;
        }

        for (uint256 i = 0; i < newAlloc.length; i++) {
            newTotal += newAlloc[i].amount;
        }

        return oldTotal == newTotal;
    }
}
