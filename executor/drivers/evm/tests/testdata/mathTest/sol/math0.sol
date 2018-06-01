pragma solidity ^0.4.0;

contract LogicTest {
    function add(int x, int y) pure public returns (int) {
        int z = x+y;
        return z;
    }

    function sub(int x, int y) pure public returns (int) {
        int z = x-y;
        return z;
    }

    function multi(int x, int y) pure public returns (int) {
        int z = x*y;
        return z;
    }

    function div(uint x, uint y) pure public returns (uint) {
        uint z = x/y;
        return z;
    }

    function sdiv(int x, int y) pure public returns (int) {
        int z = x/y;
        return z;
    }

    function mod(uint x, uint y) pure public returns (uint) {
        uint z = x%y;
        return z;
    }

    function smod(int x, int y) pure public returns (int) {
        int z = x%y;
        return z;
    }

}