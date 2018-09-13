pragma solidity ^0.4.0;

contract Test{

    function difficult() returns (uint){
        return block.difficulty;
    }

    function coinbase() returns (address){
        return block.coinbase;
    }

     function gaslimit() returns (uint){
        return block.gaslimit;
    }

      function Number() returns (uint){
        return block.number;
    }

     function Timestamp() returns (uint){
        return block.timestamp;
    }

     function gas() returns (uint){
        return msg.gas;
    }

     function origin() returns (address){
        return tx.origin;
    }

     function gasprice() returns (uint){
        return tx.gasprice;
    }

      function revert() returns(bytes){
        return revert();
    }

     function selfdestruct() returns(address){
        return selfdestruct();
    }

}


// contract blockhashTest{
//     function blockhash(uint blockNumber) returns (bytes32){
//         return block.blockhash();
//     }

// }

// contract Person{

//     bytes fail;

//     function(){
//         fail = msg.data;
//     }

//     function getFail() returns (bytes){
//         return fail;
//     }

// }


// contract CallTest{

//     function callData(address addr) returns (bool){
//         return addr.call("abc", 256);
//     }

// }
