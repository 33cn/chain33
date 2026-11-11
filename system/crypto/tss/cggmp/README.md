# chain33 CGGMP TSS wrapper

This package adds **CGGMP** (Canetti et al., threshold ECDSA) support to
`system/crypto/tss`, in parallel with the existing GG18 wrapper in
`system/crypto/tss/gg18`. It wraps the CGGMP implementation shipped inside
`github.com/getamis/alice` (`crypto/tss/ecdsa/cggmp`) the same way `gg18` wraps
alice's GG18.

`getamis/alice` is a monorepo containing GG18 / CCLST / CGGMP (ECDSA) and FROST
(EdDSA). Upgrading alice to v1.0.7 (a security-fix release) keeps GG18 behaviour
unchanged and makes the CGGMP code available.

## Session / key-material independence from GG18

GG18 and CGGMP sessions never collide:

- protocol tags are disjoint: `/gg18/{dkg,sign,reshare}` vs
  `/cggmp/{dkg,refresh,sign,ppk}`;
- each wrapper has its own private per-process session registry
  (`gg18/session.go` vs `cggmp/session.go`);
- each wrapper registers its own message handlers via `tss.RegisterMsgHandler`.

The shared transport infrastructure is reused as-is: `tss.NewReadyPeerManager`,
`tss.NewListener`, the queue-based `MessageWrapper` fan-out and the per-process
session/message-buffer pattern.

## CGGMP shape vs GG18

alice's CGGMP is not a drop-in replacement for GG18 at the crypto level:

| | GG18 | CGGMP |
|---|---|---|
| after DKG | sign directly | an extra **refresh** is required |
| partial public keys | n/a | exchanged right after DKG (`/cggmp/ppk`) |
| Paillier / Pedersen | fresh Paillier generated in-round per sign | provisioned once by the refresh phase and reused |
| sign threshold | implicit (= signer count) | explicit argument |
| DKG result | `*tss.DKGResult` (proto) | `*DKGResult` (JSON, adds `Rid` + `PartialPubKeys`) |

Therefore the wrapper does **not** hide the phases behind one shared "DKG+sign"
API — that would be a false abstraction. Instead the two packages expose the same
*shapes* and the genuinely common pieces are converged (see below).

## Public API

```
cggmp.ProcessDKG(peers, threshold, rank, sessionID, opts...)  (*DKGResult, error)
cggmp.ProcessRefresh(peers, threshold, dkgRes, sessionID, opts...)  (*RefreshResult, error)
cggmp.ProcessSign(peers, threshold, msg, dkgRes, refreshRes, sessionID, opts...)  (*sign.Result, error)
cggmp.ToBtcecSignature(result)  (*ecdsa.Signature, error)
```

Phase sequence for one key:

```
ProcessDKG            -> persist *DKGResult      (group pubkey + own share + bks + rid + partial pubkeys)
ProcessRefresh        -> persist *RefreshResult  (refreshed share + own Paillier primes + per-peer Pedersen/ppk/y)
ProcessSign (repeat)  -> sign.Result {R,S}       -> ToBtcecSignature
```

`peers` always includes the local node id and must be identical across all nodes
of a phase. `sessionID` must be identical across the nodes of one phase but can
differ between phases. `sign.Result` is alice's type; convert with
`cggmp.ToBtcecSignature`.

## What is unified vs what stays separate (interface decision)

Converged into shared code (single implementation, no duplication):

- **scalar → btcec signature**: `tss.BuildBtcecSignature(R, S)` and
  `tss.BigIntToModNScalar` in `system/crypto/tss/sig.go`. Both
  `gg18.AliceToBtcecSignature` and `cggmp.ToBtcecSignature` delegate to it
  (`gg18/api.go` was refactored, API unchanged).
- **entry-point shapes**: `ProcessDKG(peers, threshold, rank, sessionID)`,
  a `ProcessReshare/ProcessRefresh`, a `ProcessSign`, and a `ToBtcecSignature`
  converter, with `WithTimeout` options — so a caller that learns one package can
  read the other.
- **peer/session/message plumbing** and the **protocol-tag + handler**
  registration pattern.

Kept separate (per algorithm, by design — merging them would be a forced
abstraction):

- **DKG result types**: GG18 `*tss.DKGResult` (proto, in `tss.proto`) vs CGGMP
  `*cggmp.DKGResult` / `*cggmp.RefreshResult` (plain Go, JSON-serialisable).
  CGGMP simply has more key material (rid, partial pub keys, Paillier primes,
  Pedersen params) that GG18 never needs.
- **sign signature**: GG18 `ProcessSign(peers, msg, dkg, session)` vs CGGMP
  `ProcessSign(peers, threshold, msg, dkg, refresh, session)` — the extra
  `threshold` and `refresh` args are inherent to alice's CGGMP.
- **phase count**: GG18 `DKG -> Sign`; CGGMP `DKG(+ppk) -> Refresh -> Sign`.

### How a caller switches GG18 ↔ CGGMP

The change surface for a caller (e.g. the plugin `tss.go`) is limited to:

1. import `cggmp` instead of `gg18` (or both, and branch);
2. DKG: same 4-arg call, but persist the returned `*cggmp.DKGResult`;
3. add one `cggmp.ProcessRefresh` call after DKG and persist its `RefreshResult`;
4. sign: `cggmp.ProcessSign(signers, threshold, msg, dkgResult, refreshResult,
   session)` instead of `gg18.ProcessSign(signers, msg, dkgResult, session)`;
5. convert the result with `cggmp.ToBtcecSignature` (same idea as
   `gg18.AliceToBtcecSignature`).

The group public key (`dkgResult.PubX/PubY`) and the resulting btcec signature
are in the same format as GG18, so address derivation and signature verification
downstream are unchanged.

## Notes / caveats

- **`rank` is the Birkhoff rank, not a node index.** `ProcessDKG`'s `rank` argument is
  the rank of the *local* node's Birkhoff parameter (the derivative order of the shared
  polynomial evaluated at its x-coordinate), **not** a 0/1/2/3 participant number:

  - alice requires `rank + 1 < threshold` (`utils.EnsureRank`), i.e. `rank ∈ [0, threshold-2]`;
  - several participants may share the same rank — they are told apart by their
    (random) x-coordinate, so a valid set for a threshold-`t` sign is any set of
    participants whose sorted ranks satisfy `rank_i <= i` for `i = 0..t-1`
    (`tss.GetValidPeerCombination` picks one);
  - the p2p integration test uses exactly that: 4 nodes with ranks `{0, 1, 1, 1}` and
    threshold 3 for DKG/refresh, then the 3-node subset `{0, 1, 1}` for signing.

  So a plain counter `0,1,2,3` cannot be used for 4 nodes: the highest ranks are
  rejected outright (`rank+1 < threshold`, with `threshold <= 4` here), and the same
  constraint is what leaves rank-0 participants mandatory in a signer subset — the
  most common mistake when switching a caller from GG18, where the equivalent
  parameters are indexed differently.
- **ppk attribution is by authenticated peer id.** `PartialPublicKey.Sender` is
  attacker-controlled and is never used to attribute or validate a partial public key:
  collected points are keyed by `MessageWrapper.PeerID`, the peer id authenticated by the
  transport (`p2p/dht/protocol/tss` fills it from `stream.Conn().RemotePeer()`). A mismatch
  between `Sender` and the authenticated id is logged and otherwise ignored, so one peer
  can neither satisfy the quorum alone nor overwrite another participant's `g^{share}`.
- **The ppk exchange is verified against the DKG group public key.** Once all partial
  public keys are collected, `ProcessDKG` (and `ProcessRefresh` on the persisted material)
  reconstructs the group public key from them through the Birkhoff parameters
  (`birkhoffinterpolation.BkParameters.ValidatePublicKey`, the same check alice runs at the
  end of DKG/refresh). A participant that broadcast a bad `g^{share_i}` is rejected at the
  ppk stage with a `ppk check` error instead of failing later inside the refresh rounds.
  alice's DKG discards its Feldman commitments, so this reconstruction is the only local
  check available at that point.
- **Per-phase timeouts.** The default phase timeout is 30s for DKG, ppk and sign, but
  5 minutes for refresh: refresh generates a fresh 2048-bit Paillier key (safe primes) on
  every node, which alone takes ~23s for 4 nodes on localhost. `WithTimeout` overrides the
  default for all phases (a non-positive value means no timeout).
- **ssid must be session-shared.** The refresh/sign ZK challenges are derived via
  `cggmp.ComputeZKSsid(ssid, peerBk, N)` on both prover and verifier, which only
  match when `ssid` is identical across participants. `cggmp.computeSSID`
  therefore binds the *shared* session id and the DKG `rid` (alice's own cggmp
  tests likewise pass a single shared nonce). Do not derive ssid from the local
  peer's own Birkhoff parameter.
- **Threshold signing with a subset**: pass the actual signer subset as `peers` and the
  matching threshold; `len(peers)` must equal `threshold` (enforced by `ProcessSign`). Only
  those participants' bks / partial pub keys / Pedersen params are fed to alice (the sign
  core iterates the passed maps, so non-participant entries would break the Birkhoff
  coefficient computation). `tss.GetValidPeerCombination` returns exactly `threshold` peers
  sorted by rank that always include the local node — a bare "take the first `threshold`
  connected peers" is wrong: the local id must be present (alice indexes by `SelfID`), and a
  subset whose sorted ranks violate `rank_i <= i` fails inside the sign core.
- **One `Process*` per session id.** A session id names one round, so the wrapper rejects a
  second concurrent call with the same `protocol|sessionID` instead of letting the two
  attempts reset each other's state. Retries after a failure are sequential and reuse the id.
- **Messages are accepted only from participants.** Every CGGMP core wraps its message main
  in alice's `EchoMsgMain`, which attributes a message by the `Id` in its body (honest nodes
  echo it, and the core then treats that echoed id as the sender) rather than by the
  transport peer id. The session layer therefore checks the transport-authenticated peer id
  against the round's participant set, both for live sessions and when flushing messages
  buffered before the local session registered. A connected peer outside the round can
  neither contribute nor get an honest node to echo a forged id. (GG18 is unaffected: alice's
  `MsgMain` compares the sender with `msg.Id` itself.)
- The multi-process run is exercised by `cggmp_integration_test.go` (`TestCGGMP4Node`, four
  real chain33 child nodes over the p2p transport, skipped by `-short`); CI runs it with
  `-race` in the `test-cggmp-integration` job. `e2e_test.go` runs the complete
  DKG → ppk → refresh → sign crypto in-process (three parties) and round-trips
  every persisted result through JSON, proving the data flow and that the
  signature verifies against the DKG group key.
