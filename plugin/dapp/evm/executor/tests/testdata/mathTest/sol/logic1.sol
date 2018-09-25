pragma solidity ^0.4.0;

contract LogicTest1 {
    function exp(uint256 x, uint256 y) pure public returns (uint256) {
        uint z = x**y;
        return z;
    }

    function less(uint256 x, uint256 y) pure public returns (uint256) {
        if(x < y)
            return 1;
        return 0;
    }

    function more(uint256 x, uint256 y) pure public returns (uint256) {
        if(x > y)
            return 1;
        return 0;
    }

    function sless(int256 x, int256 y) pure public returns (int256) {
        if(x < y)
            return 1;
        return y-x;
    }

    function smore(int256 x, int256 y) pure public returns (int256) {
        if(x > y)
            return 1;
        return x-y;
    }

    function equals(int256 x, int256 y) pure public returns (int256) {
        if(x == y)
            return 1;
        return 0;
    }

    function iszero(int256 x) pure public returns (int) {
        if(x == 0)
            return 1;
        return 0;
    }

}