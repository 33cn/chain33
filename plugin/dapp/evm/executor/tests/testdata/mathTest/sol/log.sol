pragma solidity ^0.4.0;

contract Transfer{
  event transfer(address indexed _from, address indexed _to, uint indexed value);

  function deposit() payable {
    address current = this;
    uint value = msg.value;
    transfer(msg.sender, current, value);
  }


  function getBanlance() constant returns(uint) {
      return this.balance;
  }

  /* fallback function */
  function(){}
}