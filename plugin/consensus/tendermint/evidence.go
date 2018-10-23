package tendermint

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	ttypes "gitlab.33.cn/chain33/chain33/plugin/consensus/tendermint/types"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
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

func keyLookup(evidence ttypes.Evidence) []byte {
	return keyLookupFromHeightAndHash(evidence.Height(), evidence.Hash())
}

// big endian padded hex
func bE(h int64) string {
	return fmt.Sprintf("%0.16X", h)
}

func keyLookupFromHeightAndHash(height int64, hash []byte) []byte {
	return _key("%s/%s/%X", baseKeyLookup, bE(height), hash)
}

func keyOutqueue(evidence ttypes.Evidence, priority int64) []byte {
	return _key("%s/%s/%s/%X", baseKeyOutqueue, bE(priority), bE(evidence.Height()), evidence.Hash())
}

func keyPending(evidence ttypes.Evidence) []byte {
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
	if len(ttypes.EvidenceType2Type) == 0 {
		ttypes.EvidenceType2Type = map[string]reflect.Type{
			ttypes.DuplicateVote: reflect.TypeOf(tmtypes.DuplicateVoteEvidence{}),
		}
	}
	if len(ttypes.EvidenceType2Obj) == 0 {
		ttypes.EvidenceType2Obj = map[string]ttypes.Evidence{
			ttypes.DuplicateVote: &ttypes.DuplicateVoteEvidence{},
		}
	}
	return &EvidenceStore{
		db: db,
	}
}

// PriorityEvidence returns the evidence from the outqueue, sorted by highest priority.
func (store *EvidenceStore) PriorityEvidence() (evidence []ttypes.Evidence) {
	// reverse the order so highest priority is first
	l := store.ListEvidence(baseKeyOutqueue)
	l2 := make([]ttypes.Evidence, len(l))
	for i := range l {
		l2[i] = l[len(l)-1-i]
	}
	return l2
}

// PendingEvidence returns all known uncommitted evidence.
func (store *EvidenceStore) PendingEvidence() (evidence []ttypes.Evidence) {
	return store.ListEvidence(baseKeyPending)
}

// ListEvidence lists the evidence for the given prefix key.
// It is wrapped by PriorityEvidence and PendingEvidence for convenience.
func (store *EvidenceStore) ListEvidence(prefixKey string) (evidence []ttypes.Evidence) {
	iter := store.db.Iterator([]byte(prefixKey), nil, false)
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
	return &ei
}

// AddNewEvidence adds the given evidence to the database.
// It returns false if the evidence is already stored.
func (store *EvidenceStore) AddNewEvidence(evidence ttypes.Evidence, priority int64) bool {
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
func (store *EvidenceStore) MarkEvidenceAsBroadcasted(evidence ttypes.Evidence) {
	ei := store.getEvidenceInfo(evidence)
	key := keyOutqueue(evidence, ei.Priority)
	store.db.Delete(key)
}

// MarkEvidenceAsPending removes evidence from pending and outqueue and sets the state to committed.
func (store *EvidenceStore) MarkEvidenceAsCommitted(evidence ttypes.Evidence) {
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

func (store *EvidenceStore) getEvidenceInfo(evidence ttypes.Evidence) EvidenceInfo {
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

func EvidenceToInfoBytes(evidence ttypes.Evidence, priority int64) ([]byte, error) {
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

func (store *EvidenceStore) EvidenceFromInfoBytes(data []byte) (ttypes.Evidence, error) {
	vote2 := EvidenceInfo{}
	err := json.Unmarshal(data, &vote2)
	if err != nil {
		return nil, errors.Errorf("BytesToEvidence Unmarshal evidence info failed:%v\n", err)
	}
	if v, ok := ttypes.EvidenceType2Type[vote2.Evidence.Kind]; ok {
		tmp := v.(ttypes.Evidence).Copy()
		err = json.Unmarshal(*vote2.Evidence.Data, &tmp)
		if err != nil {
			return nil, errors.Errorf("BytesToEvidence Unmarshal evidence failed:%v\n", err)
		}
		return tmp, nil
	}
	return nil, errors.Errorf("BytesToEvidence not find evidence kind:%v\n", vote2.Evidence.Kind)

}

//-------------------------evidence pool----------------------------
// EvidencePool maintains a pool of valid evidence
// in an EvidenceStore.
type EvidencePool struct {
	evidenceStore *EvidenceStore

	// needed to load validators to verify evidence
	stateDB *CSStateDB

	// latest state
	mtx   sync.Mutex
	state State

	// never close
	evidenceChan chan ttypes.Evidence
}

func NewEvidencePool(stateDB *CSStateDB, state State, evidenceStore *EvidenceStore) *EvidencePool {
	evpool := &EvidencePool{
		stateDB:       stateDB,
		state:         state,
		evidenceStore: evidenceStore,
		evidenceChan:  make(chan ttypes.Evidence),
	}
	return evpool
}

// EvidenceChan returns an unbuffered channel on which new evidence can be received.
func (evpool *EvidencePool) EvidenceChan() <-chan ttypes.Evidence {
	return evpool.evidenceChan
}

// PriorityEvidence returns the priority evidence.
func (evpool *EvidencePool) PriorityEvidence() []ttypes.Evidence {
	return evpool.evidenceStore.PriorityEvidence()
}

// PendingEvidence returns all uncommitted evidence.
func (evpool *EvidencePool) PendingEvidence() []ttypes.Evidence {
	return evpool.evidenceStore.PendingEvidence()
}

// State returns the current state of the evpool.
func (evpool *EvidencePool) State() State {
	evpool.mtx.Lock()
	defer evpool.mtx.Unlock()
	return evpool.state
}

// Update loads the latest
func (evpool *EvidencePool) Update(block *ttypes.TendermintBlock) {
	evpool.mtx.Lock()
	defer evpool.mtx.Unlock()

	state := evpool.stateDB.LoadState()
	if state.LastBlockHeight != block.Header.Height {
		panic(fmt.Sprintf("EvidencePool.Update: loaded state with height %d when block.Height=%d", state.LastBlockHeight, block.Header.Height))
	}
	evpool.state = state

	// NOTE: shouldn't need the mutex
	evpool.MarkEvidenceAsCommitted(block.Evidence.Evidence)
}

// AddEvidence checks the evidence is valid and adds it to the pool.
// Blocks on the EvidenceChan.
func (evpool *EvidencePool) AddEvidence(evidence ttypes.Evidence) (err error) {

	// TODO: check if we already have evidence for this
	// validator at this height so we dont get spammed

	if err := VerifyEvidence(evpool.stateDB, evpool.State(), evidence); err != nil {
		return err
	}

	// fetch the validator and return its voting power as its priority
	// TODO: something better ?
	valset, _ := evpool.stateDB.LoadValidators(evidence.Height())
	_, val := valset.GetByAddress(evidence.Address())
	priority := val.VotingPower

	added := evpool.evidenceStore.AddNewEvidence(evidence, priority)
	if !added {
		// evidence already known, just ignore
		return
	}

	tendermintlog.Info("Verified new evidence of byzantine behaviour", "evidence", evidence)

	// never closes. always safe to send on
	evpool.evidenceChan <- evidence
	return nil
}

// MarkEvidenceAsCommitted marks all the evidence as committed.
func (evpool *EvidencePool) MarkEvidenceAsCommitted(evidence []*tmtypes.EvidenceEnvelope) {
	for _, ev := range evidence {
		tmp := ttypes.EvidenceEnvelope2Evidence(ev)
		if tmp != nil {
			evpool.evidenceStore.MarkEvidenceAsCommitted(tmp)
		}
	}
}

// VerifyEvidence verifies the evidence fully by checking it is internally
// consistent and sufficiently recent.
func VerifyEvidence(stateDB *CSStateDB, s State, evidence ttypes.Evidence) error {
	height := s.LastBlockHeight

	evidenceAge := height - evidence.Height()
	maxAge := s.ConsensusParams.EvidenceParams.MaxAge
	if evidenceAge > maxAge {
		return fmt.Errorf("Evidence from height %d is too old. Min height is %d",
			evidence.Height(), height-maxAge)
	}

	if err := evidence.Verify(s.ChainID); err != nil {
		return err
	}

	valset, err := stateDB.LoadValidators(evidence.Height())
	if err != nil {
		// TODO: if err is just that we cant find it cuz we pruned, ignore.
		// TODO: if its actually bad evidence, punish peer
		return err
	}

	// The address must have been an active validator at the height
	ev := evidence
	height, addr, idx := ev.Height(), ev.Address(), ev.Index()
	valIdx, val := valset.GetByAddress(addr)
	if val == nil {
		return fmt.Errorf("Address %X was not a validator at height %d", addr, height)
	} else if idx != valIdx {
		return fmt.Errorf("Address %X was validator %d at height %d, not %d", addr, valIdx, height, idx)
	}

	return nil
}
