pragma solidity ^0.4.0;

contract LogicTest1 {
    function addmodx(int x, int y, int m) pure public returns (int) {
        int z = (x + y) % m;
        return z;
    }

    function mulmod(int x, int y, int m) pure public returns (int) {
        int z = (x * y) % m;
        return z;
    }

}