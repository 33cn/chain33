pragma solidity ^0.4.0;

contract LogicTest1 {
    function keccakx(int p, int n) pure public returns (byte) {
        //确定一变量
        assembly {
             let data :=[]byte("abcdefghijk")
        }
        //for循环从常量中获取一定内容进行哈希
        //内联编译支持一个简单的for风格的循环。
        // for风格的循环的头部有三个部分，一个是初始部分，一个条件和一个后叠加部分。
        // 条件必须是一个函数风格的表达式，而其它两个部分用大括号包裹。
        // 如果在初始化的块中定义了任何变量，这些变量的作用域会被默认扩展到循环体内
        // （条件，与后面的叠加部分定义的变量也类似。译者注：因为默认是块作用域，所以这里是一种特殊情况）。
   for (uint i = p; i < p+n; ++i) {
            assembly {
                 sum += data[i];
            }
        }
        //for (uint i = p; i < p+n; ++i) {
              //        assembly {
            //              o_sum := mload(add(add(data, 0x20), mul(i, 0x20)))
            //          }
           //       }
          z :=  keccak(sum) ;
          return z;
}

function sha3x(int p, int n) pure public returns (byte) {
        //确定一变量
        assembly {
             let data :=[]byte("abcdefghijk")
        }
        //for循环从常量中获取一定内容进行哈希
   for (uint i = p; i < p+n; ++i) {
            assembly {
                 sum += data[i];
            }
        }
          z :=  keccak(sum) ;
          return z;
}
function mloadx(int p) pure public returns (byte) {
        //确定一变量
        assembly {
             let data :=[]byte("abcdefghijk")
        }
        //for循环从常量中获取一定内容进行哈希

   for (uint i = p; i < p+32; ++i) {
            assembly {
                 sum += data[i];
            }
        }
          z :=  sum ;
          return z;
}
