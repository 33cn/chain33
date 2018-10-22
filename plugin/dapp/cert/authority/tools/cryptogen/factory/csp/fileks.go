package csp

import (
	"io/ioutil"
	"os"
	"sync"

	"errors"

	"encoding/hex"
	"fmt"
	"path/filepath"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/tools/cryptogen/factory/utils"
	auth "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/utils"
)

var logger = log.New("tools", "cryptogen")

func NewFileBasedKeyStore(pwd []byte, path string, readOnly bool) (KeyStore, error) {
	ks := &fileBasedKeyStore{}
	return ks, ks.Init(pwd, path, readOnly)
}

type fileBasedKeyStore struct {
	path string

	readOnly bool
	isOpen   bool

	pwd []byte

	m sync.Mutex
}

func (ks *fileBasedKeyStore) Init(pwd []byte, path string, readOnly bool) error {
	if len(path) == 0 {
		return errors.New("An invalid KeyStore path provided. Path cannot be an empty string.")
	}

	ks.m.Lock()
	defer ks.m.Unlock()

	if ks.isOpen {
		return errors.New("KeyStore already initilized.")
	}

	ks.path = path
	ks.pwd = utils.Clone(pwd)

	err := ks.createKeyStoreIfNotExists()
	if err != nil {
		return err
	}

	err = ks.openKeyStore()
	if err != nil {
		return err
	}

	ks.readOnly = readOnly

	return nil
}

func (ks *fileBasedKeyStore) ReadOnly() bool {
	return ks.readOnly
}

func (ks *fileBasedKeyStore) StoreKey(k Key) (err error) {
	if ks.readOnly {
		return errors.New("Read only KeyStore.")
	}

	if k == nil {
		return errors.New("Invalid key. It must be different from nil.")
	}
	switch k.(type) {
	case *ecdsaPrivateKey:
		kk := k.(*ecdsaPrivateKey)

		err = ks.storePrivateKey(hex.EncodeToString(k.SKI()), kk.privKey)
		if err != nil {
			return fmt.Errorf("Failed storing ECDSA private key [%s]", err)
		}

	case *ecdsaPublicKey:
		kk := k.(*ecdsaPublicKey)

		err = ks.storePublicKey(hex.EncodeToString(k.SKI()), kk.pubKey)
		if err != nil {
			return fmt.Errorf("Failed storing ECDSA public key [%s]", err)
		}
	case *SM2PrivateKey:
		kk := k.(*SM2PrivateKey)

		err = ks.storePrivateKey(hex.EncodeToString(k.SKI()), kk.PrivKey)
		if err != nil {
			return fmt.Errorf("Failed storing SM2 private key [%s]", err)
		}

	case *SM2PublicKey:
		kk := k.(*SM2PublicKey)

		err = ks.storePublicKey(hex.EncodeToString(k.SKI()), kk.PubKey)
		if err != nil {
			return fmt.Errorf("Failed storing SM2 public key [%s]", err)
		}
	default:
		return fmt.Errorf("Key type not reconigned [%s]", k)
	}

	return
}

func (ks *fileBasedKeyStore) storePrivateKey(alias string, privateKey interface{}) error {
	rawKey, err := utils.PrivateKeyToPEM(privateKey, ks.pwd)
	if err != nil {
		logger.Error("Failed converting private key to PEM [%s]: [%s]", alias, err)
		return err
	}

	err = ioutil.WriteFile(ks.getPathForAlias(alias, "sk"), rawKey, 0700)
	if err != nil {
		logger.Error("Failed storing private key [%s]: [%s]", alias, err)
		return err
	}

	return nil
}

func (ks *fileBasedKeyStore) storePublicKey(alias string, publicKey interface{}) error {
	rawKey, err := utils.PublicKeyToPEM(publicKey, ks.pwd)
	if err != nil {
		logger.Error("Failed converting public key to PEM [%s]: [%s]", alias, err)
		return err
	}

	err = ioutil.WriteFile(ks.getPathForAlias(alias, "pk"), rawKey, 0700)
	if err != nil {
		logger.Error("Failed storing private key [%s]: [%s]", alias, err)
		return err
	}

	return nil
}

func (ks *fileBasedKeyStore) createKeyStoreIfNotExists() error {
	ksPath := ks.path
	missing, _ := auth.DirMissingOrEmpty(ksPath)

	if missing {
		err := ks.createKeyStore()
		if err != nil {
			logger.Error("Failed creating KeyStore At [%s]: [%s]", ksPath, err.Error())
			return nil
		}
	}

	return nil
}

func (ks *fileBasedKeyStore) createKeyStore() error {
	ksPath := ks.path

	os.MkdirAll(ksPath, 0755)

	return nil
}

func (ks *fileBasedKeyStore) openKeyStore() error {
	if ks.isOpen {
		return nil
	}

	return nil
}

func (ks *fileBasedKeyStore) getPathForAlias(alias, suffix string) string {
	return filepath.Join(ks.path, alias+"_"+suffix)
}
