package tendermint

import (
	"fmt"

	"encoding/json"

	"github.com/pkg/errors"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
)

/*
Requirements:
	- Valid new evidence must be persisted immediately and never forgotten
	- Uncommitted evidence must be continuously broadcast
	- Uncommitted evidence has a partial order, the evidence's priority

Impl:
	- First commit atomically in outqueue, pending, lookup.
	- Once broadcast, remove from outqueue. No need to sync
	- Once committed, atomically remove from pending and update lookup.
		- TODO: If we crash after committed but before removing/updating,
			we'll be stuck broadcasting evidence we never know we committed.
			so either share the state db and atomically MarkCommitted
			with ApplyBlock, or check all outqueue/pending on Start to see if its committed

Schema for indexing evidence (note you need both height and hash to find a piece of evidence):

"evidence-lookup"/<evidence-height>/<evidence-hash> -> EvidenceInfo
"evidence-outqueue"/<priority>/<evidence-height>/<evidence-hash> -> EvidenceInfo
"evidence-pending"/<evidence-height>/<evidence-hash> -> EvidenceInfo
*/

type envelope struct {
	Kind string           `json:"type"`
	Data *json.RawMessage `json:"data"`
}

type EvidenceInfo struct {
	Committed bool     `json:"committed"`
	Priority  int64    `json:"priority"`
	Evidence  envelope `json:"evidence"`
}

const (
	baseKeyLookup   = "evidence-lookup"   // all evidence
	baseKeyOutqueue = "evidence-outqueue" // not-yet broadcast
	baseKeyPending  = "evidence-pending"  // broadcast but not committed

)

func keyLookup(evidence types.Evidence) []byte {
	return keyLookupFromHeightAndHash(evidence.Height(), evidence.Hash())
}

// big endian padded hex
func bE(h int64) string {
	return fmt.Sprintf("%0.16X", h)
}

func keyLookupFromHeightAndHash(height int64, hash []byte) []byte {
	return _key("%s/%s/%X", baseKeyLookup, bE(height), hash)
}

func keyOutqueue(evidence types.Evidence, priority int64) []byte {
	return _key("%s/%s/%s/%X", baseKeyOutqueue, bE(priority), bE(evidence.Height()), evidence.Hash())
}

func keyPending(evidence types.Evidence) []byte {
	return _key("%s/%s/%X", baseKeyPending, bE(evidence.Height()), evidence.Hash())
}

func _key(fmt_ string, o ...interface{}) []byte {
	return []byte(fmt.Sprintf(fmt_, o...))
}

// EvidenceStore is a store of all the evidence we've seen, including
// evidence that has been committed, evidence that has been verified but not broadcast,
// and evidence that has been broadcast but not yet committed.
type EvidenceStore struct {
	db dbm.DB
}

func NewEvidenceStore(db dbm.DB) *EvidenceStore {
	if len(types.EvidenceType2Obj) == 0 {
		types.EvidenceType2Obj = map[string]interface{}{
			types.DuplicateVote: &types.DuplicateVoteEvidence{},
			types.MockGood:      &types.MockGoodEvidence{},
			types.MockBad:       &types.MockBadEvidence{},
		}
	}
	return &EvidenceStore{
		db: db,
	}
}

// PriorityEvidence returns the evidence from the outqueue, sorted by highest priority.
func (store *EvidenceStore) PriorityEvidence() (evidence []types.Evidence) {
	// reverse the order so highest priority is first
	l := store.ListEvidence(baseKeyOutqueue)
	l2 := make([]types.Evidence, len(l))
	for i := range l {
		l2[i] = l[len(l)-1-i]
	}
	return l2
}

// PendingEvidence returns all known uncommitted evidence.
func (store *EvidenceStore) PendingEvidence() (evidence []types.Evidence) {
	return store.ListEvidence(baseKeyPending)
}

// ListEvidence lists the evidence for the given prefix key.
// It is wrapped by PriorityEvidence and PendingEvidence for convenience.
func (store *EvidenceStore) ListEvidence(prefixKey string) (evidence []types.Evidence) {
	iter := store.db.Iterator([]byte(prefixKey), false)
	for iter.Next() {
		val := iter.Value()

		evi, err := store.EvidenceFromInfoBytes(val)
		if err != nil {
			fmt.Printf("ListEvidence evidence info unmarshal failed:%v", err)
		} else {
			evidence = append(evidence, evi)
		}
	}
	return evidence
}

// GetEvidence fetches the evidence with the given height and hash.
func (store *EvidenceStore) GetEvidence(height int64, hash []byte) *EvidenceInfo {
	key := keyLookupFromHeightAndHash(height, hash)
	val, e := store.db.Get(key)
	if e != nil {
		fmt.Printf(fmt.Sprintf(`GetEvidence: db get key %v failed:%v\n`, key, e))
	}

	if len(val) == 0 {
		return nil
	}
	var ei EvidenceInfo
	err := json.Unmarshal(val, &ei)
	if err != nil {
		fmt.Printf(fmt.Sprintf(`GetEvidence: unmarshal failed:%v\n`, err))
	}
	//wire.ReadBinaryBytes(val, &ei)
	return &ei
}

// AddNewEvidence adds the given evidence to the database.
// It returns false if the evidence is already stored.
func (store *EvidenceStore) AddNewEvidence(evidence types.Evidence, priority int64) bool {
	// check if we already have seen it
	ei_ := store.GetEvidence(evidence.Height(), evidence.Hash())
	if ei_ != nil && len(ei_.Evidence.Kind) == 0 {
		return false
	}

	eiBytes, err := EvidenceToInfoBytes(evidence, priority)
	if err != nil {
		fmt.Printf("AddNewEvidence failed:%v\n", err)
		return false
	}

	// add it to the store
	key := keyOutqueue(evidence, priority)
	store.db.Set(key, eiBytes)

	key = keyPending(evidence)
	store.db.Set(key, eiBytes)

	key = keyLookup(evidence)
	store.db.SetSync(key, eiBytes)

	return true
}

// MarkEvidenceAsBroadcasted removes evidence from Outqueue.
func (store *EvidenceStore) MarkEvidenceAsBroadcasted(evidence types.Evidence) {
	ei := store.getEvidenceInfo(evidence)
	key := keyOutqueue(evidence, ei.Priority)
	store.db.Delete(key)
}

// MarkEvidenceAsPending removes evidence from pending and outqueue and sets the state to committed.
func (store *EvidenceStore) MarkEvidenceAsCommitted(evidence types.Evidence) {
	// if its committed, its been broadcast
	store.MarkEvidenceAsBroadcasted(evidence)

	pendingKey := keyPending(evidence)
	store.db.Delete(pendingKey)

	ei := store.getEvidenceInfo(evidence)
	ei.Committed = true

	lookupKey := keyLookup(evidence)
	eiBytes, err := json.Marshal(ei)
	if err != nil {
		fmt.Printf("MarkEvidenceAsCommitted marshal failed:%v", err)
	}
	store.db.SetSync(lookupKey, eiBytes)
}

//---------------------------------------------------
// utils

func (store *EvidenceStore) getEvidenceInfo(evidence types.Evidence) EvidenceInfo {
	key := keyLookup(evidence)
	var ei EvidenceInfo
	b, e := store.db.Get(key)
	if e != nil {
		fmt.Printf(fmt.Sprintf(`getEvidenceInfo: db get key %v failed:%v\n`, key, e))
	}
	err := json.Unmarshal(b, &ei)
	if err != nil {
		fmt.Printf(fmt.Sprintf(`getEvidenceInfo: Unmarshal failed:%v\n`, err))
	}
	return ei
}

func EvidenceToInfoBytes(evidence types.Evidence, priority int64) ([]byte, error) {
	evi, err := json.Marshal(evidence)
	if err != nil {
		return nil, errors.Errorf("EvidenceToBytes marshal evidence failed:%v\n", err)
	}
	msg := json.RawMessage(evi)
	env := envelope{
		Kind: evidence.TypeName(),
		Data: &msg,
	}
	ei := EvidenceInfo{
		Committed: false,
		Priority:  priority,
		Evidence:  env,
	}

	eiBytes, err := json.Marshal(ei)
	if err != nil {
		return nil, errors.Errorf("EvidenceToBytes marshal evidence info failed:%v\n", err)
	}
	return eiBytes, nil
}

func (store *EvidenceStore) EvidenceFromInfoBytes(data []byte) (types.Evidence, error) {
	vote2 := EvidenceInfo{}
	err := json.Unmarshal(data, &vote2)
	if err != nil {
		return nil, errors.Errorf("BytesToEvidence Unmarshal evidence info failed:%v\n", err)
	}
	if v, ok := types.EvidenceType2Obj[vote2.Evidence.Kind]; ok {
		tmp := v.(types.Evidence).Copy()
		err = json.Unmarshal(*vote2.Evidence.Data, &tmp)
		if err != nil {
			return nil, errors.Errorf("BytesToEvidence Unmarshal evidence failed:%v\n", err)
		}
		return tmp, nil
	}
	return nil, errors.Errorf("BytesToEvidence not find evidence kind:%v\n", vote2.Evidence.Kind)

}
