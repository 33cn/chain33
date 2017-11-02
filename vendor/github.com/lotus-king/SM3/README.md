An Implementation of SM3 by Golang
==================================

```
Note by Lotus-king:
I modify it so it's interface can exactly like SHA 
```
There is no Golang implementation of SM3 in Github. So I Write it when need it. And also I have wrote some unit test for it.

BUT one thing should be noticed: **The code is a little mess.**

If you want to quickly handle it. You'd better read the document first.

Official Documents of SM3
============================

You can download the official douments [Here](http://wenku.baidu.com/user/submit/download). 

There is a MISTAKE in the it.

In page 7, The padding message of the second test case is:

```
61626364 61626364 61626364 61626364 61626364 61626364 61626364 61626364 
61626364 61626364 61626364 61626364 61626364 61626364 61626364 61626364 
80000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 
80000000 00000000 00000000 00000000 00000000 00000000 00000000 00000200

```

But should be:

```
61626364 61626364 61626364 61626364 61626364 61626364 61626364 61626364 
61626364 61626364 61626364 61626364 61626364 61626364 61626364 61626364 
80000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 
00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000200
```

The difference is the first digit in the last line. It should be '0' not '8'.



Refrenced Projects
===========================
1. https://github.com/guanzhi/GmSSL/tree/master/crypto/sm3 
1. https://github.com/yixiangzhike/AlgorithmSM/blob/master/sm/SM3.py 
1. https://github.com/JamisHoo/Cryptographic-Algorithms/blob/master/src/sm3.cc



