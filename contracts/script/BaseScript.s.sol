// SPDX-License-Identifier: MIT
pragma solidity >=0.8.15;

import {console2, Script} from "forge-std/Script.sol";

contract BaseScript is Script {
    function setUp() public virtual {
        vm.createSelectFork("https://1rpc.io/sepolia");
        uint256 deployerPrivateKey = vm.envUint("PRI_KEY");
        vm.startBroadcast(deployerPrivateKey);
    }
}
