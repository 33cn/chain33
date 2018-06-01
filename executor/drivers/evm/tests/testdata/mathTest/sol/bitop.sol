pragma solidity ^0.4.0;

contract BitTest {
    function bitnot(int256 x) pure public returns (int256) {
        return ~x;
    }

    function bitand(int256 x, int256 y) pure public returns (int256) {
        return x&y;
    }

    function bitor(int256 x, int256 y) pure public returns (int256) {
        return x|y;
    }

    function bitxor(int256 x, int256 y) pure public returns (int256) {
       return x^y;
    }

    function bitnth(uint8 x) pure public returns (bytes1) {
        bytes memory bts = "8899aabbccddeef";
        return bts[x];
    }

    function bitshiftleft(int256 x, uint256 y) pure public returns (int256) {
        return x << y;
    }

    function bitshiftright(int256 x,uint256 y) pure public returns (int) {
        return x>>y;
    }

}
