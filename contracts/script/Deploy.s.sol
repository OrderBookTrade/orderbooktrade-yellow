// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {Script, console} from "forge-std/Script.sol";
import {OrderbookAdjudicator} from "../src/OrderbookAdjudicator.sol";
import {BaseScript} from "./BaseScript.s.sol";
import {
    State,
    Channel,
    Allocation,
    StateIntent
} from "../src/interfaces/Types.sol";

contract DeployScript is BaseScript {
    OrderbookAdjudicator public adjudicator =
        OrderbookAdjudicator(0x33eA68432d7657CA49Db36f378A95c6c71d3BDF1);

    address public alice = address(0x1);
    address public bob = address(0x2);
    address public token = address(0x3);

    function run() public {
        // 1.
        // deployContract();

        // 2. adjudicate
        adjudicate();
    }

    function deployContract() public {
        adjudicator = new OrderbookAdjudicator();

        console.log("OrderbookAdjudicator deployed at:", address(adjudicator));
    }

    // adjudicate
    function adjudicate() public {
        Channel memory channel = _createChannel();

        State memory initial = _createState(1, 100, 100);

        State memory candidate = _createState(2, 80, 120);

        State[] memory proofs = new State[](1);
        proofs[0] = initial;

        bool valid = adjudicator.adjudicate(channel, candidate, proofs);

        console.log("Valid state transition:", valid);
    }

    function _createState(
        uint256 version,
        uint256 aliceAmount,
        uint256 bobAmount
    ) internal view returns (State memory) {
        Allocation[] memory allocations = new Allocation[](2);
        allocations[0] = Allocation({
            destination: alice,
            token: token,
            amount: aliceAmount
        });
        allocations[1] = Allocation({
            destination: bob,
            token: token,
            amount: bobAmount
        });

        bytes[] memory sigs = new bytes[](2);
        sigs[0] = ""; // Placeholder signatures
        sigs[1] = "";

        return
            State({
                intent: StateIntent.OPERATE,
                version: version,
                data: "",
                allocations: allocations,
                sigs: sigs
            });
    }

    function _createChannel() internal view returns (Channel memory) {
        address[] memory participants = new address[](2);
        participants[0] = alice;
        participants[1] = bob;

        return
            Channel({
                participants: participants,
                adjudicator: address(adjudicator),
                challenge: 3600, // 1 hour
                nonce: 1
            });
    }
}
