// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	//	"time"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	//"github.com/coreos/etcd/internal/raftsnap"
	"github.com/coreos/etcd/pkg/pbutil"
	"github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/wal"
	"github.com/coreos/etcd/wal/walpb"
	// chain33Types "gitlab.33.cn/chain33/chain33/types"
	"github.com/golang/protobuf/proto"
	//"github.com/coreos/etcd/snapshot"
)

func main() {
	//snapfile := flag.String("start-snap", "", "The base name of snap file to start dumping")
	waldir := flag.String("start-snapwal", "", "The base name of snapwal dir to start dumping")
	index := flag.Uint64("start-index", 0, "The index to start dumping")
	flag.Parse()

	//if len(flag.Args()) != 1 {
	//	log.Fatalf("Must provide data-dir argument (got %+v)", flag.Args())
	//}
	//dataDir := flag.Args()[0]

	if *waldir != "" && *index != 0 {
		log.Fatal("start-snapwal and start-index flags cannot be used together.")
	}

	var (
		walsnap walpb.Snapshot
		//snapshot *raftpb.Snapshot
		err error
	)

	isIndex := *index != 0

	//if isIndex {
	//	fmt.Printf("Start dumping log entries from index %d.\n", *index)
	//	walsnap.Index = *index
	//} else {
	//	if *snapfile == "" {
	//		ss := raftsnap.New(snapDir(dataDir))
	//		snapshot, err = ss.Load()
	//	} else {
	//		snapshot, err = raftsnap.Read(filepath.Join(snapDir(dataDir), *snapfile))
	//	}
	//
	//	switch err {
	//	case nil:
	//		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	//	    nodes := genIDSlice(snapshot.Metadata.ConfState.Nodes)
	//		fmt.Printf("Snapshot:\nterm=%d index=%d nodes=%s\n",
	//			walsnap.Term, walsnap.Index, nodes)
	//	case raftsnap.ErrNoSnapshot:
	//		fmt.Printf("Snapshot:\nempty\n")
	//	default:
	//		log.Fatalf("Failed loading snapshot: %v", err)
	//	}
	//	fmt.Println("Start dupmping log entries from snapshot.")
	//}

	w, err := wal.OpenForRead(*waldir, walsnap)
	if err != nil {
		log.Fatalf("Failed opening WAL: %v", err)
	}
	wmetadata, state, ents, err := w.ReadAll()
	w.Close()
	if err != nil && (!isIndex || err != wal.ErrSnapshotNotFound) {
		log.Fatalf("Failed reading WAL: %v", err)
	}
	id, cid := parseWALMetadata(wmetadata)
	vid := types.ID(state.Vote)
	fmt.Printf("WAL metadata:\nnodeID=%s clusterID=%s term=%d commitIndex=%d vote=%s\n",
		id, cid, state.Term, state.Commit, vid)

	fmt.Printf("WAL entries:\n")
	fmt.Printf("lastIndex=%d\n", ents[len(ents)-1].Index)
	fmt.Printf("%4s\t%10s\ttype\tdata\n", "term", "index")
	for _, e := range ents {
		msg := fmt.Sprintf("%4d\t%10d", e.Term, e.Index)
		switch e.Type {
		case raftpb.EntryNormal:
			msg = fmt.Sprintf("%s\tnorm", msg)
			if len(e.Data) == 0 {
				break
			}
			// 解码
			block := &Block{}
			if err := proto.Unmarshal(e.Data, block); err != nil {
				log.Printf("failed to unmarshal: %v", err)
				break
			}
			msg = fmt.Sprintf("%s\t BlockHeight:%d", msg, block.Height)
		case raftpb.EntryConfChange:
			msg = fmt.Sprintf("%s\tconf", msg)
			var r raftpb.ConfChange
			if err := r.Unmarshal(e.Data); err != nil {
				msg = fmt.Sprintf("%s\t???", msg)
			} else {
				msg = fmt.Sprintf("%s\tmethod=%s id=%s", msg, r.Type, types.ID(r.NodeID))
			}
		}
		fmt.Println(msg)
	}
}

//func walDir(dataDir string) string { return filepath.Join(dataDir, "", "wal") }

//func snapDir(dataDir string) string { return filepath.Join(dataDir, "", "snap") }

func parseWALMetadata(b []byte) (id, cid types.ID) {
	var metadata etcdserverpb.Metadata
	pbutil.MustUnmarshal(&metadata, b)
	id = types.ID(metadata.NodeID)
	cid = types.ID(metadata.ClusterID)
	return id, cid
}

//func genIDSlice(a []uint64) []types.ID {
//	ids := make([]types.ID, len(a))
//	for i, id := range a {
//		ids[i] = types.ID(id)
//	}
//	return ids
//}

// excerpt replaces middle part with ellipsis and returns a double-quoted
// string safely escaped with Go syntax.
//func excerpt(str string, pre, suf int) string {
//	if pre+suf > len(str) {
//		return fmt.Sprintf("%q", str)
//	}
//	return fmt.Sprintf("%q...%q", str[:pre], str[len(str)-suf:])
//}

type Block struct {
	Version    int64  `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	ParentHash []byte `protobuf:"bytes,2,opt,name=parentHash,proto3" json:"parentHash,omitempty"`
	TxHash     []byte `protobuf:"bytes,3,opt,name=txHash,proto3" json:"txHash,omitempty"`
	StateHash  []byte `protobuf:"bytes,4,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	Height     int64  `protobuf:"varint,5,opt,name=height" json:"height,omitempty"`
	BlockTime  int64  `protobuf:"varint,6,opt,name=blockTime" json:"blockTime,omitempty"`
	//Signature  *Signature     `protobuf:"bytes,8,opt,name=signature" json:"signature,omitempty"`
	//Txs        []*Transaction `protobuf:"bytes,7,rep,name=txs" json:"txs,omitempty"`
}

func (m *Block) Reset()         { *m = Block{} }
func (m *Block) String() string { return proto.CompactTextString(m) }
func (*Block) ProtoMessage()    {}
