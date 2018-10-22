package privacy

import (
	"bytes"
	"testing"

	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	pubs_byte [10][]byte
	secs_byte [10][]byte
)

func init() {
	pubstrs := []string{"1570eb695fa38fa7c395ddcb90e53e9b4d366a920e9c4b3ec988807d6f21914d",
		"b48dbda2574f5c4f8e65c8e826948be67b6ab6ed0a46876c2ba74781eea27095",
		"780d3ade933ffb95d4fb4ca92b976547c00aa55078621ec07bbeef306bd85a84",
		"f9fbe125e540273849d93de90c7cf208dd0047e69b466bb5e699c4c377308271",
		"0f3177da65d5940b9ad76758d31fb2d6514e0fa1c27c6175a962c7af2cf81646",
		"f0b05d2832cf1aef343ae9ccad6eae90ab1410307bd89a2c253653676bb80240",
		"8acb3a2dad555d73c46bb77d06c82cf8e28110da01be061506704e0ea85c514a",
		"c6a04322f13531adc1f6fa9262a48c1d73a8cc9d59a1d426fcf9f98e7f72f1a2",
		"167739cfef153151cf7174b0460810996c4ca02f750ba5a8647315e71f144674",
		"8c0a31aa718b1a62da8c19af6a2a4b1ab4198c83ef3d8971cb022a20da6525d0",
	}

	secstrs := []string{"8f5b3b5407d40d99d7c1e6b61b022b2f18878c2b24d6d247dbd865f6fc80400b",
		"ff8c20da1d1fe90e130c833e98fb8923bd0acd37cc67f06d70ea0807711ef007",
		"a6878618b8296d349bea9eda9c9d2615e91ecaa34884b4e2e0e28596c7ca6107",
		"92822dd0f80bf92406f243428c75457ecc789c30f6f27b38ab1b4d884a2b5408",
		"cd7869b98766bfc9dff453e95605d97d243cec6dbe36746dcfabbad3fa37dc0f",
		"1461243ab5cb04f80d1911a8df74695a5d93ce54874486d22d1b3eadea01d50e",
		"35e417253c6b38b72e9fef0bdcae35355a6ccb488198585594287c1b11b7f50c",
		"520e7d99e5dcb4b57697fdde6e3b9f5727573817e2d5d8ecc152929fcfb0ba0a",
		"1f7de34fb8aca5dff5f177e569868f1d522ffe3f706296695897942cc7d41b06",
		"7692f2253ef30c1dc82f2345f11d66a452ed6bc9d8ab06d3b685b2049855a50f",
	}

	for i := 0; i < 10; i++ {
		pubs_byte[i], _ = common.FromHex(pubstrs[i])
		secs_byte[i], _ = common.FromHex(secstrs[i])
	}
}

func TestGenerateKeyImage1(t *testing.T) {
	var pub PubKeyPrivacy
	var sec PrivKeyPrivacy
	var expected, actual KeyImage
	tmp, err := common.FromHex("1570eb695fa38fa7c395ddcb90e53e9b4d366a920e9c4b3ec988807d6f21914d")
	if err != nil {
		t.Error("initialize public key from hex failed.")
	}
	copy(pub[:], tmp)
	tmp, err = common.FromHex("8f5b3b5407d40d99d7c1e6b61b022b2f18878c2b24d6d247dbd865f6fc80400b")
	if err != nil {
		t.Error("initialize secre key from hex failed.")
	}
	copy(sec[:], tmp)
	tmp, err = common.FromHex("e744a16a913da96bc29c5197d01bb00c4e59990bbccd7785ef413d98cae1696a")
	if err != nil {
		t.Error("initialize expected image from hex failed.")
	}
	copy(expected[:], tmp)
	err = generateKeyImage(&pub, &sec, &actual)
	if err != nil {
		t.Error("generateKeyImage() failed. error ", err)
	}
	if !bytes.Equal(actual[:], expected[:]) {
		t.Fatal("generateKeyImage() failed.")
	}
}

func TestGenerateKeyImage2(t *testing.T) {
	c, err := crypto.New(types.GetSignName("privacy", privacytypes.OnetimeED25519))
	if err != nil {
		t.Errorf("create Crypto failed. %v\n", err)
	}
	privkey, err := c.GenKey()
	if err != nil {
		t.Error("Generate private key failed.")
	}
	pubKey := privkey.PubKey().Bytes()
	_, err = GenerateKeyImage(privkey, pubKey[:])
	if err != nil {
		t.Errorf("Generate private key failed. %v\n", err)
	}
}

func TestGenerateRingSignature1(t *testing.T) {
	const maxCount = 1
	var image KeyImage
	var pubs [maxCount]*PubKeyPrivacy
	var sec PrivKeyPrivacy
	var signs [maxCount]*Sign
	index := 0
	prefix_hash, err := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")
	tmp, err := common.FromHex("e7d85d6e81512c5650adce0499d6c17a83e2e29a05c1166cd2171b6b9288b3c4")
	copy(image[:], tmp)
	tmp, err = common.FromHex("15e3cc7cdb904d62f7c20d7fa51923fa2839f9e0a92ff0eddf8c12bd09089c15")
	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		copy(pub[:], tmp[32*i:32*(i+1)])
		pubs[i] = &pub
	}
	tmp, err = common.FromHex("fd8de2e12410e3da20350e6f4083b73cc3396b4812c09af633d7c223bfb2560b")
	copy(sec[:], tmp)
	tmp, err = common.FromHex("b5c0a53bce99106e74284ee1d2c64a68a14e8b4c1735e8ab9ca7215a414f97084749935d87d516b5659a01e84b08279c42e649b2e500e2ede443fe68885b0206")
	for i := 0; i < maxCount; i++ {
		signs[i] = &Sign{}
	}
	err = generateRingSignature(prefix_hash, &image, pubs[:], &sec, signs[:], index)
	if err != nil {
		t.Error("generateRingSignature() cause error ", err)
	}
}

func TestCheckRingSignature1(t *testing.T) {
	const maxCount = 1
	var image KeyImage
	var pubs [maxCount]*PubKeyPrivacy
	var signs [maxCount]*Sign

	prefix_hash, err := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")
	if err != nil {
		t.Error("initialize public key from hex failed.")
	}
	tmp, err := common.FromHex("e7d85d6e81512c5650adce0499d6c17a83e2e29a05c1166cd2171b6b9288b3c4")
	copy(image[:], tmp)
	tmp, err = common.FromHex("15e3cc7cdb904d62f7c20d7fa51923fa2839f9e0a92ff0eddf8c12bd09089c15")
	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		copy(pub[:], tmp[32*i:32*(i+1)])
		pubs[i] = &pub
	}
	tmp, err = common.FromHex("a142d0180a6047cf883125a83617c7dd56e9d8153ec04a96b3c5d9a5e03cd70cfd6827b200d6c2f4b41fb5b0162117a5447e327c29482c358a0a3c82db88fb0f")
	for i := 0; i < maxCount; i++ {
		sign := Sign{}
		copy(sign[:], tmp[64*i:64*(i+1)])
		signs[i] = &sign
	}

	if !checkRingSignature(prefix_hash, &image, pubs[:], signs[:]) {
		t.Fatal("checkRingSignature() failed.")
	}
}

func TestCheckRingSignatureAPI1(t *testing.T) {
	const maxCount = 1
	var signatures types.RingSignatureItem
	publickeys := make([][]byte, maxCount)
	prefix_hash, err := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")
	if err != nil {
		t.Errorf("common.FromHex.", err)
	}
	keyimage, err := common.FromHex("e7d85d6e81512c5650adce0499d6c17a83e2e29a05c1166cd2171b6b9288b3c4")
	if err != nil {
		t.Errorf("common.FromHex.", err)
	}

	tmp, err := common.FromHex("15e3cc7cdb904d62f7c20d7fa51923fa2839f9e0a92ff0eddf8c12bd09089c15")
	for i := 0; i < maxCount; i++ {
		publickeys[i] = append(publickeys[i], tmp...)
	}

	tmp, err = common.FromHex("a142d0180a6047cf883125a83617c7dd56e9d8153ec04a96b3c5d9a5e03cd70cfd6827b200d6c2f4b41fb5b0162117a5447e327c29482c358a0a3c82db88fb0f")
	data := make([][]byte, maxCount)
	for i := 0; i < maxCount; i++ {
		data[i] = make([]byte, 0)
		data[i] = tmp
	}
	signatures.Signature = data

	if !CheckRingSignature(prefix_hash, &signatures, publickeys, keyimage) {
		t.Fatal("checkRingSignature() failed.")
	}
}

func TestCheckRingSignature3(t *testing.T) {
	const maxCount = 3
	var image KeyImage
	var pubs [maxCount]*PubKeyPrivacy
	var signs [maxCount]*Sign

	prefix_hash, err := common.FromHex("9e7ff8bde0e318543dcedbe34c51c6b25a850578adae2e7930bbda5224c77ef5")
	if err != nil {
		t.Error("initialize public key from hex failed.")
	}
	tmp, err := common.FromHex("04e593e5e4028ce1c1194eb473efc21359b114737e5a64f14420b3cf5b22204b")
	copy(image[:], tmp)
	tmp, err = common.FromHex("6bfc9654082a7da3055121aa69ddb46852577be71d6c9a204aae3492f0db7e41194f27c9fe4d81cc8421bf8256374edf660806d78b4ed7914a3b74359c8ac0bd65ff1bca674607f7948ea0ae8e83b6d9c5092942b52d2847b6cf44c9c609264d")
	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		copy(pub[:], tmp[32*i:32*(i+1)])
		pubs[i] = &pub
	}
	tmp, err = common.FromHex("30041e9694c3184980c3bb87f817eab3f973cd969810ec9df4d2feeee907970693eba4bc5436dc7cf49ce476e091bf74d20003f0f73f6d0412909ed8c1a10701c9c4ec11623dd3c50980ead83865a03dfa27614e5e9fb875d75667c11ced390d438f5dd04a137c73a0ec9ca36dfab62c948ce596722067de0315b570db1f720bab7fb7ea1b124f55c9633548f06d1bb403d7e2e15a1fed70ab2865e324ef340327f6d0ad0a7129b272ce12a5a63836a4e96e95897ee44cc22a7048023f438006")
	for i := 0; i < maxCount; i++ {
		sign := Sign{}
		copy(sign[:], tmp[64*i:64*(i+1)])
		signs[i] = &sign
	}

	if !checkRingSignature(prefix_hash, &image, pubs[:], signs[:]) {
		t.Fatal("checkRingSignature() failed.")
	}
}

func TestCheckRingSignature17(t *testing.T) {
	const maxCount = 17
	var image KeyImage
	var pubs [maxCount]*PubKeyPrivacy
	var signs [maxCount]*Sign

	pubstrs := []string{
		"5b88cc7df0447be04ce1dcf6dd8a7a7ca85b10f8568fcbbe707946c92fd964d4",
		"0ef6633efff70769ad31c7e79527677e915a5e89c64c7df1d5fdb03b53941b87",
		"01f0336eccf6a061500a88d07a3d5c153299f5bc9b3888dd81dfd52a0a9497c2",
		"034c82d3a3285260507c2b6ce7f1c844d4b5640e11a4fccbfb3249b09fad63b4",
		"b1100f59dd616cd6e8791e1e0efc8074d748151158defbfd02a6e2fc09aae7c4",
		"72acb61ba447888273b8e3e0b0bbed53d89eab1885759f47080a550b282a518e",
		"46d503d94a2faf805ad32546834dfd8e5f0501075d32eeaa3b1359433c6a53b3",
		"c75d34f6832923b3bf22de92f16ab4c3ced2731899a9e46a8747fe77732ae29e",
		"159b650ac9824a62f9bf89bea1db67fae28a2bde63e5489c756d2b8f06601125",
		"5c458a7a79a5fefa84c2f8225576ed124f0c781e493a99bb94e43aa1db30581d",
		"64963159ecb39690ab261495a3faee331f5c48ca04f916cf60eb43624963e381",
		"e6062080fcb588f1327fe6856d3d73d2542779e2ba8c40b52ab1f9d0505f0caf",
		"7b3837fc2e2c0bb4ca1a4032d21f7c1959552899188787c65a5643829e698744",
		"32c3960095f87c4e47b330b0528ca43c1d9dc2aae59e81ab04f4b496fbd419af",
		"8078bf92b788ece5b07a1596f8a03ea3d60c082285a95e0ff8338b15c5906fbb",
		"dd490e6d97bbaff5c7b04db17a1353b7506b1685c6fc3b41f9e112c2a0cf4925",
		"c618c6b0e328886be15e2c92052dcf74cf4745a4e5e53f6d2417a9c9d139c167",
	}

	prefix_hash, err := common.FromHex("1c909782c70567e9968ded1c05a4226a3e04a07ae9db48e0153a56b2a4684779")
	if err != nil {
		t.Error("initialize public key from hex failed.")
	}
	tmp, err := common.FromHex("199c5e78a71b320b704e01850a67c371fed3aca11a04077a36e10c808fdc5f57")
	copy(image[:], tmp)

	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		tmp, err = common.FromHex(pubstrs[i])
		copy(pub[:], tmp)
		pubs[i] = &pub
	}
	tmp, err = common.FromHex("f5b1a5079f6122d2370da403efc2e5504b07d67ade7ca57ad3720c7e00a676096ad418262cd9f8c9afee8e58be894cb2bf6387a5018f67d629bf8845bdc9b307aee86f85f8612f74ed870abc81d54cb38f2f9108c20dbf81eb7c1160935bc801cdca5d3f09f1d4bac810cb03ec32753f65a2071a97ba787c2d550ea5ff4d890ac398a1576d108ee67138e7596af54199247c6413e06124af8db76c0ad67c160f351e53029cac4a073c8b188eb8b5151ebf1f34a6693ddbf467ac8c20eec51204c3ca29fd5814f0848b966d2bca9543fc953a0d04bb4782950123d87bd378f60d3cdc5e44160ed4265b7cd84a8200244fa24bac86c50c35b885a15f1bc1de9909fabab87d504fc0cb0b263564e606505477ded1f89543bd4d6ed87ce61918f705359c525c0e0a4cc1d77043c71dfdf9c6834e2623197af1629e34962d940eb4080d06dddaa6e12953232595906289e4712f6aa8c1189f336b823a6a38ac95390b63c508a3409256f0c4df687d4b49990d9878fc5c97af32aa99bc9ffa1215860f69fc141c763fe1f4ed3ccc72e6817e926164752686d55238e74032c47c8c580f7042a62bd74858768815a05b58887f9828bf0f184eceba1416ac294abe453105a0de353316374b1c0560acc0f98181e3229084721a0d2a6ed891d57b805984044372cbb566c91696f5d63e7e5d8cc906c5fb029e71e87a339fcbaee9502f3d01dfd203dbd04797c33370dd0626baabfb9577ca1c294d1d8c6b82ac9624496f04893ad6b2222799cfd1d4ec525b011ecf72b01f6f3b8a6a165f0169fa7cc0960025787d4253e1064e71190d36965605adb850e72e17685e35772cdd3fdfb3d505e96e5eb0ecf8c11bcef62c79e0d49e3ebc76e56d18753403a880b7d7ed399c0e143df235ac8b0748d99e1b0cdcc511abc079c1bdbfead04265d9ad44e5dd7c0e97e4afbacba478539071db02315cc9326c8edb3d8530c7fc78af9ca12b241c04f87bffab95d85f8486ff01aec75e7e38a75ff023e4b2b0f59999c4ac067b500863a63de803da2c89de74878879a39aec6cf800b117ef89ea068a71db5ef70200163d1b4beaeea8c642430e15a64baefd350cb16ca5dae6563c13162904908301485d2c4e5bac6d55dd947a169a38df2848d5e227482760f6d818c45f66a6380507d691ef87e3550626254da3ec65411e8e5aa2ab4846222e208c9ba18cd5ee09617398af31b9eac38b59180bfaec538e5542d698da20016720ced6edb7ba270059c1e1e7107d71524e4a9cf562b5da263b52368f477c5205c768ac36975a250310c37d1f40f5a6eac857ed6082c12f5cf20a0403afcde3f1ea2e4743cb1ba10fd5d92038b1fc65d09467aeac9fe7cd24f78b9376d47d2ef1d3677ede62b2640544dd1b301292b8d082a97bd910ca24fbb5eea878f2cfa4e983bb25a8a66a25065988477eec0c1815a3fa2ff421621707954e9eacdf823eea02c2e3742797c6097e570479b36ca4580c5be9bab187de6a1fa6609768b3fa6448b5bc56d6de820d")
	for i := 0; i < maxCount; i++ {
		sign := Sign{}
		copy(sign[:], tmp[64*i:64*(i+1)])
		signs[i] = &sign
	}

	if !checkRingSignature(prefix_hash, &image, pubs[:], signs[:]) {
		t.Fatal("checkRingSignature() failed.")
	}
}

func TestRingSignatere1(t *testing.T) {
	const maxCount = 2
	var image KeyImage
	var sec PrivKeyPrivacy
	var pubs [maxCount]*PubKeyPrivacy
	var signs [maxCount]*Sign
	index := 0

	// 初始化测试数据
	prefix_hash, _ := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")
	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		sign := Sign{}
		pubs[i] = &pub
		signs[i] = &sign

		copy(pub[:], pubs_byte[i])
		if i == index {
			// 创建 KeyImage
			copy(sec[:], secs_byte[i])
			generateKeyImage(&pub, &sec, &image)
		}
	}
	// 创建环签名
	err := generateRingSignature(prefix_hash, &image, pubs[:], &sec, signs[:], index)
	if err != nil {
		t.Fatal("generateRingSignature() failed. error ", err)
	}
	// 消炎环签名
	if !checkRingSignature(prefix_hash, &image, pubs[:], signs[:]) {
		t.Fatal("checkRingSignature() failed.")
	}
}

func testRingSignatureOncetime(maxCount int, t *testing.T) {
	var image KeyImage
	var sec PrivKeyPrivacy
	index := 0
	pubs := make([]*PubKeyPrivacy, maxCount)
	signs := make([]*Sign, maxCount)

	// 初始化测试数据
	prefix_hash, _ := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")

	c, err := crypto.New(types.GetSignName("privacy", privacytypes.OnetimeED25519))
	if err != nil {
		t.Errorf("create Crypto failed. %v\n", err)
	}
	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		sign := Sign{}

		privkey, err := c.GenKey()
		if err != nil {
			t.Error("Generate private key failed.")
		}
		pubKey := privkey.PubKey().Bytes()

		pubs[i] = &pub
		signs[i] = &sign

		copy(pub[:], pubKey)
		if i == index {
			// 创建 KeyImage
			copy(sec[:], privkey.Bytes())
			err = generateKeyImage(&pub, &sec, &image)
			if err != nil {
				t.Errorf("generateKeyImage() failed. error ", err)
			}
		}
	}
	// 创建环签名
	err = generateRingSignature(prefix_hash, &image, pubs[:], &sec, signs[:], index)
	if err != nil {
		t.Fatal("generateRingSignature() failed. error ", err)
	}
	// 消炎环签名
	if !checkRingSignature(prefix_hash, &image, pubs[:], signs[:]) {
		t.Fatal("checkRingSignature() failed.")
	}
}

func TestRingSignatere2(t *testing.T) {
	testRingSignatureOncetime(100, t)
}

func TestGenerateRingSignatureAPI(t *testing.T) {
	const maxCount = 10
	var utxos []*privacytypes.UTXOBasic
	var keyImage []byte
	var sec [64]byte

	rand.Seed(time.Now().Unix())
	// step1. init params
	c, err := crypto.New(types.GetSignName("privacy", privacytypes.OnetimeED25519))
	if err != nil {
		t.Errorf("create Crypto failed. %v\n", err)
	}
	privkey, err := c.GenKey()
	if err != nil {
		t.Error("Generate private key failed.")
	}

	realUtxoIndex := rand.Int() % maxCount
	prefix_hash, _ := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")
	utxos = make([]*privacytypes.UTXOBasic, maxCount)
	for i := 0; i < maxCount; i++ {
		utxo := privacytypes.UTXOBasic{}
		utxos[i] = &utxo
		utxo.OnetimePubkey = append(utxo.OnetimePubkey[:], pubs_byte[i]...)
		if i == realUtxoIndex {
			pubKey := privkey.PubKey().Bytes()
			// 增加指定的密钥对
			copy(utxo.OnetimePubkey[:], pubKey)
			copy(sec[:], privkey.Bytes())

			// 创建 KeyImage
			image, err := GenerateKeyImage(privkey, pubKey[:])
			if err != nil {
				t.Errorf("Generate private key failed. %v\n", err)
			}
			keyImage = append(keyImage, image[:]...)
		}
	}

	var signaturedata *types.RingSignatureItem
	// step2. generate ring signature
	if signaturedata, err = GenerateRingSignature(prefix_hash, utxos, sec[:], realUtxoIndex, keyImage); err != nil {
		t.Errorf("GenerateRingSignature() failed. ", err)
	}

	publickeys := make([][]byte, maxCount)
	for i := 0; i < maxCount; i++ {
		publickeys[i] = append(publickeys[i], utxos[i].OnetimePubkey...)
	}
	// step3. checksignature
	if !CheckRingSignature(prefix_hash, signaturedata, publickeys, keyImage) {
		t.Error("CheckRingSignature() failed.")
	}
}

func benchRingSignatureOncetime(maxCount int) {
	var image KeyImage
	var sec PrivKeyPrivacy
	index := 0
	pubs := make([]*PubKeyPrivacy, maxCount)
	signs := make([]*Sign, maxCount)

	// 初始化测试数据
	prefix_hash, _ := common.FromHex("fd1f64844a7d6a9f74fc2141bceba9d9d69b1fd6104f93bfa42a6d708a6ab22c")

	c, _ := crypto.New(types.GetSignName("privacy", privacytypes.OnetimeED25519))
	for i := 0; i < maxCount; i++ {
		pub := PubKeyPrivacy{}
		sign := Sign{}

		privkey, _ := c.GenKey()
		pubKey := privkey.PubKey().Bytes()

		pubs[i] = &pub
		signs[i] = &sign

		copy(pub[:], pubKey)
		if i == index {
			// 创建 KeyImage
			copy(sec[:], privkey.Bytes())
			generateKeyImage(&pub, &sec, &image)
		}
	}
	// 创建环签名
	generateRingSignature(prefix_hash, &image, pubs[:], &sec, signs[:], index)
	// 效验环签名
	checkRingSignature(prefix_hash, &image, pubs[:], signs[:])
}

func Benchmark_RingSignature(b *testing.B) {
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		benchRingSignatureOncetime(b.N)
	}
	b.StopTimer()
}

// 最终版本的单元测试测试用例集合

// 最终版本的压力测试测试用例集合
func Benchmark_RingSignatureAllStep(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
