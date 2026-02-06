// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {Test, console} from "forge-std/Test.sol";
import {OrderbookAdjudicator} from "../src/OrderbookAdjudicator.sol";
import {Channel, State, Allocation, StateIntent} from "../src/interfaces/Types.sol";

contract OrderbookAdjudicatorTest is Test {
    OrderbookAdjudicator public adjudicator;

    address public alice = address(0x1);
    address public bob = address(0x2);
    address public token = address(0x3);

    function setUp() public {
        adjudicator = new OrderbookAdjudicator();
    }

    function _createChannel() internal view returns (Channel memory) {
        address[] memory participants = new address[](2);
        participants[0] = alice;
        participants[1] = bob;

        return Channel({
            participants: participants,
            adjudicator: address(adjudicator),
            challenge: 3600, // 1 hour
            nonce: 1
        });
    }

    function _createState(uint256 version, uint256 aliceAmount, uint256 bobAmount)
        internal
        view
        returns (State memory)
    {
        Allocation[] memory allocations = new Allocation[](2);
        allocations[0] = Allocation({destination: alice, token: token, amount: aliceAmount});
        allocations[1] = Allocation({destination: bob, token: token, amount: bobAmount});

        bytes[] memory sigs = new bytes[](2);
        sigs[0] = ""; // Placeholder signatures
        sigs[1] = "";

        return State({intent: StateIntent.OPERATE, version: version, data: "", allocations: allocations, sigs: sigs});
    }

    function test_AdjudicateValidState() public {
        Channel memory chan = _createChannel();

        // Initial state: Alice 100, Bob 100
        State memory initial = _createState(1, 100, 100);

        // New state after trade: Alice 80, Bob 120 (total conserved)
        State memory candidate = _createState(2, 80, 120);

        State[] memory proofs = new State[](1);
        proofs[0] = initial;

        bool valid = adjudicator.adjudicate(chan, candidate, proofs);
        assertTrue(valid, "Valid state should be accepted");
    }

    function test_AdjudicateInvalidVersion() public {
        Channel memory chan = _createChannel();

        State memory initial = _createState(2, 100, 100);
        State memory candidate = _createState(1, 80, 120); // Version going backwards

        State[] memory proofs = new State[](1);
        proofs[0] = initial;

        bool valid = adjudicator.adjudicate(chan, candidate, proofs);
        assertFalse(valid, "State with lower version should be rejected");
    }

    function test_AllocationConservation() public {
        Channel memory chan = _createChannel();

        State memory initial = _createState(1, 100, 100);
        State memory candidate = _createState(2, 100, 120); // Total increased (invalid)

        State[] memory proofs = new State[](1);
        proofs[0] = initial;

        bool valid = adjudicator.adjudicate(chan, candidate, proofs);
        assertFalse(valid, "Non-conserving allocation should be rejected");
    }

    function test_FirstStateNoProofs() public {
        Channel memory chan = _createChannel();
        State memory candidate = _createState(1, 100, 100);
        State[] memory proofs = new State[](0);

        bool valid = adjudicator.adjudicate(chan, candidate, proofs);
        assertTrue(valid, "First state with no proofs should be valid");
    }

    function test_InvalidSignatureCount() public {
        Channel memory chan = _createChannel();

        // Create state with only 1 signature (need 2 participants)
        Allocation[] memory allocations = new Allocation[](2);
        allocations[0] = Allocation({destination: alice, token: token, amount: 100});
        allocations[1] = Allocation({destination: bob, token: token, amount: 100});

        bytes[] memory sigs = new bytes[](1); // Only 1 sig, need 2
        sigs[0] = "";

        State memory candidate =
            State({intent: StateIntent.OPERATE, version: 1, data: "", allocations: allocations, sigs: sigs});

        State[] memory proofs = new State[](0);

        bool valid = adjudicator.adjudicate(chan, candidate, proofs);
        assertFalse(valid, "State with insufficient signatures should be rejected");
    }
}
