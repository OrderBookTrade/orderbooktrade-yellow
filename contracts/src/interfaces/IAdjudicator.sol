// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {Channel, State} from "./Types.sol";

/**
 * @title Adjudicator Interface
 * @notice Interface for state validation and outcome determination
 * @dev Copied from https://github.com/erc7824/nitrolite
 */
interface IAdjudicator {
    /**
     * @notice Validates a candidate state based on application-specific rules
     * @param chan The channel configuration
     * @param candidate The proposed state to be validated
     * @param proofs Array of previous states for context
     * @return valid True if the candidate state is valid
     */
    function adjudicate(Channel calldata chan, State calldata candidate, State[] calldata proofs)
        external
        returns (bool valid);
}
