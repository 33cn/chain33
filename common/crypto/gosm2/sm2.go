// NCNL is No Copyright and No License
//
//

package gosm2

/*
#cgo CFLAGS: -I /usr/local/include
#cgo LDFLAGS: -L /usr/local/lib -lcrypto -lssl

#include <openssl/ec.h>
#include <openssl/bn.h>
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <openssl/engine.h>
#include <openssl/sm2.h>
//#include <openssl/ossl_typ.h>

RAND_METHOD fake_rand;
const RAND_METHOD *old_rand;

static const char rnd_seed[] =
	"string to make the random number generator think it has entropy";
static const char *rnd_number = NULL;

static int fbytes(unsigned char *buf, int num)
{
	int ret = 0;
	BIGNUM *bn = NULL;

	if (!BN_hex2bn(&bn, rnd_number)) {
		goto end;
	}
	if (BN_num_bytes(bn) > num) {
		goto end;
	}
	memset(buf, 0, num);
	if (!BN_bn2bin(bn, buf + num - BN_num_bytes(bn))) {
		goto end;
	}
	ret = 1;
end:
	BN_free(bn);
	return ret;
}

static int change_rand(const char *hex)
{
	if (!(old_rand = RAND_get_rand_method())) {
		return 0;
	}

	fake_rand.seed		= old_rand->seed;
	fake_rand.cleanup	= old_rand->cleanup;
	fake_rand.add		= old_rand->add;
	fake_rand.status	= old_rand->status;
	fake_rand.bytes		= fbytes;
	fake_rand.pseudorand	= old_rand->bytes;

	if (!RAND_set_rand_method(&fake_rand)) {
		return 0;
	}

	rnd_number = hex;
	return 1;
}

static int restore_rand(void)
{
	rnd_number = NULL;
	if (!RAND_set_rand_method(old_rand))
		return 0;
	else	return 1;
}

void gmssl_init() {
	ERR_load_crypto_strings();
	OpenSSL_add_all_algorithms();
	//OPENSSL_config(NULL);
}

void gmssl_clearup() {
	EVP_cleanup();
	CRYPTO_cleanup_all_ex_data();
	ERR_free_strings();
}

EC_KEY* pub_from_priv(EC_KEY* priv) {
	EC_KEY* pub = EC_KEY_dup(priv);
	EC_KEY_set_private_key(pub, NULL);
	return pub;
}


int generatePem(EC_KEY *eckey, char **pem) {
    int rn = 0;
    char * errorMessage = "";
    char *pemholder = calloc(240, sizeof(char));
    BIO *out = NULL;

    if ((out = BIO_new(BIO_s_mem())) == NULL) {
        rn = -1;
        errorMessage = "Error in BIO_new(BIO_s_mem())";
        goto clearVariables;
    }

    BUF_MEM *buf = NULL;

    if (PEM_write_bio_ECPrivateKey(out, eckey, NULL, NULL, 0, NULL, NULL) == 0) {
        printf("PEM_write_bio_ECPrivateKey error");
    }

    BIO_get_mem_ptr(out, &buf);

    memcpy(pemholder, buf->data, 227);
    printf("%s\n", pemholder);
    if (buf->data[226] == '\n') {
        pemholder[226] = '\0';
        rn = 226;
    } else {
        rn = -1;
        errorMessage = "invalid PEM generated";
        goto clearVariables;
    }
    *pem = pemholder;

    goto clearVariables;

clearVariables:
    if (out != NULL)
        BIO_free_all(out);
    if (errorMessage[0] != '\0')
        printf("Error: %s\n", errorMessage);

    return rn;
};

EC_KEY* new_key(const char* s) {
	EC_KEY* eckey = EC_KEY_new_by_curve_name(NID_sm2p256v1);
	if (!eckey) {
		return NULL;
	}

	EC_KEY_set_asn1_flag(eckey, OPENSSL_EC_NAMED_CURVE);
    EC_KEY_generate_key(eckey);

	BIGNUM *pbn = NULL;
  	if (!BN_hex2bn(&pbn, s)) {
		return NULL;
    }

    if (!EC_KEY_set_private_key(eckey, pbn)) {
    	BN_free(pbn);
		return NULL;
    }
	BN_free(pbn);
	return eckey;
}

EC_KEY* generate_key() {
	EC_KEY* eckey = EC_KEY_new_by_curve_name(NID_sm2p256v1);
	if (!eckey) {
		return NULL;
	}

	EC_KEY_set_asn1_flag(eckey, OPENSSL_EC_NAMED_CURVE);

    EC_KEY_generate_key(eckey);

	return eckey;
 }

 int sm2_digest(EC_KEY* eckey, const unsigned char* msg, size_t mlen, unsigned char* out, size_t *outlen) {
 	unsigned char dgst[EVP_MAX_MD_SIZE];
	size_t dgstlen;
	const EVP_MD *id_md = EVP_sm3();
	const EVP_MD *msg_md = EVP_sm3();

	dgstlen = sizeof(dgst);
	if (!SM2_compute_message_digest(id_md, msg_md, msg, mlen, SM2_DEFAULT_ID_GMT09, SM2_DEFAULT_ID_LENGTH,
		dgst, &dgstlen, eckey)) {
		return -1;
	}

	memcpy(out, dgst, dgstlen);
	*outlen = dgstlen;
	return 1;
 }

 int encrypt(EC_KEY* eckey, const unsigned char* in, int inlen, unsigned char** out, size_t *outlen) {
 	EVP_PKEY* pkey = EVP_PKEY_new();
 	if (!EVP_PKEY_set1_EC_KEY(pkey, eckey)) {
 		EVP_PKEY_free(pkey);
 		printf("go here0");
		return -1;
 	}

 	EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new(pkey, NULL);
 	if (!ctx) {
 		EVP_PKEY_free(pkey);
 		printf("%s\n", "go here1");
		return -1;
 	}
 	if (EVP_PKEY_encrypt_init(ctx) <= 0) {
 		printf("%s\n", "go here2");
		return -1;
 	}

 	if (EVP_PKEY_encrypt(ctx, NULL, outlen, in, inlen) <= 0){
 		printf("%s\n", "go here3");
		return -1;
 	}

 	*out = OPENSSL_malloc(*outlen);
	if (!*out) {
		printf("%s\n", "go here4");
		return -1;
	}

	if (EVP_PKEY_encrypt(ctx, *out, outlen, in, inlen) <= 0) {
		printf("%s\n", "go here5");
		return -1;
	}

	printf("go here!!!\n");
	return 1;
 }

 int decrypt(EC_KEY* eckey, const unsigned char* in, int inlen, unsigned char** out, size_t *outlen) {
 	EVP_PKEY* pkey = EVP_PKEY_new();
 	if (!EVP_PKEY_set1_EC_KEY(pkey, eckey)) {
 		EVP_PKEY_free(pkey);
		return -1;
 	}

 	EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new(pkey, NULL);
 	if (!ctx) {
 		EVP_PKEY_free(pkey);
		return -1;
 	}
 	if (EVP_PKEY_decrypt_init(ctx) <= 0) {
		return -1;
 	}

 	if (EVP_PKEY_decrypt(ctx, NULL, outlen, in, inlen) <= 0){
		return -1;
 	}

 	*out = OPENSSL_malloc(*outlen);
	if (!*out) {
		return -1;
	}

	if (EVP_PKEY_decrypt(ctx, *out, outlen, in, inlen) <= 0) {
		return -1;
	}

	return 1;
 }

int sig2bin(const ECDSA_SIG *sig, unsigned char* out) {
	int r = i2d_ECDSA_SIG(sig, &out);
	if (r <= 0) {
		return -1;
	}
	return r;
}

*/
import "C"

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"math/rand"
	"time"
	"unsafe"

	// "github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
)

func Init() {
	C.gmssl_init()
}

func Clearup() {
	C.gmssl_clearup()
}

func init() {
	Init()
	rand.Seed(time.Now().Unix())
}

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

type PublicKey struct {
	pub *C.EC_KEY
}

func (k *PublicKey) Hash() ([]byte, error) {
	buf, err := k.OStrEncode()
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(buf)
	return hash[:], nil
}

func (k *PublicKey) HashString() string {
	hash, err := k.Hash()
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash)
}

func (k *PublicKey) Address() ([]byte, error) {
	hash, err := k.Hash()
	if err != nil {
		return nil, err
	}
	return Ripemd160(hash), nil
}

func (k *PublicKey) AddressString() string {
	buf, err := k.Address()
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func (k *PublicKey) OStrEncode() ([]byte, error) {
	r := C.i2o_ECPublicKey(k.pub, (**C.uchar)(unsafe.Pointer(nil)))
	if r == 0 {
		return nil, errors.New("i2o_ECPublicKey error0")
	}

	var pp unsafe.Pointer
	defer C.free(pp)
	r = C.i2o_ECPublicKey(k.pub, (**C.uchar)(unsafe.Pointer(&pp)))
	if r == 0 {
		return nil, errors.New("i2o_ECPublicKey error1")
	}

	return C.GoBytes(pp, r), nil
}

func (k *PublicKey) OStrDecode(buf []byte) error {
	pub, err := OStrDecodePublicKey(buf)
	if err != nil {
		return err
	}
	C.free(unsafe.Pointer(k.pub))
	k.pub = pub.pub
	return nil
}

func OStrDecodePublicKey(buf []byte) (*PublicKey, error) {
	key := C.EC_KEY_new_by_curve_name(C.NID_sm2p256v1)
	cbuf := C.CBytes(buf)
	defer C.free(cbuf)

	pub := C.o2i_ECPublicKey(&key, (**C.uchar)(unsafe.Pointer(&cbuf)), C.long(len(buf)))
	if unsafe.Pointer(pub) == nil {
		return nil, errors.New("o2i_ECPublicKey error.")
	}
	return &PublicKey{pub: pub}, nil
}

func (k *PublicKey) EncodeString() string {
	out, err := k.OStrEncode()
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(out)
}

func (k *PublicKey) DecodeString(s string) error {
	buf, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	return k.OStrDecode(buf)
}

type PriveKey struct {
	*PublicKey
	priv *C.EC_KEY
}

func (k *PriveKey) DEREncode() ([]byte, error) {
	old := C.EC_KEY_get_enc_flags(k.priv)
	C.EC_KEY_set_enc_flags(k.priv, old|C.EC_PKEY_NO_PARAMETERS)

	r := C.i2d_ECPrivateKey(k.priv, (**C.uchar)(unsafe.Pointer(nil)))
	if r == 0 {
		return nil, errors.New("i2d_ECPrivateKey error0")
	}

	var pp unsafe.Pointer
	defer C.free(pp)
	r = C.i2d_ECPrivateKey(k.priv, (**C.uchar)(unsafe.Pointer(&pp)))
	if r == 0 {
		return nil, errors.New("i2d_ECPrivateKey error1")
	}

	gp := C.GoBytes(pp, r)
	// log.Println(gp)

	return gp, nil
}

func (k *PriveKey) DERDecode(buf []byte) error {
	priv, err := DERDecodePrivateKey(buf)
	if err != nil {
		return err
	}
	C.free(unsafe.Pointer(k.priv))
	C.free(unsafe.Pointer(k.pub))
	k.priv = priv.priv
	k.pub = priv.pub
	return nil
}

func DERDecodePrivateKey(buf []byte) (*PriveKey, error) {
	key := C.EC_KEY_new_by_curve_name(C.NID_sm2p256v1)
	cbuf := C.CBytes(buf)
	defer C.free(cbuf)

	priv := C.d2i_ECPrivateKey(&key, (**C.uchar)(unsafe.Pointer(&cbuf)), C.long(len(buf)))
	if unsafe.Pointer(priv) == nil {
		return nil, errors.New("d2i_ECPrivateKey error.")
	}
	pub := C.pub_from_priv(priv)
	return &PriveKey{PublicKey: &PublicKey{pub: pub}, priv: priv}, nil
}

func (k *PriveKey) EncodeString() string {
	buf, err := k.DEREncode()
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(buf)
}

func (k *PriveKey) DecodeString(s string) error {
	buf, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	return k.DERDecode(buf)
}

func NewKeyFormSecret(secret string) (*PriveKey, error) {
	hexs := hex.EncodeToString([]byte(secret))
	sc := C.CString(hexs)
	defer C.free(unsafe.Pointer(sc))
	priv := C.new_key(sc)
	if priv == nil {
		return nil, errors.New("generate_key error")
	}
	pub := C.pub_from_priv(priv)

	return &PriveKey{PublicKey: &PublicKey{pub: pub}, priv: priv}, nil
}

func NewKey() (*PriveKey, error) {
	priv := C.generate_key()
	if priv == nil {
		return nil, errors.New("generate_key error")
	}
	pub := C.pub_from_priv(priv)

	return &PriveKey{PublicKey: &PublicKey{pub: pub}, priv: priv}, nil
}

func Encrypt1(pub *PublicKey, msg []byte) ([]byte, error) {
	cin := C.CBytes(msg)
	defer C.free(unsafe.Pointer(cin))

	inlen := C.int(len(msg))
	outlen := C.size_t(0)

	var pp unsafe.Pointer
	defer C.free(pp)

	ret := C.encrypt(pub.pub, (*C.uchar)(cin), inlen, (**C.uchar)(unsafe.Pointer(&pp)), &outlen)
	if ret != 1 {
		return nil, errors.New("encrypt error.")
	}

	return C.GoBytes(unsafe.Pointer(pp), C.int(outlen)), nil
}

func Decrypt1(priv *PriveKey, cipertext []byte) ([]byte, error) {
	cin := C.CBytes(cipertext)
	defer C.free(unsafe.Pointer(cin))

	inlen := C.int(len(cipertext))
	outlen := C.size_t(0)

	var pp unsafe.Pointer
	defer C.free(pp)

	ret := C.decrypt(priv.priv, (*C.uchar)(cin), inlen, (**C.uchar)(unsafe.Pointer(&pp)), &outlen)
	if ret != 1 {
		return nil, errors.New("decrypt error.")
	}

	return C.GoBytes(unsafe.Pointer(pp), C.int(outlen)), nil
}

func Encrypt(pub *PublicKey, msg []byte) ([]byte, error) {
	cmsg := C.CBytes(msg)
	defer C.free(unsafe.Pointer(cmsg))

	C.change_rand(C.CString("4C62EEFD6ECFC2B95B92FD6C3D9575148AFA17425546D49018E5388D49DD7B4F"))
	defer C.restore_rand()

	mlen := C.size_t(len(msg))
	outlen := C.size_t(0)

	ret := C.SM2_encrypt(C.NID_sm3, (*C.uchar)(cmsg), mlen, (*C.uchar)(unsafe.Pointer(nil)), &outlen, pub.pub)
	if ret != 1 {
		return nil, errors.New("SM2_encrypt error0.")
	}

	out := make([]byte, int(outlen))
	cout := C.CBytes(out)
	defer C.free(unsafe.Pointer(cout))

	ret = C.SM2_encrypt(C.NID_sm3, (*C.uchar)(cmsg), mlen, (*C.uchar)(cout), &outlen, pub.pub)
	if ret != 1 {
		return nil, errors.New("SM2_encrypt error1.")
	}

	return C.GoBytes(unsafe.Pointer(cout), C.int(outlen)), nil
}

func Decrypt(priv *PriveKey, cipertext []byte) ([]byte, error) {
	ctxt := C.CBytes(cipertext)
	defer C.free(unsafe.Pointer(ctxt))

	txtlen := C.size_t(len(cipertext))
	outlen := C.size_t(0)
	ret := C.SM2_decrypt(C.NID_sm3, (*C.uchar)(ctxt), txtlen, (*C.uchar)(unsafe.Pointer(nil)), &outlen, priv.priv)
	if ret != 1 {
		return nil, errors.New("SM2_decrypt error0.")
	}
	out := make([]byte, outlen)
	cout := C.CBytes(out)
	defer C.free(unsafe.Pointer(cout))

	ret = C.SM2_decrypt(C.NID_sm3, (*C.uchar)(ctxt), txtlen, (*C.uchar)(cout), &outlen, priv.priv)
	if ret != 1 {
		return nil, errors.New("SM2_decrypt error1.")
	}

	return C.GoBytes(unsafe.Pointer(cout), C.int(outlen)), nil
}

func Hash2(msg []byte) ([]byte, error) {
	cmsg := C.CBytes(msg)
	defer C.free(cmsg)
	mlen := C.size_t(len(msg))

	out := make([]byte, 64)
	cout := C.CBytes(out)
	defer C.free(cout)
	outlen := C.uint(0)
	if C.EVP_Digest(cmsg, mlen, (*C.uchar)(cout), &outlen, C.EVP_sm3(), (*C.ENGINE)(unsafe.Pointer(nil))) <= 0 {
		return nil, errors.New("EVP_Digest error")
	}
	return C.GoBytes(cout, C.int(outlen)), nil
}

func Hash(pub *PublicKey, msg []byte) ([]byte, error) {
	cmsg := C.CBytes(msg)
	defer C.free(cmsg)
	mlen := C.size_t(len(msg))

	out := make([]byte, 64)
	cout := C.CBytes(out)
	defer C.free(cout)
	outlen := C.size_t(0)

	if C.sm2_digest(pub.pub, (*C.uchar)(cmsg), mlen, (*C.uchar)(cout), &outlen) != 1 {
		return nil, errors.New("SM2_digest error.")
	}
	return C.GoBytes(cout, C.int(outlen)), nil
}

func Sign(priv *PriveKey, hash []byte) ([]byte, error) {
	ch := C.CBytes(hash)
	defer C.free(ch)

	sig := make([]byte, 256)
	csig := C.CBytes(sig)
	defer C.free(csig)
	siglen := C.uint(0)

	C.change_rand(C.CString("6CB28D99385C175C94F94E934817663FC176D925DD72B727260DBAAE1FB2F96F"))
	defer C.restore_rand()

	if C.SM2_sign(C.NID_undef, (*C.uchar)(ch), C.int(len(hash)), (*C.uchar)(csig), &siglen, priv.priv) != 1 {
		return nil, errors.New("SM2_sign error.")
	}

	return C.GoBytes(csig, C.int(siglen)), nil
}

func Verify(pub *PublicKey, hash, sig []byte) error {
	ch := C.CBytes(hash)
	defer C.free(ch)
	csig := C.CBytes(sig)
	defer C.free(csig)

	if C.SM2_verify(C.NID_undef, (*C.uchar)(ch), C.int(len(hash)), (*C.uchar)(csig), C.int(len(sig)), pub.pub) != 1 {
		return errors.New("SM2_verify error.")
	}

	return nil
}

func bin2bin(b []byte) (*C.BIGNUM, error) {
	bn := C.BN_new()
	// defer C.BN_free(bn)
	cb := C.CBytes(b)
	defer C.free(unsafe.Pointer(cb))
	bn = C.BN_bin2bn((*C.uchar)(cb), (C.int)(len(b)), bn)
	if unsafe.Pointer(nil) == unsafe.Pointer(bn) {
		return nil, errors.New("BN_bin2bn error")
	}
	return bn, nil
}

func Sigbin2Der(b []byte) ([]byte, error) {
	sig := C.ECDSA_SIG_new()
	defer C.ECDSA_SIG_free(sig)

	bnR, err := bin2bin(b[:32])
	if err != nil {
		return nil, err
	}
	bnS, err := bin2bin(b[32:])
	if err != nil {
		return nil, err
	}
	C.ECDSA_SIG_set0(sig, bnR, bnS)

	sigBuf := make([]byte, 256)
	csigBuf := C.CBytes(sigBuf)
	defer C.free(csigBuf)

	r := C.sig2bin(sig, (*C.uchar)(csigBuf))
	if r <= 0 {
		return nil, errors.New("i2d_ECDSA_SIG")
	}
	return C.GoBytes(csigBuf, C.int(r)), nil
}
