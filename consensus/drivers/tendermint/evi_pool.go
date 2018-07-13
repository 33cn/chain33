package tendermint

import (
	"fmt"
	"sync"

	"encoding/json"

	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
)

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
	evidenceChan chan types.Evidence
}

func NewEvidencePool(stateDB *CSStateDB, state State, evidenceStore *EvidenceStore) *EvidencePool {
	evpool := &EvidencePool{
		stateDB:       stateDB,
		state:         state,
		evidenceStore: evidenceStore,
		evidenceChan:  make(chan types.Evidence),
	}
	return evpool
}

// EvidenceChan returns an unbuffered channel on which new evidence can be received.
func (evpool *EvidencePool) EvidenceChan() <-chan types.Evidence {
	return evpool.evidenceChan
}

// PriorityEvidence returns the priority evidence.
func (evpool *EvidencePool) PriorityEvidence() []types.Evidence {
	return evpool.evidenceStore.PriorityEvidence()
}

// PendingEvidence returns all uncommitted evidence.
func (evpool *EvidencePool) PendingEvidence() []types.Evidence {
	return evpool.evidenceStore.PendingEvidence()
}

// State returns the current state of the evpool.
func (evpool *EvidencePool) State() State {
	evpool.mtx.Lock()
	defer evpool.mtx.Unlock()
	return evpool.state
}

// Update loads the latest
func (evpool *EvidencePool) Update(block *types.Block) {
	evpool.mtx.Lock()
	defer evpool.mtx.Unlock()

	state := evpool.stateDB.LoadState()
	if state.LastBlockHeight != block.Height {
		panic(fmt.Sprintf("EvidencePool.Update: loaded state with height %d when block.Height=%d", state.LastBlockHeight, block.Height))
	}
	evpool.state = state

	// NOTE: shouldn't need the mutex
	evpool.MarkEvidenceAsCommitted(block.Evidence.Evidence)
}

// AddEvidence checks the evidence is valid and adds it to the pool.
// Blocks on the EvidenceChan.
func (evpool *EvidencePool) AddEvidence(evidence types.Evidence) (err error) {

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
func (evpool *EvidencePool) MarkEvidenceAsCommitted(evidence types.EvidenceEnvelopeList) {
	for _, ev := range evidence {
		if v, ok := types.EvidenceType2Obj[ev.Kind]; ok {
			tmp := v.(types.Evidence).Copy()
			err := json.Unmarshal(*ev.Data, &tmp)
			if err != nil {
				tendermintlog.Error("MarkEvidenceAsCommitted envelop unmarshal failed", "error", err)
				return
			}
			evpool.evidenceStore.MarkEvidenceAsCommitted(tmp)
		}
	}
}

// VerifyEvidence verifies the evidence fully by checking it is internally
// consistent and sufficiently recent.
func VerifyEvidence(stateDB *CSStateDB, s State, evidence types.Evidence) error {
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
