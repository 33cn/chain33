package secp256k1


/*
Key Exchange:
<sipa> HaltingState: you need the multiplicative tweak
<sipa> there's no "ECDH" as such in the library, but the required operations are there
<sipa> it's trivial
<sipa> both compute a private key, compute the corresponding public key, send the public keys to eachother, and then tweak the other's public key with their own private key
<sipa> the result is the shared secret
<kjj> the shared secret is (your private key) * (their public key), which is equal to G * (your privkey) * (their privkey)
*/

/*



<sipa> HaltingState: you have a and B, i have A and b
* airq (~airq@31-12.192-178.cust.bluewin.ch) has joined #bitcoin-dev
<sipa> where A = a*G and B = b*G
<sipa> so both can compute a*B = b*A = a*b*G
<HaltingState> but what if you dont have my pubkey
<sipa> i always do
<HaltingState> i have to send my pubkey with the message
<sipa> you can't do ECDH otherwise
<sipa> yes?
<sipa> so in that case, one of the keys is the actual receiver's pubkey
<sipa> the other is an ephemeral key
<sipa> created by the sender, just for this message
<HaltingState> so my pubkey is in plaintext, so i should generate new random pubkey for first part until session key is setup for symmetric encryption
<sipa> so i want to send an encrypted message to you
<sipa> i have you public key P = p*G
<sipa> i generate an ephemeral key E = e*G
<sipa> and compute H(e*P), and send (E,enc(key=H(e*P),msg))
<sipa> the receiver than has p and E, so can compute p*E = p*e*G = e*p*G = e*P, and the hash thereof
<sipa> and can thus decrypt the message
*/


/*
<sipa> where A = a*G and B = b*G
<sipa> so both can compute a*B = b*A = a*b*G
<HaltingState> but what if you dont have my pubkey
<sipa> i always do
* Coincidental has quit (Remote host closed the connection)
<HaltingState> i have to send my pubkey with the message
<sipa> you can't do ECDH otherwise
* NLNico has quit (Quit: Leaving)
<sipa> yes?
<sipa> so in that case, one of the keys is the actual receiver's pubkey
<sipa> the other is an ephemeral key
<sipa> created by the sender, just for this message
<HaltingState> so my pubkey is in plaintext, so i should generate new random pubkey for first part until session key is setup for symmetric encryption
<sipa> so i want to send an encrypted message to you
<sipa> i have you public key P = p*G
*/

/*
<HaltingState> sipa, int secp256k1_ecdsa_pubkey_create(unsigned char *pubkey, int *pubkeylen, const unsigned char *seckey, int compressed);
<HaltingState> is that how i generate private/public keys?
<sipa> HaltingState: you pass in a random 32-byte string as seckey
<sipa> HaltingState: if it is valid, the corresponding pubkey is put in pubkey
<sipa> and true is returned
<sipa> otherwise, false is returned
<sipa> around 1 in 2^128 32-byte strings are invalid, so the odds of even ever seeing one is extremely rare

<sipa> private keys are mathematically numbers
<sipa> each has a corresponding point on the curve as public key
<sipa> a private key is just a number
<sipa> a public key is a point with x/y coordinates
<sipa> almost every 256-bit number is a valid private key (one with a point on the curve corresponding to it)
<sipa> HaltingState: ok?

<sipa> more than half of random points are not on the curve
<sipa> and actually, it is less than  the square root, not less than half, sorry :)
!!!
<sipa> a private key is a NUMBER
<sipa> a public key is a POINT
<gmaxwell> half the x,y values in the field are not on the curve, a private key is an integer.

<sipa> HaltingState: yes, n,q = private keys; N,Q = corresponding public keys (N=n*G, Q=q*G); then it follows that n*Q = n*q*G = q*n*G = q*N
<sipa> that's the reason ECDH works
<sipa> multiplication is associative and commutative
*/

/*
<HaltingState> sipa, ok; i am doing compact signatures and I want to know; can someone change the signature to get another valid signature for same message without the private key
<HaltingState> because i know they can do that for the normal 72 byte signatures that openssl was putting out
<sipa> HaltingState: if you don't enforce non-malleability, yes
<sipa> HaltingState: if you force the highest bit of t

<sipa> it _creates_ signatures that already satisfy that condition
<sipa> but it will accept ones that don't
<sipa> maybe i should change that, and be strict
<HaltingState> yes; i want some way to know signature is valid but fails malleability
<sipa> well if the highest bit of S is 1, you can take its complement
<sipa> and end up with a valid signature
<sipa> that is canonical
*/

/*

<HaltingState> sipa, I am signing messages and highest bit of the compact signature is 1!!!
<HaltingState>  if (b & 0x80) == 0x80 {
<HaltingState>   log.Panic("b= %v b2= %v \n", b, b&0x80)
<HaltingState>  }
<sipa> what bit?
* Pengoo has quit (Ping timeout: 272 seconds)
<HaltingState> the highest bit of the first byte of signature
<sipa> it's the highest bit of S
<sipa> so the 32nd byte
<HaltingState> wtf

*/

/*
 For instance, nonces are used in HTTP digest access authentication to calculate an MD5 digest
 of the password. The nonces are different each time the 401 authentication challenge
 response code is presented, thus making replay attacks virtually impossible.

can verify client/server match without sending password over network
*/

/*
<hanihani> one thing I dont get about armory for instance,
is how the hot-wallet can generate new addresses without
knowing the master key
*/

/*
<HaltingState> i am yelling at the telehash people for using secp256r1
instead of secp256k1; they thing r1 is "more secure" despite fact that
there is no implementation that works and wrapping it is now taking
up massive time, lol
<gmaxwell> ...

<gmaxwell> You know that the *r curves are selected via an undisclosed
secret process, right?
<gmaxwell> HaltingState: telehash is offtopic for this channel.
*/
/*
 For instance, nonces are used in HTTP digest access authentication to calculate an MD5 digest
 of the password. The nonces are different each time the 401 authentication challenge
 response code is presented, thus making replay attacks virtually impossible.

can verify client/server match without sending password over network
*/

/*
void secp256k1_start(void);
void secp256k1_stop(void);

 * Verify an ECDSA signature.
 *  Returns: 1: correct signature
 *           0: incorrect signature
 *          -1: invalid public key
 *          -2: invalid signature
 *
int secp256k1_ecdsa_verify(const unsigned char *msg, int msglen,
                           const unsigned char *sig, int siglen,
                           const unsigned char *pubkey, int pubkeylen);

http://www.nilsschneider.net/2013/01/28/recovering-bitcoin-private-keys.html

Why did this work? ECDSA requires a random number for each signature. If this random
number is ever used twice with the same private key it can be recovered.
This transaction was generated by a hardware bitcoin wallet using a pseudo-random number
generator that was returning the same “random” number every time.

Nonce is 32 bytes?

 * Create an ECDSA signature.
 *  Returns: 1: signature created
 *           0: nonce invalid, try another one
 *  In:      msg:    the message being signed
 *           msglen: the length of the message being signed
 *           seckey: pointer to a 32-byte secret key (assumed to be valid)
 *           nonce:  pointer to a 32-byte nonce (generated with a cryptographic PRNG)
 *  Out:     sig:    pointer to a 72-byte array where the signature will be placed.
 *           siglen: pointer to an int, which will be updated to the signature length (<=72).
 *
int secp256k1_ecdsa_sign(const unsigned char *msg, int msglen,
                         unsigned char *sig, int *siglen,
                         const unsigned char *seckey,
                         const unsigned char *nonce);


 * Create a compact ECDSA signature (64 byte + recovery id).
 *  Returns: 1: signature created
 *           0: nonce invalid, try another one
 *  In:      msg:    the message being signed
 *           msglen: the length of the message being signed
 *           seckey: pointer to a 32-byte secret key (assumed to be valid)
 *           nonce:  pointer to a 32-byte nonce (generated with a cryptographic PRNG)
 *  Out:     sig:    pointer to a 64-byte array where the signature will be placed.
 *           recid:  pointer to an int, which will be updated to contain the recovery id.
 *
int secp256k1_ecdsa_sign_compact(const unsigned char *msg, int msglen,
                                 unsigned char *sig64,
                                 const unsigned char *seckey,
                                 const unsigned char *nonce,
                                 int *recid);

 * Recover an ECDSA public key from a compact signature.
 *  Returns: 1: public key succesfully recovered (which guarantees a correct signature).
 *           0: otherwise.
 *  In:      msg:        the message assumed to be signed
 *           msglen:     the length of the message
 *           compressed: whether to recover a compressed or uncompressed pubkey
 *           recid:      the recovery id (as returned by ecdsa_sign_compact)
 *  Out:     pubkey:     pointer to a 33 or 65 byte array to put the pubkey.
 *           pubkeylen:  pointer to an int that will contain the pubkey length.
 *

recovery id is between 0 and 3

int secp256k1_ecdsa_recover_compact(const unsigned char *msg, int msglen,
                                    const unsigned char *sig64,
                                    unsigned char *pubkey, int *pubkeylen,
                                    int compressed, int recid);


 * Verify an ECDSA secret key.
 *  Returns: 1: secret key is valid
 *           0: secret key is invalid
 *  In:      seckey: pointer to a 32-byte secret key
 *
int secp256k1_ecdsa_seckey_verify(const unsigned char *seckey);

** Just validate a public key.
 *  Returns: 1: valid public key
 *           0: invalid public key
 *
int secp256k1_ecdsa_pubkey_verify(const unsigned char *pubkey, int pubkeylen);

** Compute the public key for a secret key.
 *  In:     compressed: whether the computed public key should be compressed
 *          seckey:     pointer to a 32-byte private key.
 *  Out:    pubkey:     pointer to a 33-byte (if compressed) or 65-byte (if uncompressed)
 *                      area to store the public key.
 *          pubkeylen:  pointer to int that will be updated to contains the pubkey's
 *                      length.
 *  Returns: 1: secret was valid, public key stores
 *           0: secret was invalid, try again.
 *
int secp256k1_ecdsa_pubkey_create(unsigned char *pubkey, int *pubkeylen, const unsigned char *seckey, int compressed);
*/
