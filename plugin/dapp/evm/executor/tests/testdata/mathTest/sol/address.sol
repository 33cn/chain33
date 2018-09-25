pragma solidity ^0.4.0;

contract LogicTest1 {
function returnContractAddress() constant returns (address) {
return this;
}
function getBalance(address addr) constant returns (uint){
return addr.balance;
}



}
pragma solidity ^0.4.0;

contract Person{

    bytes fail;

    function(){
        fail = msg.data;
    }

    function getFail() returns (bytes){
        return fail;
    }

}


contract CallTest{

    function callData(address addr) returns (bool){
        return addr.call("abc", 256);
    }

}


